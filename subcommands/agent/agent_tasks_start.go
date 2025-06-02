package agent

import (
	"flag"
	"fmt"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/scheduler"
	"github.com/PlakarKorp/plakar/subcommands"
)

type AgentTasksStart struct {
	subcommands.SubcommandBase
}

func (cmd *AgentTasksStart) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("agent tasks start", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}
	flags.Parse(args)
	if flags.NArg() != 0 {
		return fmt.Errorf("too many arguments")
	}

	return nil
}

func (cmd *AgentTasksStart) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	if agentContextSingleton == nil {
		return 1, fmt.Errorf("agent not started")
	}

	agentContextSingleton.mtx.Lock()
	defer agentContextSingleton.mtx.Unlock()

	if agentContextSingleton.schedulerConfig == nil {
		return 1, fmt.Errorf("agent scheduler does not have a configuration")
	}

	if agentContextSingleton.schedulerState&AGENT_SCHEDULER_RUNNING != 0 {
		return 1, fmt.Errorf("agent scheduler already running")
	}

	// this needs to execute in the agent context, not the client context
	agentContextSingleton.schedulerCtx = appcontext.NewAppContextFrom(agentContextSingleton.agentCtx)
	go scheduler.NewScheduler(agentContextSingleton.schedulerCtx, agentContextSingleton.schedulerConfig).Run()

	agentContextSingleton.schedulerState = AGENT_SCHEDULER_RUNNING
	return 0, nil
}
