package notion

import (
	"bytes"
	"encoding/json"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/snapshot/importer"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const notionURL = "https://api.notion.com/v1"
const notionSearchURL = notionURL + "/search"
const pageSize = 1 // Number of pages to fetch at once default is 100

type SearchResponse struct {
	Results    []Page `json:"results"`
	HasMore    bool   `json:"has_more"`
	NextCursor string `json:"next_cursor"`
}

type Page struct {
	Object string `json:"object"`
	ID     string `json:"id"`
	Parent struct {
		Type   string `json:"type"`
		PageID string `json:"page_id"`
	}
	//Properties struct {
	//	Title struct {
	//		Title []struct {
	//			Text struct {
	//				Content string `json:"content"` // The title text (later used to create the displayed name)
	//			} `json:"text"`
	//		} `json:"title"`
	//	} `json:"title"`
	//} `json:"properties"`
}

type PageInfo struct {
	ID    string
	Title string
}

func (p *NotionImporter) fetchAllPages(cursor string, results chan<- *importer.ScanResult, wg *sync.WaitGroup) error {
	bodyMap := map[string]interface{}{
		"page_size": pageSize,
	}
	if cursor != "" {
		bodyMap["start_cursor"] = cursor
	}
	bodyJSON, _ := json.Marshal(bodyMap)

	req, err := http.NewRequest("POST", notionSearchURL, bytes.NewBuffer(bodyJSON))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", "2022-06-28")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var response SearchResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return err
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		AddPagesToTree(response.Results, results)
	}()

	if response.HasMore {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.fetchAllPages(response.NextCursor, results, wg)
		}()
	}

	return nil
}

type PageNode struct {
	Page            Page
	Children        []*PageNode
	Parent          *PageNode
	ConnectedToRoot bool
}

// Global maps
var nodeMap = make(map[string]*PageNode)           // PageID -> PageNode
var waitingChildren = make(map[string][]*PageNode) // ParentID -> []*PageNode

func AddPagesToTree(pages []Page, results chan<- *importer.ScanResult) {
	for _, page := range pages {
		id := page.ID
		parentID := page.Parent.PageID

		// Get or create the node
		node, exists := nodeMap[id]
		if !exists {
			node = &PageNode{Page: page}
			nodeMap[id] = node
		} else {
			node.Page = page
		}

		// Determine if it's a root node
		if parentID == "" {
			propagateConnectionToRoot(node, results)
		} else {
			if parent, ok := nodeMap[parentID]; ok {
				// Attach to parent
				node.Parent = parent
				parent.Children = append(parent.Children, node)

				// Propagate connection if parent is already connected to root
				if parent.ConnectedToRoot {
					propagateConnectionToRoot(node, results)
				}
			} else {
				// Parent not yet known; defer
				waitingChildren[parentID] = append(waitingChildren[parentID], node)
			}
		}

		// Check if this node has waiting children
		if children, ok := waitingChildren[id]; ok {
			for _, child := range children {
				child.Parent = node
				node.Children = append(node.Children, child)

				// Propagate root connection if current node is connected
				if node.ConnectedToRoot {
					propagateConnectionToRoot(child, results)
				}
			}
			delete(waitingChildren, id)
		}
	}
}

func propagateConnectionToRoot(node *PageNode, results chan<- *importer.ScanResult) {
	if node.ConnectedToRoot {
		return
	}
	node.ConnectedToRoot = true
	results <- importer.NewScanRecord(GetPathToRoot(node), "", objects.NewFileInfo(node.Page.ID, 0, os.ModeDir, time.Time{}, 0, 0, 0, 0, 0), nil)
	results <- importer.NewScanRecord(GetPathToRoot(node)+"/content.json", "", objects.NewFileInfo("content.json", 0, 0, time.Time{}, 0, 0, 0, 0, 0), nil)
	for _, child := range node.Children {
		propagateConnectionToRoot(child, results)
	}
}

func GetPathToRoot(node *PageNode) string {
	var path []string
	current := node

	for current != nil {
		title := current.Page.ID
		path = append([]string{title}, path...)
		current = current.Parent
	}

	return "/" + strings.Join(path, "/")
}
