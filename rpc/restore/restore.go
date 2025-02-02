package restore

import (
	"fmt"
	"log"
	"strings"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/utils"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/snapshot"
	"github.com/PlakarKorp/plakar/snapshot/exporter"
)

type Restore struct {
	RepositoryLocation string
	RepositorySecret   []byte

	Path        string
	Rebase      bool
	Concurrency uint64
	Quiet       bool
	Snapshots   []string
}

func (cmd *Restore) Name() string {
	return "restore"
}

func (cmd *Restore) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	go eventsProcessorStdio(ctx, cmd.Quiet)

	var exporterInstance exporter.Exporter
	var err error
	exporterInstance, err = exporter.NewExporter(cmd.Path)
	if err != nil {
		log.Fatal(err)
	}
	defer exporterInstance.Close()

	opts := &snapshot.RestoreOptions{
		MaxConcurrency: cmd.Concurrency,
		Rebase:         cmd.Rebase,
	}

	if len(cmd.Snapshots) == 0 {
		metadatas, err := utils.GetHeaders(repo, nil)
		if err != nil {
			log.Fatal(err)
		}

		for i := len(metadatas); i != 0; i-- {
			metadata := metadatas[i-1]
			if ctx.CWD == metadata.Importer.Directory || strings.HasPrefix(ctx.CWD, fmt.Sprintf("%s/", metadata.Importer.Directory)) {
				snap, err := snapshot.Load(repo, metadata.GetIndexID())
				if err != nil {
					return 1, err
				}
				snap.Restore(exporterInstance, ctx.CWD, ctx.CWD, opts)
				snap.Close()
				return 0, nil
			}
		}
		return 1, fmt.Errorf("could not find a snapshot to restore this path from")
	}

	snapshots, err := utils.GetSnapshots(repo, cmd.Snapshots)
	if err != nil {
		return 1, err
	}

	for offset, snap := range snapshots {
		_, pattern := utils.ParseSnapshotID(cmd.Snapshots[offset])
		snap.Restore(exporterInstance, exporterInstance.Root(), pattern, opts)
		snap.Close()
	}
	return 0, nil
}
