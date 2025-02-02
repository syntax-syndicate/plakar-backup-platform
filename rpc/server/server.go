package server

import (
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/server/httpd"
	"github.com/PlakarKorp/plakar/server/plakard"
)

type Server struct {
	RepositoryLocation string
	RepositorySecret   []byte

	Protocol string
	Addr     string
	NoDelete bool
}

func (cmd *Server) Name() string {
	return "server"
}

func (cmd *Server) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	switch cmd.Protocol {
	case "http":
		httpd.Server(repo, cmd.Addr, cmd.NoDelete)
	case "plakar":
		options := &plakard.ServerOptions{
			NoOpen:   true,
			NoCreate: true,
			NoDelete: cmd.NoDelete,
		}
		plakard.Server(ctx, repo, cmd.Addr, options)
	default:
		ctx.GetLogger().Error("unsupported protocol: %s", cmd.Protocol)
	}
	return 0, nil
}
