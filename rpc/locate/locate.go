package locate

import (
	"fmt"
	"os"
	"path"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/utils"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/snapshot"
)

type Locate struct {
	RepositoryLocation string
	RepositorySecret   []byte

	Snapshot string
	Patterns []string
}

func (cmd *Locate) Name() string {
	return "locate"
}

func (cmd *Locate) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	var snapshotIDs []objects.Checksum
	if cmd.Snapshot != "" {
		snapshotIDs = utils.LookupSnapshotByPrefix(repo, cmd.Snapshot)
	} else {
		var err error
		snapshotIDs, err = repo.GetSnapshots()
		if err != nil {
			ctx.GetLogger().Error("...")
			return 1, err
		}
	}

	for _, snapshotID := range snapshotIDs {
		snap, err := snapshot.Load(repo, snapshotID)
		if err != nil {
			ctx.GetLogger().Error("locate: could not get snapshot: %s", err)
			return 1, err
		}

		fs, err := snap.Filesystem()
		if err != nil {
			ctx.GetLogger().Error("locate: could not get filesystem: %s", err)
			snap.Close()
			return 1, err
		}
		for pathname, err := range fs.Pathnames() {
			if err != nil {
				ctx.GetLogger().Error("locate: could not get pathname: %s", err)
				snap.Close()
				return 1, err
			}

			for _, pattern := range cmd.Patterns {
				matched := false
				if path.Base(pathname) == pattern {
					matched = true
				}
				if !matched {
					matched, err := path.Match(pattern, path.Base(pathname))
					if err != nil {
						ctx.GetLogger().Error("locate: could not match pattern: %s", err)
						snap.Close()
						return 1, err
					}
					if !matched {
						continue
					}
				}
				fmt.Fprintf(os.Stdout, "%x:%s\n", snap.Header.Identifier[0:4], pathname)
			}
		}
		snap.Close()
	}
	return 0, nil
}
