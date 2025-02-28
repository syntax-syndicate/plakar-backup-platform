/*
 * Copyright (c) 2021 Gilles Chehade <gilles@poolp.org>
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package cleanup

import (
	"bytes"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/caching"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/resources"
	"github.com/PlakarKorp/plakar/snapshot"
)

func init() {
	subcommands.Register("cleanup", parse_cmd_cleanup)
}

func parse_cmd_cleanup(ctx *appcontext.AppContext, repo *repository.Repository, args []string) (subcommands.Subcommand, error) {
	flags := flag.NewFlagSet("cleanup", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s\n", flags.Name())
	}
	flags.Parse(args)

	return &Cleanup{
		RepositoryLocation: repo.Location(),
		RepositorySecret:   ctx.GetSecret(),
	}, nil
}

type Cleanup struct {
	RepositoryLocation string
	RepositorySecret   []byte

	repository     *repository.Repository
	maintainanceID objects.MAC
}

func (cmd *Cleanup) Name() string {
	return "cleanup"
}

// Builds the local cache of snapshot -> packfiles
func (cmd *Cleanup) updateCache(ctx *appcontext.AppContext, cache *caching.MaintainanceCache) error {
	for snapshotID := range cmd.repository.ListSnapshots() {
		snapshot, err := snapshot.Load(cmd.repository, snapshotID)
		if err != nil {
			return err
		}

		ok, err := cache.HasSnapshot(snapshotID)
		if err != nil {
			return err
		}

		if ok {
			continue
		}

		fmt.Fprintf(ctx.Stdout, "updating cache with snapshot: %x\n", snapshotID)
		iter, err := snapshot.ListPackfiles()
		if err != nil {
			return err
		}

		for packfile, err := range iter {
			if err != nil {
				return err
			}

			if err := cache.PutPackfile(snapshotID, packfile); err != nil {
				return err
			}
		}

		cache.PutSnapshot(snapshotID, nil)
		snapshot.Close()
	}

	// While ListSnapshots doesn't return deleted snapshots, we still need to
	// go over them to remove previously added one to our local cache.
	for snapshotID := range cmd.repository.ListDeletedSnapShots() {
		ok, err := cache.HasSnapshot(snapshotID)
		if err != nil {
			return err
		}

		if !ok {
			continue
		}

		fmt.Fprintf(ctx.Stdout, "deleting %x from local cache\n", snapshotID)
		cache.DeleletePackfiles(snapshotID)
		cache.DeleteSnapshot(snapshotID)
	}

	return nil
}

func (cmd *Cleanup) colourPass(ctx *appcontext.AppContext, cache *caching.MaintainanceCache) error {
	var packfiles map[objects.MAC]struct{} = make(map[objects.MAC]struct{})

	for packfileMAC := range cmd.repository.ListPackfiles() {
		if !cache.HasPackfile(packfileMAC) {
			packfiles[packfileMAC] = struct{}{}
		}
	}

	sc, err := cmd.repository.AppContext().GetCache().Scan(cmd.maintainanceID)
	if err != nil {
		return err
	}

	// First pass, coloring, we just flag those packfiles as being selected for deletion.
	// For now we keep the same serial so that those delete gets merged in.
	// Once we do the real deletion we will rebuild the aggregated view
	// excluding those ressources alltogether.
	deltaState := cmd.repository.NewStateDelta(sc)

	coloredPackfiles := 0
	for packfile := range packfiles {
		has, err := cmd.repository.HasDeletedPackfile(packfile)
		if err != nil {
			return err
		}

		if !has {
			coloredPackfiles++
			if err := deltaState.DeleteResource(resources.RT_PACKFILE, packfile); err != nil {
				return err
			}
		}
	}

	fmt.Fprintf(ctx.Stdout, "colour: Coloured %d packfiles for deletion\n", coloredPackfiles)

	buf := &bytes.Buffer{}
	if err := deltaState.SerializeToStream(buf); err != nil {
		return err
	}

	if err := cmd.repository.PutState(cmd.maintainanceID, buf); err != nil {
		return err
	}

	return nil
}

func (cmd *Cleanup) sweepPass(ctx *appcontext.AppContext, cache *caching.MaintainanceCache) error {
	// These need to be configurable per repo, but we don't have a mechanism yet (comes in a PR soon!)
	cutoff := time.Now().AddDate(0, 0, -30)
	doDeletion := false

	// First go over all the packfiles coloured by first pass.
	packfileremoved := 0
	blobRemoved := 0
	for packfileMAC, deletionTime := range cmd.repository.ListDeletedPackfiles() {
		if deletionTime.After(cutoff) {
			continue
		}

		// At this point we have to re-check if our packfile is really unused,
		// because we could have had a concurrent backup with the coloring
		// phase.
		if cache.HasPackfile(packfileMAC) {
			fmt.Fprintf(ctx.Stderr, "cleanup: Concurrent backup used %x, uncolouring the packfile.\n", packfileMAC)
			cmd.repository.RemoveDeletedPackfile(packfileMAC)
			continue
		}

		// First thing we remove the packfile entry from our state, this means
		// that now effectively all of its blob are unreachable
		if err := cmd.repository.RemovePackfile(packfileMAC); err != nil {
			fmt.Fprintf(ctx.Stderr, "cleanup: Failed to remove packfile %s from state\n", packfileMAC)
		} else {
			cmd.repository.RemoveDeletedPackfile(packfileMAC)
		}

		packfileremoved++
	}

	// Second garbage collect dangling blobs in our state. This is the blobs we
	// just orphaned plus potential orphan blobs from aborted backups etc.
	toDelete := map[objects.MAC]struct{}{}
	for blob, err := range cmd.repository.ListOrphanBlobs() {
		if err != nil {
			fmt.Fprintf(ctx.Stderr, "cleanup: Failed to fetch orphaned blob\n")
			continue
		}

		blobRemoved++
		toDelete[blob.Location.Packfile] = struct{}{}
		if err := cmd.repository.RemoveBlob(blob.Type, blob.Blob, blob.Location.Packfile); err != nil {
			// No hurt in this failing, we just have cruft left around, but they are unreachable anyway.
			fmt.Fprintf(ctx.Stderr, "cleanup: garbage orphaned blobs pass failed to remove blob %x, type %s\n", blob.Blob, blob.Type)
		}
	}

	fmt.Fprintf(ctx.Stdout, "cleanup: %d blobs and %d packfiles were removed\n", blobRemoved, packfileremoved)
	if err := cmd.repository.PutCurrentState(); err != nil {
		return err
	}

	if doDeletion {
		for packfileMAC := range toDelete {
			if err := cmd.repository.DeletePackfile(packfileMAC); err != nil {
				fmt.Fprintf(ctx.Stderr, "cleanup: Sweep pass failed to delete packfile %x, skipping it\n", packfileMAC)
			}
		}
	}

	return nil
}

func (cmd *Cleanup) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	// the cleanup algorithm is a bit tricky and needs to be done in the correct sequence,
	// here's what it has to do:
	//
	// 1. fetch all packfiles in the repository
	// 2. iterate over snapshots
	// 3. for each snapshot, resolve the packfiles it references
	// 4. remove the packfiles from the list of packfiles
	// 5. remaining packfiles should be marked as deleted in the state
	// 6. remove the packfile in repository once it's flagged as deleted AND all snapshots have been `snapshot.Check`-ed
	// 7. rebuild a new aggregate state with a new serial without the deleted packfiles

	cmd.repository = repo
	// This random id generation for non snapshot state should probably be encapsulated somewhere.
	n, err := rand.Read(cmd.maintainanceID[:])
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "cleanup: Failed to read from random source.%s\n", err)
		return 1, err
	}
	if n != len(cmd.maintainanceID) {
		fmt.Fprintf(ctx.Stderr, "cleanup: Failed to read from random source.%s\n", err)
		return 1, io.ErrShortWrite
	}

	done, err := cmd.Lock()
	if err != nil {
		return 1, err
	}
	defer cmd.Unlock(done)

	cache, err := repo.AppContext().GetCache().Maintainance(repo.Configuration().RepositoryID)
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "cleanup: Failed to open local cache %s\n", err)
		return 1, err
	}

	if err := cmd.updateCache(ctx, cache); err != nil {
		fmt.Fprintf(ctx.Stderr, "cleanup: Failed to update local cache %s\n", err)
		return 1, err
	}

	if err := cmd.colourPass(ctx, cache); err != nil {
		fmt.Fprintf(ctx.Stderr, "cleanup: Colouring pass failed %s\n", err)
		return 1, err
	}

	if err := cmd.sweepPass(ctx, cache); err != nil {
		fmt.Fprintf(ctx.Stderr, "cleanup: Sweep pass failed %s\n", err)
		return 1, err
	}

	return 0, nil
}

func (cmd *Cleanup) Lock() (chan bool, error) {
	lock := repository.NewExclusiveLock(cmd.repository.AppContext().Hostname)

	buffer := &bytes.Buffer{}
	err := lock.SerializeToStream(buffer)
	if err != nil {
		return nil, err
	}

	err = cmd.repository.PutLock(cmd.maintainanceID, buffer)
	if err != nil {
		return nil, err
	}

	// We installed the lock, now let's see if there is a conflicting exclusive lock or not.
	locksID, err := cmd.repository.GetLocks()
	if err != nil {
		// We still need to delete it, and we need to do so manually.
		cmd.repository.DeleteLock(cmd.maintainanceID)
		return nil, err
	}

	for _, lockID := range locksID {
		if lockID == cmd.maintainanceID {
			continue
		}

		version, rd, err := cmd.repository.GetLock(lockID)
		if err != nil {
			cmd.repository.DeleteLock(cmd.maintainanceID)
			return nil, err
		}

		lock, err := repository.NewLockFromStream(version, rd)
		if err != nil {
			cmd.repository.DeleteLock(cmd.maintainanceID)
			return nil, err
		}

		/* Kick out stale locks */
		if lock.IsStale() {
			err := cmd.repository.DeleteLock(lockID)
			if err != nil {
				cmd.repository.DeleteLock(cmd.maintainanceID)
				return nil, err
			}
		}

		// There is a lock in place, we need to abort.
		err = cmd.repository.DeleteLock(cmd.maintainanceID)
		if err != nil {
			return nil, err
		}

		return nil, fmt.Errorf("Can't take exclusive lock, repository is already locked")
	}

	// The following bit is a "ping" mechanism, Lock() is a bit badly named at this point,
	// we are just refreshing the existing lock so that the watchdog doesn't removes us.
	lockDone := make(chan bool)
	go func() {
		for {
			select {
			case <-lockDone:
				return
			case <-time.After(repository.LOCK_REFRESH_RATE):
				lock := repository.NewExclusiveLock(cmd.repository.AppContext().Hostname)

				buffer := &bytes.Buffer{}

				// We ignore errors here on purpose, it's tough to handle them
				// correctly, and if they happen we will be ripped by the
				// watchdog anyway.
				lock.SerializeToStream(buffer)
				cmd.repository.PutLock(cmd.maintainanceID, buffer)
			}
		}
	}()

	return lockDone, nil
}

func (cmd *Cleanup) Unlock(ping chan bool) error {
	close(ping)
	return cmd.repository.DeleteLock(cmd.maintainanceID)
}
