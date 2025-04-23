package diag

import (
	"flag"
	"fmt"
	"strings"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/btree"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/cmd/plakar/utils"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/resources"
	"github.com/PlakarKorp/plakar/snapshot/vfs"
)

type DiagXattr struct {
	subcommands.SubcommandBase

	SnapshotPath string
}

func (cmd *DiagXattr) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("diag xattr", flag.ExitOnError)
	flags.Parse(args)

	if len(flags.Args()) < 1 {
		return fmt.Errorf("usage: %s xattr SNAPSHOT[:PATH]", flags.Name())
	}

	cmd.RepositorySecret = ctx.GetSecret()
	cmd.SnapshotPath = flags.Args()[0]
	return nil
}

func (cmd *DiagXattr) Name() string {
	return "diag_xattr"
}

func (cmd *DiagXattr) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
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

	rd, err := repo.GetBlob(resources.RT_XATTR_BTREE, snap.Header.GetSource(0).VFS.Xattrs)
	if err != nil {
		return 1, err
	}

	store := repository.NewRepositoryStore[string, objects.MAC](repo, resources.RT_XATTR_NODE)
	tree, err := btree.Deserialize(rd, store, vfs.PathCmp)
	if err != nil {
		return 1, err
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
