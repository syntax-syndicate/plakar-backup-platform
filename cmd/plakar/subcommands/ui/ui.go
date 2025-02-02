//go:build go1.16
// +build go1.16

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

package ui

import (
	"flag"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/rpc"
	"github.com/PlakarKorp/plakar/rpc/ui"
)

func init() {
	subcommands.Register("ui", parse_cmd_ui)
}

func parse_cmd_ui(ctx *appcontext.AppContext, repo *repository.Repository, args []string) (rpc.RPC, error) {
	var opt_addr string
	var opt_cors bool
	var opt_noauth bool
	var opt_nospawn bool

	flags := flag.NewFlagSet("ui", flag.ExitOnError)
	flags.StringVar(&opt_addr, "addr", "", "address to listen on")
	flags.BoolVar(&opt_cors, "cors", false, "enable CORS")
	flags.BoolVar(&opt_noauth, "no-auth", false, "don't use authentication")
	flags.BoolVar(&opt_nospawn, "no-spawn", false, "don't spawn browser")
	flags.Parse(args)
	return &ui.Ui{
		RepositoryLocation: repo.Location(),
		RepositorySecret:   ctx.GetSecret(),
		Addr:               opt_addr,
		Cors:               opt_cors,
		NoAuth:             opt_noauth,
		NoSpawn:            opt_nospawn,
	}, nil
}
