package handlers

import (
	"encoding/hex"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os/user"
	"path"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/utils"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/snapshot/vfs"
	"github.com/dustin/go-humanize"
)

type Ls struct {
	RepositoryLocation string
	RepositorySecret   []byte

	Recursive   bool
	Tag         string
	DisplayUUID bool
	Paths       []string
}

func (cmd *Ls) Name() string {
	return "ls"
}

func (cmd *Ls) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	if len(cmd.Paths) == 0 {
		list_snapshots(ctx, repo, cmd.DisplayUUID, cmd.Tag)
		return 0, nil
	}

	if err := list_snapshot(ctx, repo, cmd.Paths[0], cmd.Recursive); err != nil {
		log.Println("error:", err)
		return 1, err
	}
	return 0, nil
}

func list_snapshots(ctx *appcontext.AppContext, repo *repository.Repository, useUuid bool, tag string) {
	metadatas, err := utils.GetHeaders(repo, nil)
	if err != nil {
		log.Fatalf("%s: could not fetch snapshots list", flag.CommandLine.Name())
	}

	for _, metadata := range metadatas {
		if tag != "" {
			found := false
			for _, t := range metadata.Tags {
				if tag == t {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if !useUuid {
			fmt.Fprintf(ctx.Stdout, "%s %10s%10s%10s %s\n",
				metadata.Timestamp.UTC().Format(time.RFC3339),
				hex.EncodeToString(metadata.GetIndexShortID()),
				humanize.Bytes(metadata.Summary.Directory.Size+metadata.Summary.Below.Size),
				metadata.Duration.Round(time.Second),
				metadata.Importer.Directory)
		} else {
			indexID := metadata.GetIndexID()
			fmt.Fprintf(ctx.Stdout, "%s %3s%10s%10s %s\n",
				metadata.Timestamp.UTC().Format(time.RFC3339),
				hex.EncodeToString(indexID[:]),
				humanize.Bytes(metadata.Summary.Directory.Size+metadata.Summary.Below.Size),
				metadata.Duration.Round(time.Second),
				metadata.Importer.Directory)
		}
	}
}

func list_snapshot(ctx *appcontext.AppContext, repo *repository.Repository, snapshotPath string, recursive bool) error {
	prefix, pathname := utils.ParseSnapshotID(snapshotPath)
	pathname = path.Clean(pathname)

	snap, err := utils.OpenSnapshotByPrefix(repo, prefix)
	if err != nil {
		log.Fatalf("%s: could not fetch snapshot: %s", flag.CommandLine.Name(), err)
	}
	defer snap.Close()

	pvfs, err := snap.Filesystem()
	if err != nil {
		log.Fatal(err)
	}

	return pvfs.WalkDir(pathname, func(path string, d *vfs.Entry, err error) error {
		if err != nil {
			log.Println("error at", path, ":", err)
			return err
		}
		if path == pathname {
			return nil
		}

		sb, err := d.Info()
		if err != nil {
			return err
		}

		var username, groupname string
		if finfo, ok := sb.Sys().(objects.FileInfo); ok {
			pwUserLookup, err := user.LookupId(fmt.Sprintf("%d", finfo.Uid()))
			username = fmt.Sprintf("%d", finfo.Uid())
			if err == nil {
				username = pwUserLookup.Username
			}

			grGroupLookup, err := user.LookupGroupId(fmt.Sprintf("%d", finfo.Gid()))
			groupname = fmt.Sprintf("%d", finfo.Gid())
			if err == nil {
				groupname = grGroupLookup.Name
			}
		}

		entryname := path
		if !recursive {
			entryname = d.Name()
		}

		fmt.Fprintf(ctx.Stdout, "%s %s % 8s % 8s % 8s %s\n",
			sb.ModTime().UTC().Format(time.RFC3339),
			sb.Mode(),
			username,
			groupname,
			humanize.Bytes(uint64(sb.Size())),
			entryname)

		if !recursive && pathname != path && sb.IsDir() {
			return fs.SkipDir
		}
		return nil
	})
}
