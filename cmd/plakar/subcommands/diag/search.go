package diag

import (
	"context"
	"fmt"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/utils"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/snapshot"
)

type DiagSearch struct {
	RepositorySecret []byte

	SnapshotPath string
	Mimes        []string
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
