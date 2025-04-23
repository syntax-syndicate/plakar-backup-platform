package diag

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/cmd/plakar/utils"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/snapshot"
)

type DiagSearch struct {
	subcommands.SubcommandBase

	SnapshotPath string
	Mimes        []string
}

func (cmd *DiagSearch) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("diag search", flag.ExitOnError)
	flags.Parse(args)

	var path string
	var mimes []string

	switch flags.NArg() {
	case 1:
		path = flags.Arg(0)
	case 2:
		path, mimes = flags.Arg(0), strings.Split(flags.Arg(1), ",")
	default:
		return fmt.Errorf("usage: %s search snapshot[:path] mimes",
			flags.Name())
	}

	cmd.RepositorySecret = ctx.GetSecret()
	cmd.SnapshotPath = path
	cmd.Mimes = mimes

	return nil
}

func (cmd *DiagSearch) Name() string {
	return "diag_search"
}

func (cmd *DiagSearch) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	snap, pathname, err := utils.OpenSnapshotByPath(repo, cmd.SnapshotPath)
	if err != nil {
		return 1, err
	}
	defer snap.Close()

	opts := snapshot.SearchOpts{
		Recursive: true,
		Prefix:    pathname,
		Mimes:     cmd.Mimes,
	}
	it, err := snap.Search(context.Background(), &opts)
	if err != nil {
		return 1, err
	}

	for entry, err := range it {
		if err != nil {
			return 1, err
		}
		fmt.Fprintf(ctx.Stdout, "%x:%s\n", snap.Header.Identifier[0:4], entry.Path())
	}

	return 0, nil
}
