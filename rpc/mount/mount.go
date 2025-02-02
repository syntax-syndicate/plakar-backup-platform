package mount

import (
	"log"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/plakarfs"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

type Mount struct {
	RepositoryLocation string
	RepositorySecret   []byte

	Mountpoint string
}

func (cmd *Mount) Name() string {
	return "mount"
}

func (cmd *Mount) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	c, err := fuse.Mount(
		cmd.Mountpoint,
		fuse.FSName("plakar"),
		fuse.Subtype("plakarfs"),
		fuse.LocalVolume(),
	)
	if err != nil {
		log.Fatalf("Mount: %v", err)
	}
	defer c.Close()
	ctx.GetLogger().Info("mounted repository %s at %s", repo.Location(), cmd.Mountpoint)

	err = fs.Serve(c, plakarfs.NewFS(repo, cmd.Mountpoint))
	if err != nil {
		return 1, err
	}
	<-c.Ready
	if err := c.MountError; err != nil {
		return 1, err
	}
	return 0, nil

}
