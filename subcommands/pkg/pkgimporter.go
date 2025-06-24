/*
 * Copyright (c) 2025 Matthieu Masson <matthieu.masson@plakar.io>
 * Copyright (c) 2025 Omar Polo <omar.polo@plakar.io>
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

package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/snapshot/importer"
)

type pkgerImporter struct {
	files []string
}

func (imp *pkgerImporter) Origin() string {
	return ""
}

func (imp *pkgerImporter) Type() string {
	return "pkger"
}

func (imp *pkgerImporter) Root() string {
	return "/"
}

func (imp *pkgerImporter) Scan() (<-chan *importer.ScanResult, error) {
	ch := make(chan *importer.ScanResult, 1)

	go func() {
		defer close(ch)

		info := objects.NewFileInfo("/", 0, 0700|os.ModeDir, time.Unix(0, 0), 0, 0, 0, 0, 1)
		ch <- &importer.ScanResult{
			Record: &importer.ScanRecord{
				Pathname: "/",
				FileInfo: info,
			},
		}

		for _, f := range imp.files {
			fbase := filepath.Base(f)
			name := filepath.Join("/", fbase)

			fp, err := os.Open(f)
			if err != nil {
				ch <- importer.NewScanError(f, fmt.Errorf("Failed to open file"))
				break
			}

			fi, err := fp.Stat()
			if err != nil {
				ch <- importer.NewScanError(f, fmt.Errorf("Failed to stat file"))
				break
			}

			ch <- &importer.ScanResult{
				Record: &importer.ScanRecord{
					Pathname: name,
					FileInfo: objects.FileInfoFromStat(fi),
					Reader:   fp,
				},
			}
		}
	}()

	return ch, nil
}

func (imp *pkgerImporter) Close() error {
	return nil
}
