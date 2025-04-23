package diag

import (
	"flag"
	"fmt"

	"github.com/PlakarKorp/plakar/kloset/appcontext"
	"github.com/PlakarKorp/plakar/kloset/repository"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/utils"
)

type DiagErrors struct {
	subcommands.SubcommandBase

	SnapshotID string
}

func (cmd *DiagErrors) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("diag errors", flag.ExitOnError)
	flags.Parse(args)

	if len(flags.Args()) < 1 {
		return fmt.Errorf("usage: %s errors SNAPSHOT", flags.Name())
	}

	cmd.RepositorySecret = ctx.GetSecret()
	cmd.SnapshotID = flags.Args()[0]

	return nil
}

func (cmd *DiagErrors) Name() string {
	return "diag_errors"
}

func (cmd *DiagErrors) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	snap, pathname, err := utils.OpenSnapshotByPath(repo, cmd.SnapshotID)
	if err != nil {
		return 1, err
	}
	defer snap.Close()

	fs, err := snap.Filesystem()
	if err != nil {
		return 1, err
	}

	errstream, err := fs.Errors(pathname)
	if err != nil {
		return 1, err
	}

	for item := range errstream {
		fmt.Fprintf(ctx.Stdout, "%s: %s\n", item.Name, item.Error)
	}
	return 0, nil
}
