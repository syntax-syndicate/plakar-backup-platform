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

// package notion
//
// import (
//
//	"fmt"
//	"io"
//	"log"
//	"sync"
//
//	"github.com/PlakarKorp/plakar/appcontext"
//	"github.com/PlakarKorp/plakar/snapshot/importer"
//
// )
//
//	type NotionImporter struct {
//		token   string
//		rootID  string
//		// add more fields as needed (e.g., HTTP client, cache, etc.)
//	}
//
//	func init() {
//		importer.Register("notion", NewNotionImporter)
//	}
//
//	func NewNotionImporter(appCtx *appcontext.AppContext, name string, config map[string]string) (importer.Importer, error) {
//		token, ok := config["token"]
//		if !ok {
//			return nil, fmt.Errorf("missing token in config")
//		}
//
//		rootID, ok := config["root_id"]
//		if !ok {
//			return nil, fmt.Errorf("missing root_id in config")
//		}
//
//		return &NotionImporter{
//			token:  token,
//			rootID: rootID,
//		}, nil
//	}
//
//	func (p *NotionImporter) Scan() (<-chan *importer.ScanResult, error) {
//		results := make(chan *importer.ScanResult, 1000)
//		var wg sync.WaitGroup
//
//		wg.Add(1)
//		go func() {
//			defer wg.Done()
//			// TODO: Walk through Notion pages/blocks
//			// For each block/page, create a ScanResult:
//			// results <- importer.NewScanRecord("path", "", fileinfo, nil)
//			log.Println("Notion scanning not implemented yet")
//		}()
//
//		go func() {
//			wg.Wait()
//			close(results)
//		}()
//
//		return results, nil
//	}
//
//	func (p *NotionImporter) NewReader(pathname string) (io.ReadCloser, error) {
//		// TODO: Fetch page content or attachment from Notion
//		return nil, fmt.Errorf("NewReader not implemented for Notion")
//	}
//
//	func (p *NotionImporter) NewExtendedAttributeReader(pathname string, attribute string) (io.ReadCloser, error) {
//		return nil, fmt.Errorf("extended attributes are not supported on Notion")
//	}
//
//	func (p *NotionImporter) GetExtendedAttributes(pathname string) ([]importer.ExtendedAttributes, error) {
//		return nil, fmt.Errorf("extended attributes are not supported on Notion")
//	}
//
//	func (p *NotionImporter) Close() error {
//		// Nothing to close for now
//		return nil
//	}
//
//	func (p *NotionImporter) Root() string {
//		return p.rootID
//	}
//
//	func (p *NotionImporter) Origin() string {
//		return "notion.so"
//	}
//
//	func (p *NotionImporter) Type() string {
//		return "notion"
//	}
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const notionToken = "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"
const notionSearchURL = "https://api.notion.com/v1/search"

type NotionSearchResponse struct {
	Results    []Page `json:"results"`
	HasMore    bool   `json:"has_more"`
	NextCursor string `json:"next_cursor"`
}

type Page struct {
	Object     string `json:"object"`
	ID         string `json:"id"`
	Properties struct {
		Title struct {
			Title []struct {
				Text struct {
					Content string `json:"content"`
				} `json:"text"`
			} `json:"title"`
		} `json:"title"`
	} `json:"properties"`
}

type PageInfo struct {
	ID    string
	Title string
}

var pageMap = make(map[string]PageInfo)

func fetchAllPages(cursor string) error {
	bodyMap := map[string]interface{}{
		"page_size": 100,
	}
	if cursor != "" {
		bodyMap["start_cursor"] = cursor
	}
	bodyJSON, _ := json.Marshal(bodyMap)

	req, err := http.NewRequest("POST", notionSearchURL, bytes.NewBuffer(bodyJSON))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+notionToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", "2022-06-28")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	var result NotionSearchResponse
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		return err
	}

	// Traitement des pages récupérées
	for _, page := range result.Results {
		title := "(no title)"
		if len(page.Properties.Title.Title) > 0 {
			title = page.Properties.Title.Title[0].Text.Content
		}
		pageMap[page.ID] = PageInfo{ID: page.ID, Title: title}
		fmt.Printf("\n🔸 Page: %s | ID: %s\n", title, page.ID)
		fetchBlocks(page.ID) // Récupérer les blocs de la page
	}

	// Récupération des pages suivantes si disponible
	if result.HasMore {
		return fetchAllPages(result.NextCursor)
	}

	return nil
}

func fetchBlocks(blockID string) error {
	url := fmt.Sprintf("https://api.notion.com/v1/blocks/%s/children?page_size=100", blockID)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+notionToken)
	req.Header.Set("Notion-Version", "2022-06-28")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Extraction des IDs de pages mentionnées dans les blocs
	pageIDs, _ := ExtractMentionedPageIDs(body)
	for _, pageID := range pageIDs {
		// Affichage de la page mentionnée avec son titre et son ID
		pageTitle := pageMap[pageID].Title
		fmt.Printf("\n🔸 Page: %s | Page ID: %s | Mentioned in block ID: %s\n", pageTitle, pageID, blockID)
		// Récupérer les enfants de cette page mentionnée
		fetchBlocks(pageID) // Appel récursif pour récupérer les blocs de la page mentionnée
	}
	return nil
}

type RichTextMention struct {
	Type string `json:"type"`
	Page struct {
		ID string `json:"id"`
	} `json:"page"`
}

type RichText struct {
	Type    string           `json:"type"`
	Mention *RichTextMention `json:"mention,omitempty"`
}

type Paragraph struct {
	RichText []RichText `json:"rich_text"`
}

type Block struct {
	Type      string     `json:"type"`
	Paragraph *Paragraph `json:"paragraph,omitempty"`
}

type ChildrenResponse struct {
	Results []Block `json:"results"`
}

// ExtractMentionedPageIDs parses a Notion children response and extracts the page IDs from mentions
func ExtractMentionedPageIDs(body []byte) ([]string, error) {
	var children ChildrenResponse
	if err := json.Unmarshal(body, &children); err != nil {
		return nil, err
	}

	var pageIDs []string
	for _, block := range children.Results {
		if block.Type == "paragraph" && block.Paragraph != nil {
			for _, rt := range block.Paragraph.RichText {
				if rt.Type == "mention" && rt.Mention != nil && rt.Mention.Type == "page" {
					pageIDs = append(pageIDs, rt.Mention.Page.ID)
				}
			}
		}
	}

	return pageIDs, nil
}

func main() {
	err := fetchAllPages("")
	if err != nil {
		fmt.Println("Erreur:", err)
	}
}
