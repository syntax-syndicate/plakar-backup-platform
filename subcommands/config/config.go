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

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/subcommands/agent"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &agent.AgentRestart{} },
		subcommands.AgentSupport|subcommands.BeforeRepositoryOpen|subcommands.IgnoreVersion, "config", "reload")
	subcommands.Register(func() subcommands.Subcommand { return &ConfigCmd{} },
		subcommands.BeforeRepositoryOpen, "config")
	subcommands.Register(func() subcommands.Subcommand { return &ConfigKlosetCmd{} },
		subcommands.BeforeRepositoryOpen, "kloset")
	subcommands.Register(func() subcommands.Subcommand { return &ConfigRemoteCmd{} },
		subcommands.BeforeRepositoryOpen, "remote")
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
	if len(cmd.args) != 0 {
		return 1, fmt.Errorf("config command takes no argument")
	}
	ctx.Config.Render(ctx.Stdout)
	return 0, nil
}
