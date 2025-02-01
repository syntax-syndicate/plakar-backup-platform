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

package checksum

import (
	"flag"
	"fmt"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/rpc"
	"github.com/PlakarKorp/plakar/rpc/checksum"
)

func init() {
	subcommands.Register2("checksum", parse_cmd_checksum)
}

func parse_cmd_checksum(ctx *appcontext.AppContext, repo *repository.Repository, args []string) (rpc.RPC, error) {
	var enableFastChecksum bool

	flags := flag.NewFlagSet("checksum", flag.ExitOnError)
	flags.BoolVar(&enableFastChecksum, "fast", false, "enable fast checksum (return recorded checksum)")

	flags.Parse(args)

	if flags.NArg() == 0 {
		ctx.GetLogger().Error("%s: at least one parameter is required", flags.Name())
		return nil, fmt.Errorf("at least one parameter is required")
	}

	return &checksum.Checksum{
		RepositoryLocation: repo.Location(),
		RepositorySecret:   ctx.GetSecret(),
		Fast:               enableFastChecksum,
		Targets:            flags.Args(),
	}, nil
}
