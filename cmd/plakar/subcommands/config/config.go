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

package version

import (
	"flag"
	"fmt"
	"strings"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/repository"
)

func init() {
	subcommands.Register("config", parse_cmd_config)
}

func parse_cmd_config(ctx *appcontext.AppContext, repo *repository.Repository, args []string) (subcommands.Subcommand, error) {
	flags := flag.NewFlagSet("config", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s\n", flags.Name())
		flags.PrintDefaults()
	}

	flags.Parse(args)
	args = flags.Args()
	if len(args) > 1 {
		return nil, fmt.Errorf("too many arguments")
	}

	return &Config{
		args: args,
	}, nil
}

type Config struct {
	args []string
}

func (cmd *Config) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	if len(cmd.args) == 0 {
		for label, values := range ctx.Config.Labels {
			fmt.Fprintf(ctx.Stdout, "%s:\n", label)
			for k, v := range values {
				fmt.Fprintf(ctx.Stdout, "  %s: %v\n", k, v)
			}
		}
		return 0, nil
	}

	kv := strings.SplitN(cmd.args[0], "=", 2)
	key := strings.TrimSpace(kv[0])
	atoms := strings.Split(key, ".")
	if len(atoms) < 2 {
		return 0, fmt.Errorf("config: invalid key")
	}

	if len(kv) == 1 {
		if val, ok := ctx.Config.Lookup(atoms[0], atoms[1]); ok {
			fmt.Println(val)
		}
	} else {
		value := strings.TrimSpace(kv[1])
		ctx.Config.Set(atoms[0], atoms[1], value)
		ctx.Config.Save()
	}

	return 0, nil
}
