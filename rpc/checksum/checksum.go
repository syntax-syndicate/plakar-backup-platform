package checksum

import (
	"fmt"
	"io"
	"path"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/utils"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/snapshot"
	"github.com/PlakarKorp/plakar/snapshot/vfs"
)

type Checksum struct {
	RepositoryLocation string
	RepositorySecret   []byte

	Fast    bool
	Targets []string
}

func (cmd *Checksum) Name() string {
	return "checksum"
}

func (cmd *Checksum) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	snapshots, err := utils.GetSnapshots(repo, cmd.Targets)
	if err != nil {
		ctx.GetLogger().Error("checksum: could not obtain snapshots list: %s", err)
		return 1, err
	}

	errors := 0
	for offset, snap := range snapshots {
		defer snap.Close()

		fs, err := snap.Filesystem()
		if err != nil {
			continue
		}

		_, pathname := utils.ParseSnapshotID(cmd.Targets[offset])
		if pathname == "" {
			ctx.GetLogger().Error("checksum: missing filename for snapshot %x", snap.Header.GetIndexShortID())
			errors++
			continue
		}

		displayChecksums(ctx, fs, repo, snap, pathname, cmd.Fast)
	}

	return 0, nil
}

func displayChecksums(ctx *appcontext.AppContext, fs *vfs.Filesystem, repo *repository.Repository, snap *snapshot.Snapshot, pathname string, fastcheck bool) error {
	fsinfo, err := fs.GetEntry(pathname)
	if err != nil {
		return err
	}

	if fsinfo.Stat().Mode().IsDir() {
		iter, err := fsinfo.Getdents(fs)
		if err != nil {
			return err
		}
		for child := range iter {
			if err := displayChecksums(ctx, fs, repo, snap, path.Join(pathname, child.Stat().Name()), fastcheck); err != nil {
				return err
			}
		}
		return nil
	}
	if !fsinfo.Stat().Mode().IsRegular() {
		return nil
	}

	object, err := snap.LookupObject(fsinfo.Object.Checksum)
	if err != nil {
		return err
	}

	checksum := object.Checksum
	if !fastcheck {
		rd, err := snap.NewReader(pathname)
		if err != nil {
			return err
		}
		defer rd.Close()

		hasher := repo.Hasher()
		if _, err := io.Copy(hasher, rd); err != nil {
			return err
		}
	}
	fmt.Fprintf(ctx.Stdout, "SHA256 (%s) = %x\n", pathname, checksum)
	return nil
}
