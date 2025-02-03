package subcommands

import (
	"fmt"
	"sort"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/repository"
)

type Subcommand interface {
	Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error)
}

type parse_args_fn func(*appcontext.AppContext, *repository.Repository, []string) (Subcommand, error)

var subcommands map[string]parse_args_fn = make(map[string]parse_args_fn)

func Register(command string, fn parse_args_fn) {
	subcommands[command] = fn
}

func Parse(ctx *appcontext.AppContext, repo *repository.Repository, command string, args []string, agentless bool) (Subcommand, error) {
	parsefn, exists := subcommands[command]
	if !exists {
		return nil, fmt.Errorf("unknown command: %s", command)
	}
	return parsefn(ctx, repo, args)
}

func List() []string {
	var list []string
	for command := range subcommands {
		list = append(list, command)
	}
	sort.Strings(list)
	return list
}
