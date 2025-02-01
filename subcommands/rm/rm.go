package rm

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/utils"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/snapshot"
	"github.com/dustin/go-humanize"
)

type Rm struct {
	RepositoryLocation string
	RepositorySecret   []byte

	Tag        string
	BeforeDate time.Time
	Prefixes   []string
}

func (cmd *Rm) Name() string {
	return "rm"
}

func (cmd *Rm) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	var snapshots []*snapshot.Snapshot
	if !cmd.BeforeDate.IsZero() || cmd.Tag != "" {
		if len(cmd.Prefixes) != 0 {
			tmp, err := utils.GetSnapshots(repo, cmd.Prefixes)
			if err != nil {
				log.Fatal(err)
			}
			snapshots = tmp
		} else {
			tmp, err := utils.GetSnapshots(repo, nil)
			if err != nil {
				log.Fatal(err)
			}
			snapshots = tmp
		}
	} else {
		tmp, err := utils.GetSnapshots(repo, cmd.Prefixes)
		if err != nil {
			log.Fatal(err)
		}
		snapshots = tmp
	}

	errors := 0
	wg := sync.WaitGroup{}
	for _, snap := range snapshots {
		if !cmd.BeforeDate.IsZero() && snap.Header.Timestamp.After(cmd.BeforeDate) {
			continue
		}
		if cmd.Tag != "" {
			found := false
			for _, t := range snap.Header.Tags {
				if cmd.Tag == t {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		wg.Add(1)
		go func(snap *snapshot.Snapshot) {
			defer snap.Close()

			t0 := time.Now()
			err := repo.DeleteSnapshot(snap.Header.GetIndexID())
			if err != nil {
				ctx.GetLogger().Error("%s", err)
				errors++
			}
			wg.Done()
			ctx.GetLogger().Info("removed snapshot %x of size %s in %s",
				snap.Header.GetIndexShortID(),
				humanize.Bytes(snap.Header.Summary.Directory.Size+snap.Header.Summary.Below.Size),
				time.Since(t0))
		}(snap)
	}
	wg.Wait()

	if errors != 0 {
		return 1, fmt.Errorf("failed to remove %d snapshots", errors)
	}
	return 0, nil
}
