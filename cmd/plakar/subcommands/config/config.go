/*
 * Copyright (c) 2021 Gilles Chehade <gilles@poolp.org>
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package config

import (
	"flag"
	"fmt"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/repository"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &ConfigCmd{} }, "config")
}

func (cmd *ConfigCmd) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("config", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s\n", flags.Name())
		flags.PrintDefaults()
	}

	flags.Parse(args)
	cmd.args = flags.Args()

	return nil
}

type ConfigCmd struct {
	subcommands.SubcommandBase

	args []string
}

func (cmd *ConfigCmd) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	if len(cmd.args) == 0 {
		ctx.Config.Render(ctx.Stdout)
		return 0, nil
	}

	var err error
	switch cmd.args[0] {
	case "remote":
		err = cmd_remote(ctx, cmd.args[1:])
	case "repository", "repo":
		err = cmd_repository(ctx, cmd.args[1:])
	default:
		err = fmt.Errorf("unknown subcommand %s", cmd.args[0])
	}

	if err != nil {
		return 1, err
	}
	return 0, nil
}

func cmd_remote(ctx *appcontext.AppContext, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: plakar config remote [create | set | unset | validate]")
	}

	switch args[0] {
	case "create":
		if len(args) != 2 {
			return fmt.Errorf("usage: plakar config remote create name")
		}
		name := args[1]
		if ctx.Config.HasRemote(name) {
			return fmt.Errorf("remote %q already exists", name)
		}
		ctx.Config.Remotes[name] = make(map[string]string)
		return ctx.Config.Save()

	case "set":
		if len(args) != 4 {
			return fmt.Errorf("usage: plakar config remote set name option value")
		}
		name, option, value := args[1], args[2], args[3]
		if !ctx.Config.HasRemote(name) {
			return fmt.Errorf("remote %q does not exists", name)
		}
		ctx.Config.Remotes[name][option] = value
		return ctx.Config.Save()

	case "unset":
		if len(args) != 3 {
			return fmt.Errorf("usage: plakar config remote unset name option")
		}
		name, option := args[1], args[2]
		if !ctx.Config.HasRemote(name) {
			return fmt.Errorf("remote %q does not exists", name)
		}
		delete(ctx.Config.Remotes[name], option)
		return ctx.Config.Save()

	case "validate":
		if len(args) != 2 {
			return fmt.Errorf("usage: plakar config remote validate name")
		}
		return fmt.Errorf("validation not implemented")

	default:
		return fmt.Errorf("usage: plakar config remote [create | set | unset | validate]")
	}
}

func cmd_repository(ctx *appcontext.AppContext, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: plakar config repository [create | default | set | unset | validate]")
	}

	switch args[0] {
	case "create":
		if len(args) != 2 {
			return fmt.Errorf("usage: plakar config repository create name")
		}
		name := args[1]
		if ctx.Config.HasRepository(name) {
			return fmt.Errorf("repository %q already exists", name)
		}
		ctx.Config.Repositories[name] = make(map[string]string)
		return ctx.Config.Save()

	case "default":
		if len(args) != 2 {
			return fmt.Errorf("usage: plakar config repository default name")
		}
		name := args[1]
		if !ctx.Config.HasRepository(name) {
			return fmt.Errorf("repository %q doesn't exist", name)
		}
		if _, ok := ctx.Config.Repositories[name]["location"]; !ok {
			return fmt.Errorf("repository %q doesn't have a location set", name)
		}
		ctx.Config.DefaultRepository = name
		return ctx.Config.Save()

	case "set":
		if len(args) != 4 {
			return fmt.Errorf("usage: plakar config repository set name option value")
		}
		name, option, value := args[1], args[2], args[3]
		if !ctx.Config.HasRepository(name) {
			return fmt.Errorf("repository %q does not exists", name)
		}
		ctx.Config.Repositories[name][option] = value
		return ctx.Config.Save()

	case "unset":
		if len(args) != 3 {
			return fmt.Errorf("usage: plakar config repository unset name option")
		}
		name, option := args[1], args[2]
		if !ctx.Config.HasRepository(name) {
			return fmt.Errorf("repository %q does not exists", name)
		}
		delete(ctx.Config.Repositories[name], option)
		return ctx.Config.Save()

	case "validate":
		if len(args) != 2 {
			return fmt.Errorf("usage: plakar config repository validate name")
		}
		return fmt.Errorf("validation not implemented")

	default:
		return fmt.Errorf("usage: plakar config repository [create | default | set | unset | validate]")
	}
}
