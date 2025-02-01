package rpc

import (
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/repository"
)

type RPC interface {
	Name() string
	Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error)
}
