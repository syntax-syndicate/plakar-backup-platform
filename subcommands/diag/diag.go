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
	"github.com/PlakarKorp/plakar/subcommands"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &DiagSnapshot{} }, "diag", "snapshot")
	subcommands.Register(func() subcommands.Subcommand { return &DiagErrors{} }, "diag", "errors")
	subcommands.Register(func() subcommands.Subcommand { return &DiagState{} }, "diag", "state")
	subcommands.Register(func() subcommands.Subcommand { return &DiagPackfile{} }, "diag", "packfile")
	subcommands.Register(func() subcommands.Subcommand { return &DiagObject{} }, "diag", "object")
	subcommands.Register(func() subcommands.Subcommand { return &DiagVFS{} }, "diag", "vfs")
	subcommands.Register(func() subcommands.Subcommand { return &DiagXattr{} }, "diag", "xattr")
	subcommands.Register(func() subcommands.Subcommand { return &DiagContentType{} }, "diag", "contenttype")
	subcommands.Register(func() subcommands.Subcommand { return &DiagLocks{} }, "diag", "locks")
	subcommands.Register(func() subcommands.Subcommand { return &DiagSearch{} }, "diag", "search")
	subcommands.Register(func() subcommands.Subcommand { return &DiagRepository{} }, "diag")
}
