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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
	done       chan struct{}
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
		done:       make(chan struct{}, 1),
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
	go func() {
		wg.Wait()
		p.done <- struct{}{}
		close(p.done)
	}()

	var wg2 sync.WaitGroup
	wg2.Add(1)
	go func() {
		defer wg2.Done()

		for {
			if len(p.done) == 1 {
				// all scan are done, check if there are any readers left
				if p.nReader == 0 && len(results) == 0 && len(p.notionChan) == 0 {
					return
				}
			}
			record := <-p.notionChan
			if record.EOF == true {
				p.nReader--
				continue
			}
			// do something with the record
			type block struct {
				ID          string `json:"id"`
				HasChildren bool   `json:"has_children"`
				Type        string `json:"type"`
			}
			var b block
			if err := json.Unmarshal(record.Block, &b); err != nil {
				results <- importer.NewScanError("", err)
				continue
			}
			log.Printf("block: %s, %b", b.ID, b.HasChildren)
			if b.HasChildren && b.Type != "child_page" {
				fInfo := objects.NewFileInfo(
					b.ID,
					0,
					os.ModeDir,
					time.Time{},
					0,
					0,
					0,
					0,
					0,
				)
				results <- importer.NewScanRecord(record.pathTo+"/"+b.ID, "", fInfo, nil)
				fInfo.Lmode = 0
				fInfo.Lname = "blocks.json"
				results <- importer.NewScanRecord(record.pathTo+"/"+b.ID+"/"+fInfo.Lname, "", fInfo, nil)
				p.nReader++
			}
		}
	}()

	go func() {
		wg2.Wait()

		fInfo := objects.NewFileInfo(
			"content.json",
			0,
			0,
			time.Time{},
			0,
			0,
			0,
			0,
			0,
		)
		results <- importer.NewScanRecord("/content.json", "", fInfo, nil) //this is WIP:
		// the of conent.json is to create each page in the root directory, just to start the process of restore,
		// it can either simply list the number of pages or have a notion compliant format made by hand with a special reader dedicated to this.

		close(results)
	}()
	return results, nil
}

func (p *NotionImporter) NewReader(pathname string) (io.ReadCloser, error) {
	id := path.Base(path.Dir(pathname))
	name := path.Base(pathname)
	var rd io.Reader
	var err error

	if name == "header.json" {
		rd, err = NewNotionReaderHeader(p.token, id)
	} else if name == "blocks.json" {
		rd, err = NewNotionReaderBlocks(p.token, id, path.Dir(pathname), p.notionChan)
	} else if name == "content.json" {
		for {
			if len(p.done) == 1 {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		buff := make([]byte, 0)
		buff = append(buff, []byte("[")...)
		for i, id := range topLevelPages {
			buff = append(buff, []byte("{\"parent\":{\"page_id\":\""+p.rootID+"\"},\"id\":\""+id+"\"}")...)
			if i == len(topLevelPages)-1 {
				buff = append(buff, []byte("]")...)
			} else {
				buff = append(buff, []byte(",")...)
			}
		}
		rd = bytes.NewReader(buff)
	} else {
		return nil, fmt.Errorf("unsupported file type: %s", name)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create Notion reader: %w", err)
	}
	return io.NopCloser(rd), nil
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
