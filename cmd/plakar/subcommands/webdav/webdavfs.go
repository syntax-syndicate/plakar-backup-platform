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
	"fmt"
	"io"
	"os"

	"github.com/PlakarKorp/plakar/snapshot/vfs"
	"golang.org/x/net/webdav"
)

type PlakarFS struct {
	vfsRoot *vfs.Filesystem
}

func (fs *PlakarFS) resolve(name string) (*vfs.Entry, error) {
	return fs.vfsRoot.GetEntry(name)
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
	entry, err := fs.resolve(name)
	if err != nil {
		return nil, err
	}
	return entry.Stat(), nil
}

func (fs *PlakarFS) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	entry, err := fs.resolve(name)
	if err != nil {
		return nil, err
	}
	if entry.Stat().IsDir() {
		return &PlakarDir{vfsRoot: fs.vfsRoot, entry: entry}, nil
	}
	reader, err := fs.vfsRoot.Open(name)
	if err != nil {
		return nil, err
	}
	return &PlakarFile{vfsRoot: fs.vfsRoot, reader: reader, info: entry.Stat()}, nil
}

type PlakarFile struct {
	vfsRoot *vfs.Filesystem
	reader  io.ReadCloser
	info    os.FileInfo
}

func (f *PlakarFile) Write([]byte) (int, error) {
	return 0, fmt.Errorf("cannot write to directory")
}

func (f *PlakarFile) Read(p []byte) (int, error) {
	return f.reader.Read(p)
}

func (f *PlakarFile) Close() error {
	return f.reader.Close()
}

func (f *PlakarFile) Seek(offset int64, whence int) (int64, error) {
	return 0, fmt.Errorf("seek not supported") // you could buffer if needed
}

func (f *PlakarFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, fmt.Errorf("not a directory")
}

func (f *PlakarFile) Stat() (os.FileInfo, error) {
	return f.info, nil
}

type PlakarDir struct {
	vfsRoot *vfs.Filesystem
	entry   *vfs.Entry
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
