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
	"path"
	"sync"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/snapshot/importer"
)

type NotionImporter struct {
	token  string
	rootID string // TODO: take a look at this

	notionChan chan notionRecord
	nReader    int
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
		token:      token,
		rootID:     "/",
		notionChan: make(chan notionRecord, 1000),
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

		err := p.fetchAllPages("", results, &wg)
		if err != nil {
			results <- importer.NewScanError("", err) // TODO: handle error more gracefully
			return
		}
	}()

	// WIP:
	// the goal of this second go routine is to keep track of the number of readers,
	// and process the records from the readers as they are being read.
	// the main problem is that we need to be sure that all readers are done
	// before we close the results channel.
	// this is a bit tricky because we don't know how many readers there will be,
	// because:
	// 1. the scan records above should finish to be processed first.
	// 2. the routine below coould create new readers too, reapeating the problem.
	// main question is:
	// 1. how do we know when all readers are done? (p.nReader == 0 is not enough,
	//	  is the last reader done, or not even started?)
	// 2. how do we know when all records are processed?
	var wg2 sync.WaitGroup
	wg2.Add(1)
	go func() {
		defer wg2.Done()

		for {
			record := <-p.notionChan
			if record.EOF == true {
				p.nReader--
				continue
			}
			// do something with the record
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()
	return results, nil
}

func (p *NotionImporter) NewReader(pathname string) (io.ReadCloser, error) {
	p.nReader++
	file := path.Base(path.Dir(pathname))
	nRd, err := NewNotionReader(p.token, file, p.notionChan)
	if err != nil {
		return nil, fmt.Errorf("failed to create Notion reader: %w", err)
	}
	return io.NopCloser(nRd), nil
}

func (p *NotionImporter) NewExtendedAttributeReader(pathname string, attribute string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("extended attributes are not supported on Notion")
}

func (p *NotionImporter) GetExtendedAttributes(pathname string) ([]importer.ExtendedAttributes, error) {
	return nil, fmt.Errorf("extended attributes are not supported on Notion")
}

func (p *NotionImporter) Close() error {
	ClearNodeTree()
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
