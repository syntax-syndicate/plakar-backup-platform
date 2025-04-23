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

package webdav

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/cmd/plakar/utils"
	"github.com/PlakarKorp/plakar/repository"
	"golang.org/x/net/webdav"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Webdav{} }, "webdav")
}

type Webdav struct {
	subcommands.SubcommandBase

	SnapshotPath string
}

func (cmd *Webdav) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("webdav", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s\n", flags.Name())
		flags.PrintDefaults()
	}
	flags.Parse(args)

	if flags.NArg() < 1 {
		return fmt.Errorf("webdav: missing snapshot path")
	}

	cmd.SnapshotPath = flags.Arg(0)
	return nil
}

func (cmd *Webdav) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {

	snap, _, err := utils.OpenSnapshotByPath(repo, cmd.SnapshotPath)
	if err != nil {
		ctx.GetLogger().Error("%s", err)
		return 1, fmt.Errorf("webdav: could not open snapshot: %s", cmd.SnapshotPath)
	}
	defer snap.Close()

	vfsRoot, err := snap.Filesystem()
	if err != nil {
		ctx.GetLogger().Error("%s", err)
		return 1, fmt.Errorf("webdav: could not open snapshot filesystem: %s", cmd.SnapshotPath)
	}

	fs := &PlakarFS{vfsRoot: vfsRoot}

	handler := &webdav.Handler{
		Prefix:     "/", // WebDAV path prefix
		FileSystem: fs,
		LockSystem: webdav.NewMemLS(),
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Optional: add basic auth
		// user, pass, _ := r.BasicAuth()
		// if user != "admin" || pass != "secret" {
		//     w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		//     http.Error(w, "Unauthorized", http.StatusUnauthorized)
		//     return
		// }
		handler.ServeHTTP(w, r)
	})

	log.Println("Starting WebDAV server on http://localhost:8080/")
	return 1, http.ListenAndServe(":8080", nil)
}
