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
	"github.com/PlakarKorp/plakar/cmd/plakar/utils"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/snapshot/exporter"
	"github.com/PlakarKorp/plakar/snapshot/importer"
	"github.com/PlakarKorp/plakar/storage"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Version{} }, "version")
}

func (_ *Version) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("version", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s\n", flags.Name())
		flags.PrintDefaults()
	}

	flags.Parse(args)

	return nil
}

type Version struct {
	subcommands.SubcommandBase
}

func (cmd *Version) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	fmt.Printf("plakar/%s\n", utils.GetVersion())

	fmt.Println()
	fmt.Println("importers:", strings.Join(importer.Backends(), ", "))
	fmt.Println("exporters:", strings.Join(exporter.Backends(), ", "))
	fmt.Println("klosets:", strings.Join(storage.Backends(), ", "))

	return 0, nil
}
