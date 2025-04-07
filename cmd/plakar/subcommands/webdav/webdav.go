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
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/snapfs"
	"golang.org/x/net/webdav"
)

func init() {
	subcommands.Register("webdav", parse_cmd_webdav)
}

func parse_cmd_webdav(ctx *appcontext.AppContext, args []string) (subcommands.Subcommand, error) {
	flags := flag.NewFlagSet("webdav", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s\n", flags.Name())
		flags.PrintDefaults()
	}
	flags.Parse(args)

	snapshotID := flags.Arg(0)

	return &Webdav{
		SnapshotPath: snapshotID,
	}, nil
}

type Webdav struct {
	SnapshotPath string
}

type PlakarFS struct {
	vfsRoot snapfs.FS
}

func (fs *PlakarFS) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	return os.ErrPermission // read-only for now
}

func (fs *PlakarFS) RemoveAll(ctx context.Context, name string) error {
	return os.ErrPermission
}

func (fs *PlakarFS) Rename(ctx context.Context, oldName, newName string) error {
	return os.ErrPermission
}

func (fs *PlakarFS) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	return fs.vfsRoot.Stat(name)
}

func (fs *PlakarFS) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	return fs.vfsRoot.Open(name)
}

type file struct {
	vfs *snapfs.FS
	fp fs.File
	path string
}

func (f *file) Write([]byte) (int, error)  { return 0, errors.ErrUnsupported }
func (f *file) Read(p []byte) (int, error) { return f.fp.Read(p) }
func (f *file) Close() error               { return f.fp.Close() }

func (f *file) Seek(offset int64, whence int) (int64, error) {
	if seeker, ok := f.fp.(io.Seeker); ok {
		return seeker.Seek(offset, whence)
	}
	return 0, errors.ErrUnsupported
}

func (f *file) Readdir(count int) (ret []fs.FileInfo, err error) {
	if dir, ok := f.fp.(fs.ReadDirFile); ok {
		return dir.ReadDir(count)
		// info, err := dir.ReadDir(count)
		// if err != nil {
		// 	return nil, err
		// }
		// for i := range info {
		// 	fp, err := f.vfs.Open(path.Join(f.path, info[i].Name()))
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// 	sb, err := fp.Stat()
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// 	ret = append(ret, sb)
		// }
	}
	return nil, errors.ErrUnsupported
}

func (f *PlakarFile) Stat() (os.FileInfo, error) {
	return f.info, nil
}

func (d *PlakarDir) Close() error {
	return nil
}

func (d *PlakarDir) Write([]byte) (int, error) {
	return 0, fmt.Errorf("cannot write to directory")
}

func (d *PlakarDir) Read(p []byte) (int, error) {
	return 0, fmt.Errorf("cannot read directory")
}

func (d *PlakarDir) Seek(offset int64, whence int) (int64, error) {
	return 0, fmt.Errorf("seek not supported")
}

func (d *PlakarDir) Readdir(count int) ([]os.FileInfo, error) {
	iter, err := d.entry.Getdents(d.vfsRoot)
	if err != nil {
		return nil, err
	}

	ret := make([]os.FileInfo, 0)
	for childEntry, err := range iter {
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		ret = append(ret, childEntry.Stat())
	}
	return ret, nil
}

func (d *PlakarDir) Stat() (os.FileInfo, error) {
	return d.entry.Stat(), nil
}

func (cmd *Webdav) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	// snap, _, err := utils.OpenSnapshotByPath(repo, cmd.SnapshotPath)
	// if err != nil {
	// 	ctx.GetLogger().Error("%s", err)
	// 	return 1, fmt.Errorf("webdav: could not open snapshot: %s", cmd.SnapshotPath)
	// }
	// defer snap.Close()

	// vfsRoot, err := snap.Filesystem()
	// if err != nil {
	// 	ctx.GetLogger().Error("%s", err)
	// 	return 1, fmt.Errorf("webdav: could not open snapshot filesystem: %s", cmd.SnapshotPath)
	// }

	vfs, err := snapfs.NewFS(repo)
	if err != nil {
		return 1, fmt.Errorf("webdav: failed to open VFS: %w", err)
	}

	fs := &PlakarFS{vfsRoot: vfs}

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
