package archive

import (
	"flag"
	"io"
	"log"
	"os"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/utils"
	"github.com/PlakarKorp/plakar/repository"
)

type Archive struct {
	RepositoryLocation string
	RepositorySecret   []byte

	Rebase         bool
	Output         string
	Format         string
	SnapshotPrefix string
}

func (cmd *Archive) Name() string {
	return "archive"
}

func (cmd *Archive) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	snapshotPrefix, pathname := utils.ParseSnapshotID(cmd.SnapshotPrefix)
	snap, err := utils.OpenSnapshotByPrefix(repo, snapshotPrefix)
	if err != nil {
		log.Fatalf("%s: could not open snapshot: %s", flag.CommandLine.Name(), snapshotPrefix)
	}
	defer snap.Close()

	var out io.WriteCloser
	if cmd.Output == "-" {
		out = os.Stdout
	} else {
		tmp, err := os.CreateTemp("", "plakar-archive-")
		if err != nil {
			log.Fatalf("%s: %s: %s", flag.CommandLine.Name(), pathname, err)
		}
		defer os.Remove(tmp.Name())
		out = tmp
	}

	if err = snap.Archive(out, cmd.Format, []string{pathname}, cmd.Rebase); err != nil {
		log.Fatal(err)
	}

	if err := out.Close(); err != nil {
		return 1, err
	}
	if out, isFile := out.(*os.File); isFile {
		if err := os.Rename(out.Name(), cmd.Output); err != nil {
			return 1, err
		}
	}
	return 0, nil

}
