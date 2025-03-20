package diag

import (
	"fmt"
	"strings"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/utils"
	"github.com/PlakarKorp/plakar/repository"
)

type DiagContentType struct {
	RepositorySecret []byte

	SnapshotPath string
}

func (cmd *DiagContentType) Name() string {
	return "diag_contenttype"
}

func (cmd *DiagContentType) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	snap, pathname, err := utils.OpenSnapshotByPath(repo, cmd.SnapshotPath)
	if err != nil {
		return 1, err
	}
	defer snap.Close()

	if pathname == "" {
		pathname = "/"
	}
	if !strings.HasSuffix(pathname, "/") {
		pathname += "/"
	}

	tree, err := snap.ContentTypeIdx()
	if err != nil {
		return 1, err
	}
	if tree == nil {
		return 1, fmt.Errorf("no content-type index available in the snapshot")
	}

	it, err := tree.ScanFrom(pathname)
	if err != nil {
		return 1, err
	}

	for it.Next() {
		path, _ := it.Current()
		if !strings.HasPrefix(path, pathname) {
			break
		}

		fmt.Fprintln(ctx.Stdout, path)
	}
	if err := it.Err(); err != nil {
		return 1, err
	}

	return 0, nil
}
