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

package diff

import (
	"flag"
	"fmt"
	"log"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/cmd/plakar/utils"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/rpc/diff"
)

func init() {
	subcommands.Register("diff", parse_cmd_diff)
}

func parse_cmd_diff(ctx *appcontext.AppContext, repo *repository.Repository, args []string) (subcommands.Subcommand, error) {
	var opt_highlight bool
	flags := flag.NewFlagSet("diff", flag.ExitOnError)
	flags.BoolVar(&opt_highlight, "highlight", false, "highlight output")
	flags.Parse(args)

	if flags.NArg() != 2 {
		fmt.Println("args", flags.Args())
		log.Fatalf("%s: needs two snapshot ID and/or snapshot files to diff", flag.CommandLine.Name())
	}

	snapshotPrefix1, pathname1 := utils.ParseSnapshotID(flags.Arg(0))
	snapshotPrefix2, pathname2 := utils.ParseSnapshotID(flags.Arg(1))

	return &diff.Diff{
		RepositoryLocation: repo.Location(),
		RepositorySecret:   ctx.GetSecret(),
		Highlight:          opt_highlight,
		SnapshotPrefix1:    snapshotPrefix1,
		Pathname1:          pathname1,
		SnapshotPrefix2:    snapshotPrefix2,
		Pathname2:          pathname2,
	}, nil
}
