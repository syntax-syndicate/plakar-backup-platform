/*
 * Copyright (c) 2025 Your Name <your.email@example.com>
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

package notion

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/snapshot/importer"
)

type NotionImporter struct {
	token  string
	rootID string
}

func init() {
	importer.Register("notion", NewNotionImporter)
}

func NewNotionImporter(appCtx *appcontext.AppContext, name string, config map[string]string) (importer.Importer, error) {
	token, ok := config["token"]
	if !ok {
		return nil, fmt.Errorf("missing token in config")
	}
	return &NotionImporter{
		token:  token,
		rootID: "/",
	}, nil
}

func (p *NotionImporter) Scan() (<-chan *importer.ScanResult, error) {
	results := make(chan *importer.ScanResult, 1000)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		fInfo := objects.NewFileInfo(
			"/",
			0,
			os.ModeDir,
			time.Time{},
			0,
			0,
			0,
			0,
			0,
		)

		results <- importer.NewScanRecord("/", "", fInfo, nil)

		p.fetchAllPages("", results, &wg)
	}()
	go func() {
		wg.Wait()
		close(results)
	}()
	return results, nil
}

func (p *NotionImporter) NewReader(pathname string) (io.ReadCloser, error) {
	return p.fetchBlocks(strings.TrimPrefix(pathname, "/"))
}

func (p *NotionImporter) NewExtendedAttributeReader(pathname string, attribute string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("extended attributes are not supported on Notion")
}

func (p *NotionImporter) GetExtendedAttributes(pathname string) ([]importer.ExtendedAttributes, error) {
	return nil, fmt.Errorf("extended attributes are not supported on Notion")
}

func (p *NotionImporter) Close() error {
	// Nothing to close for now
	return nil
}

func (p *NotionImporter) Root() string {
	return p.rootID
}

func (p *NotionImporter) Origin() string {
	return "notion.so"
}

func (p *NotionImporter) Type() string {
	return "notion"
}
