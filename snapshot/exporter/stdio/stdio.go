/*
 * Copyright (c) 2025 Gilles Chehade <gilles@plakar.io>
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

package stdio

import (
	"io"
	"os"
	"strings"

	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/snapshot/exporter"
)

type StdioExporter struct {
	filePath string
}

func init() {
	exporter.Register("stdout", NewStdioExporter)
}

func NewStdioExporter(config map[string]string) (exporter.Exporter, error) {
	location := config["location"]
	location = strings.TrimPrefix("stdout://", location)

	return &StdioExporter{
		filePath: location,
	}, nil
}

func (p *StdioExporter) Root() string {
	return "/"
}

func (p *StdioExporter) CreateDirectory(pathname string) error {
	// can't mkdir on Stdio
	return nil
}

func (p *StdioExporter) StoreFile(pathname string, fp io.Reader) error {
	_, err := io.Copy(os.Stdout, fp)
	return err
}

func (p *StdioExporter) SetPermissions(pathname string, fileinfo *objects.FileInfo) error {
	// can't chown/chmod on Stdio
	return nil
}

func (p *StdioExporter) Close() error {
	return nil
}
