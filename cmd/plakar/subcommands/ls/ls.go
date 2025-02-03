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

package ls

import (
	"flag"
	"fmt"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/rpc/ls"
)

func init() {
	subcommands.Register("ls", parse_cmd_ls)
}

func parse_cmd_ls(ctx *appcontext.AppContext, repo *repository.Repository, args []string) (subcommands.Subcommand, error) {
	var opt_recursive bool
	var opt_tag string
	var opt_uuid bool

	flags := flag.NewFlagSet("ls", flag.ExitOnError)
	flags.BoolVar(&opt_uuid, "uuid", false, "display uuid instead of short ID")
	flags.StringVar(&opt_tag, "tag", "", "filter by tag")
	flags.BoolVar(&opt_recursive, "recursive", false, "recursive listing")
	flags.Parse(args)

	if flags.NArg() > 1 {
		return nil, fmt.Errorf("too many arguments")
	}

	return &ls.Ls{
		RepositoryLocation: repo.Location(),
		RepositorySecret:   ctx.GetSecret(),
		Recursive:          opt_recursive,
		Tag:                opt_tag,
		DisplayUUID:        opt_uuid,
		Path:               flags.Arg(0),
	}, nil
}
