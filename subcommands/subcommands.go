package subcommands

import (
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/repository"
)

type Subcommand interface {
	Name() string
	Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error)
}
