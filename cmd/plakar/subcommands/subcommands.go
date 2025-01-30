package subcommands

import (
	"fmt"
	"sort"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/handlers"
	"github.com/PlakarKorp/plakar/repository"
)

type parsefn func(*appcontext.AppContext, *repository.Repository, []string) (handlers.Subcommand, error)

var subcommands2 map[string]parsefn = make(map[string]parsefn)

func Register2(command string, fn parsefn) {
	subcommands2[command] = fn
}

////

var subcommands map[string]func(*appcontext.AppContext, *repository.Repository, []string) (int, error) = make(map[string]func(*appcontext.AppContext, *repository.Repository, []string) (int, error))

func Register(command string, fn func(*appcontext.AppContext, *repository.Repository, []string) (int, error)) {
	subcommands[command] = fn
}

func Parse(ctx *appcontext.AppContext, repo *repository.Repository, command string, args []string, agentless bool) (handlers.Subcommand, error) {
	parsefn, exists := subcommands2[command]
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
