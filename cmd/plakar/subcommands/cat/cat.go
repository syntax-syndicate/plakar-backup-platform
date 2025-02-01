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

package cat

import (
	"flag"
	"fmt"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/repository"
	api_subcommands "github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/subcommands/cat"
)

func init() {
	subcommands.Register2("cat", parse_cmd_cat)
}

func parse_cmd_cat(ctx *appcontext.AppContext, repo *repository.Repository, args []string) (api_subcommands.Subcommand, error) {
	var opt_nodecompress bool
	var opt_highlight bool

	flags := flag.NewFlagSet("cat", flag.ExitOnError)
	flags.BoolVar(&opt_nodecompress, "no-decompress", false, "do not try to decompress output")
	flags.BoolVar(&opt_highlight, "highlight", false, "highlight output")
	flags.Parse(args)

	if flags.NArg() == 0 {
		return nil, fmt.Errorf("at least one parameter is required")
	}

	return &cat.Cat{
		RepositoryLocation: repo.Location(),
		RepositorySecret:   ctx.GetSecret(),
		NoDecompress:       opt_nodecompress,
		Highlight:          opt_highlight,
		Paths:              flags.Args(),
	}, nil
}
