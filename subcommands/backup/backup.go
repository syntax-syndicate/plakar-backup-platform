package backup

import (
	"encoding/base64"
	"fmt"
	"log"
	"path/filepath"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/snapshot"
	"github.com/PlakarKorp/plakar/snapshot/importer"
	"github.com/dustin/go-humanize"
	"github.com/gobwas/glob"
	"github.com/google/uuid"
)

type Backup struct {
	RepositoryLocation string
	RepositorySecret   []byte

	Concurrency uint64
	Identity    string
	Tags        string
	Excludes    []glob.Glob
	Exclude     []string
	Quiet       bool
	Path        string
}

func (cmd *Backup) Name() string {
	return "backup"
}

func (cmd *Backup) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	snap, err := snapshot.New(repo)
	if err != nil {
		ctx.GetLogger().Error("%s", err)
		return 1, err
	}
	defer snap.Close()

	var tags []string
	if cmd.Tags == "" {
		tags = []string{}
	} else {
		tags = []string{cmd.Tags}
	}

	opts := &snapshot.BackupOptions{
		MaxConcurrency: cmd.Concurrency,
		Name:           "default",
		Tags:           tags,
		Excludes:       cmd.Excludes,
	}

	scanDir := ctx.CWD
	if cmd.Path != "" {
		scanDir = cmd.Path
	}

	imp, err := importer.NewImporter(scanDir)
	if err != nil {
		if !filepath.IsAbs(scanDir) {
			scanDir = filepath.Join(ctx.CWD, scanDir)
		}
		imp, err = importer.NewImporter("fs://" + scanDir)
		if err != nil {
			log.Fatalf("failed to create an import for %s: %s", scanDir, err)
		}
	}

	ep := startEventsProcessor(ctx, imp.Root(), true, cmd.Quiet)
	if err := snap.Backup(scanDir, imp, opts); err != nil {
		ep.Close()
		return 1, fmt.Errorf("failed to create snapshot: %w", err)
	}
	ep.Close()

	signedStr := "unsigned"
	if ctx.Identity != uuid.Nil {
		signedStr = "signed"
	}
	ctx.GetLogger().Info("created %s snapshot %x with root %s of size %s in %s",
		signedStr,
		snap.Header.GetIndexShortID(),
		base64.RawStdEncoding.EncodeToString(snap.Header.Root[:]),
		humanize.Bytes(snap.Header.Summary.Directory.Size+snap.Header.Summary.Below.Size),
		snap.Header.Duration)
	return 0, nil
}
