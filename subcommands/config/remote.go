package config

import (
	"flag"
	"fmt"
	"strings"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
)

type ConfigRemoteCmd struct {
	subcommands.SubcommandBase

	args []string
}

func (cmd *ConfigRemoteCmd) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("remote", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s\n", flags.Name())
		flags.PrintDefaults()
	}

	flags.Parse(args)
	cmd.args = flags.Args()

	return nil
}

func (cmd *ConfigRemoteCmd) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {

	err := cmd_remote_config(ctx, cmd.args)
	if err != nil {
		return 1, err
	}
	return 0, nil
}

func cmd_remote_config(ctx *appcontext.AppContext, args []string) error {
	usage := "usage: plakar remote [create | set | unset | check | ping]"
	if len(args) == 0 {
		return fmt.Errorf(usage)
	}

	switch args[0] {
	case "create":
		usage := "usage: plakar remote create <name> <location> [<key>=<value>, ...]"
		if len(args) < 3 {
			return fmt.Errorf(usage)
		}
		name, location := args[1], args[2]
		if ctx.Config.HasRemote(name) {
			return fmt.Errorf("remote %q already exists", name)
		}
		ctx.Config.Remotes[name] = make(map[string]string)
		ctx.Config.Remotes[name]["location"] = location
		for _, kv := range args[3:] {
			key, val, found := strings.Cut(kv, "=")
			if !found {
				return fmt.Errorf(usage)
			}
			if key == "" {
				return fmt.Errorf(usage)
			}
			ctx.Config.Remotes[name][key] = val
		}
		return ctx.Config.Save()

	case "set":
		usage := "usage: plakar remote set <name> [<key>=<value>, ...]"
		if len(args) < 2 {
			return fmt.Errorf(usage)
		}
		name := args[1]
		if !ctx.Config.HasRemote(name) {
			return fmt.Errorf("remote %q does not exists", name)
		}
		for _, kv := range args[2:] {
			key, val, found := strings.Cut(kv, "=")
			if !found {
				return fmt.Errorf(usage)
			}
			if key == "" {
				return fmt.Errorf(usage)
			}
			ctx.Config.Remotes[name][key] = val
		}
		return ctx.Config.Save()

	case "unset":
		usage := "usage: plakar remote unset <name> [<key>, ...]"
		if len(args) < 2 {
			return fmt.Errorf(usage)
		}
		name := args[1]
		if !ctx.Config.HasRemote(name) {
			return fmt.Errorf("remote %q does not exists", name)
		}
		for _, key := range args[2:] {
			if key == "location" {
				return fmt.Errorf("cannot unset location")
			}
			delete(ctx.Config.Remotes[name], key)
		}
		return ctx.Config.Save()

	case "check":
		usage := "usage: plakar remote check <name>"
		if len(args) != 2 {
			return fmt.Errorf(usage)
		}
		name := args[1]
		if !ctx.Config.HasRemote(name) {
			return fmt.Errorf("remote %q does not exists", name)
		}

		return fmt.Errorf("check not implemented")

	case "ping":
		usage := "usage: plakar remote ping <name>"
		if len(args) != 2 {
			return fmt.Errorf(usage)
		}
		name := args[1]
		if !ctx.Config.HasRemote(name) {
			return fmt.Errorf("remote %q does not exists", name)
		}

		return fmt.Errorf("ping not implemented")

	default:
		return fmt.Errorf(usage)
	}
}
