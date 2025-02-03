package rpc

import (
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/repository"
)

// RPC extends subcommands.Subcommand, but it also includes the Name() method used to identify the RPC on decoding.
type RPC interface {
	Name() string
	Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error)
}
