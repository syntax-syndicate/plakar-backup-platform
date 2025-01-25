package subcommands

import (
	"fmt"
	"sort"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/repository"
)

var subcommands map[string]func(*appcontext.AppContext, *repository.Repository, []string) int = make(map[string]func(*appcontext.AppContext, *repository.Repository, []string) int)

func Register(command string, fn func(*appcontext.AppContext, *repository.Repository, []string) int) {
	subcommands[command] = fn
}

func Execute(ctx *appcontext.AppContext, repo *repository.Repository, command string, args []string) (int, error) {
	fn, exists := subcommands[command]
	if !exists {
		return 1, fmt.Errorf("unknown command: %s", command)
	}
	return fn(ctx, repo, args), nil
}

func List() []string {
	var list []string
	for command := range subcommands {
		list = append(list, command)
	}
	sort.Strings(list)
	return list
}
