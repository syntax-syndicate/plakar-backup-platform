package subcommands

import (
	"fmt"
	"sort"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/rpc"
)

type parse_args_fn func(*appcontext.AppContext, *repository.Repository, []string) (rpc.RPC, error)

var subcommands map[string]parse_args_fn = make(map[string]parse_args_fn)

func Register(command string, fn parse_args_fn) {
	subcommands[command] = fn
}

func Parse(ctx *appcontext.AppContext, repo *repository.Repository, command string, args []string, agentless bool) (rpc.RPC, error) {
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
