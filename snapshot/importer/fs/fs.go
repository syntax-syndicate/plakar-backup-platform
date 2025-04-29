/*
 * Copyright (c) 2023 Gilles Chehade <gilles@poolp.org>
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

package fs

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/snapshot/importer"
	"github.com/pkg/xattr"
)

type FSImporter struct {
	rootDir string
}

func init() {
	importer.Register("fs", NewFSImporter)
}

func NewFSImporter(appCtx *appcontext.AppContext, name string, config map[string]string) (importer.Importer, error) {
	location := config["location"]

	if !path.IsAbs(location) {
		return nil, fmt.Errorf("not an absolute path %s", location)
	}

	location = path.Clean(location)

	return &FSImporter{
		rootDir: location,
	}, nil
}

func (p *FSImporter) Origin() string {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "localhost"
	}
	return hostname
}

func (p *FSImporter) Type() string {
	return "fs"
}

func (p *FSImporter) Scan() (<-chan *importer.ScanResult, error) {
	results := make(chan *importer.ScanResult, 1000)
	go p.walkDir_walker(results, p.rootDir, 256)
	return results, nil
}

func (f *FSImporter) realpathFollow(path string) (resolved string, err error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		realpath, err := os.Readlink(path)
		if err != nil {
			return "", err
		}

		if !filepath.IsAbs(realpath) {
			realpath = filepath.Join(filepath.Dir(path), realpath)
		}
		path = realpath
	}

	return path, nil
}

func (p *FSImporter) NewReader(pathname string) (io.ReadCloser, error) {
	if pathname[0] == '/' && runtime.GOOS == "windows" {
		pathname = pathname[1:]
	}
	return os.Open(pathname)
}

func (p *FSImporter) NewExtendedAttributeReader(pathname string, attribute string) (io.ReadCloser, error) {
	if pathname[0] == '/' && runtime.GOOS == "windows" {
		pathname = pathname[1:]
	}

	data, err := xattr.Get(pathname, attribute)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (p *FSImporter) Close() error {
	return nil
}

func (p *FSImporter) Root() string {
	return p.rootDir
}
