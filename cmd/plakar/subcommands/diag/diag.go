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

package diag

import (
	"flag"
	"fmt"
	"strings"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
)

func init() {
	subcommands.Register("diag", parse_cmd_diag)
}

func parse_cmd_diag(ctx *appcontext.AppContext, args []string) (subcommands.Subcommand, error) {
	if len(args) == 0 {
		return &DiagRepository{
			RepositorySecret: ctx.GetSecret(),
		}, nil
	}

	flags := flag.NewFlagSet("diag", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s snapshot SNAPSHOT\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s errors SNAPSHOT\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s state [STATE]...\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s search snapshot[:path] mime\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s packfile [PACKFILE]...\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s object [OBJECT]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s vfs SNAPSHOT[:PATH]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s xattr SNAPSHOT[:PATH]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s contenttype SNAPSHOT[:PATH]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s locks\n", flags.Name())
	}
	flags.Parse(args)

	// Determine which concept to show information for based on flags.Args()[0]
	switch flags.Arg(0) {
	case "snapshot":
		if len(flags.Args()) < 2 {
			return nil, fmt.Errorf("usage: %s snapshot SNAPSHOT", flags.Name())
		}
		return &DiagSnapshot{
			RepositorySecret: ctx.GetSecret(),
			SnapshotID:       flags.Args()[1],
		}, nil
	case "errors":
		if len(flags.Args()) < 2 {
			return nil, fmt.Errorf("usage: %s errors SNAPSHOT", flags.Name())
		}
		return &DiagErrors{
			RepositorySecret: ctx.GetSecret(),
			SnapshotID:       flags.Args()[1],
		}, nil
	case "state":
		return &DiagState{
			RepositorySecret: ctx.GetSecret(),
			Args:             flags.Args()[1:],
		}, nil
	case "packfile":
		optLocate := ""
		flags := flag.NewFlagSet("diag packfile", flag.ExitOnError)
		flags.Usage = func() {
			fmt.Fprintf(flags.Output(), "Usage: %s\n", flags.Name())
			fmt.Fprintf(flags.Output(), "       %s packfile [PACKFILE]...\n", flags.Name())
		}
		flags.StringVar(&optLocate, "locate", "", "Locate resource in packfile")
		flags.Parse(args[1:])

		return &DiagPackfile{
			RepositorySecret: ctx.GetSecret(),
			Args:             flags.Args(),
			Locate:           optLocate,
		}, nil
	case "object":
		if len(flags.Args()) < 2 {
			return nil, fmt.Errorf("usage: %s object OBJECT", flags.Name())
		}
		return &DiagObject{
			RepositorySecret: ctx.GetSecret(),
			ObjectID:         flags.Args()[1],
		}, nil
	case "vfs":
		if len(flags.Args()) < 2 {
			return nil, fmt.Errorf("usage: %s vfs SNAPSHOT[:PATH]", flags.Name())
		}
		return &DiagVFS{
			RepositorySecret: ctx.GetSecret(),
			SnapshotPath:     flags.Args()[1],
		}, nil
	case "xattr":
		if len(flags.Args()) < 2 {
			return nil, fmt.Errorf("usage: %s xattr SNAPSHOT[:PATH]", flags.Name())
		}
		return &DiagXattr{
			RepositorySecret: ctx.GetSecret(),
			SnapshotPath:     flags.Args()[1],
		}, nil
	case "contenttype":
		if len(flags.Args()) < 2 {
			return nil, fmt.Errorf("usage: %s contenttype SNAPSHOT[:PATH]", flags.Name())
		}
		return &DiagContentType{
			RepositorySecret: ctx.GetSecret(),
			SnapshotPath:     flags.Args()[1],
		}, nil
	case "locks":
		return &DiagLocks{
			RepositorySecret: ctx.GetSecret(),
		}, nil
	case "search":
		var path string
		var mimes []string

		switch flags.NArg() {
		case 2:
			path = flag.Arg(2)
		case 3:
			path, mimes = flag.Arg(2), strings.Split(flag.Arg(3), ",")
		default:
			return nil, fmt.Errorf("usage: %s search snapshot[:path] mimes",
				flags.Name())
		}
		return &DiagSearch{
			RepositorySecret: ctx.GetSecret(),
			SnapshotPath:     path,
			Mimes:            mimes,
		}, nil
	}
	return nil, fmt.Errorf("Invalid parameter. usage: diag [contenttype|snapshot|object|state|packfile|vfs|xattr|errors|search]")
}
