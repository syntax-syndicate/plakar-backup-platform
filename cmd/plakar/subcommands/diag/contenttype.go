package info

import (
	"fmt"
	"strings"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/btree"
	"github.com/PlakarKorp/plakar/cmd/plakar/utils"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/resources"
)

type InfoContentType struct {
	RepositoryLocation string
	RepositorySecret   []byte

	SnapshotPath string
}

func (cmd *InfoContentType) Name() string {
	return "info_contenttype"
}

func (cmd *InfoContentType) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	snapshotPrefix, _ := utils.ParseSnapshotID(cmd.SnapshotPath)
	snap, err := utils.OpenSnapshotByPrefix(repo, snapshotPrefix)
	if err != nil {
		return 1, err
	}
	defer snap.Close()

	rd, err := repo.GetBlob(resources.RT_BTREE, snap.Header.GetSource(0).Indexes[0].Value)
	if err != nil {
		return 1, err
	}

	store := repository.NewRepositoryStore[string, objects.MAC](repo, resources.RT_BTREE)
	tree, err := btree.Deserialize(rd, store, strings.Compare)
	if err != nil {
		return 1, err
	}

	it, err := tree.ScanAll()
	if err != nil {
		return 1, err
	}

	for it.Next() {
		path, _ := it.Current()
		fmt.Fprintln(ctx.Stdout, path)
	}

	return 0, nil
}
