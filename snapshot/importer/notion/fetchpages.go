package notion

import (
	"bytes"
	"encoding/json"
	"fmt"
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
const pageSize = 1 // number of pages to fetch at once default is 100

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
	Properties struct {
		Title struct {
			Title []struct {
				Text struct {
					Content string `json:"content"` // The title text (later used to create the displayed name)
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

	//var prettyJSON bytes.Buffer
	//json.Indent(&prettyJSON, respBody, "", "  ")
	//log.Print("\n==================\n")
	//log.Print(prettyJSON.String())
	//log.Print("\n==================\n")

	var response SearchResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return err
	}

	//results <- importer.NewScanRecord()

	// Traitement des pages récupérées
	wg.Add(1)
	go func() {
		defer wg.Done()

		AddPagesToTree(response.Results, results)
		roots := GetRootNodes()
		PrintHierarchy(roots, 0)

		//for _, page := range response.Results {
		//	title := "(no title)"
		//	if len(page.Properties.Title.Title) > 0 {
		//		title = page.Properties.Title.Title[0].Text.Content
		//	}
		//	pageMap[page.ID] = PageInfo{ID: page.ID, Title: title}
		//	fmt.Printf("\n🔸 Page: %s | ID: %s\n", title, page.ID)
		//
		//	//fInfo := objects.NewFileInfo(
		//	//	page.ID,
		//	//	-1,
		//	//	0,
		//	//	time.Time{},
		//	//	0,
		//	//	0,
		//	//	0,
		//	//	0,
		//	//	0,
		//	//)
		//	//
		//	//results <- importer.NewScanRecord("/"+page.ID, "", fInfo, nil)
		//}
	}()

	// Récupération des pages suivantes si disponible
	if response.HasMore {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.fetchAllPages(response.NextCursor, results, wg)
		}()
	}

	return nil
}

type BlockResponse struct {
	//Results []Block `json:"results"`
	HasMore    bool   `json:"has_more"`
	NextCursor string `json:"next_cursor"`
}

type PageNode struct {
	Page            Page
	Children        []*PageNode
	Parent          *PageNode
	ConnectedToRoot bool
}

func PrintHierarchy(nodes []*PageNode, level int) {
	prefix := strings.Repeat("  ", level)
	for _, node := range nodes {
		title := "Untitled"
		if len(node.Page.Properties.Title.Title) > 0 {
			title = node.Page.Properties.Title.Title[0].Text.Content
		}
		fmt.Printf("%s- %s (ID: %s) (%v)\n", prefix, title, node.Page.ID, node.ConnectedToRoot)
		PrintHierarchy(node.Children, level+1)
	}
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
	results <- importer.NewScanRecord(GetPathToRoot(node)+"/content.json", "", objects.NewFileInfo("index.json", 0, 0, time.Time{}, 0, 0, 0, 0, 0), nil)
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

func GetRootNodes() []*PageNode {
	var roots []*PageNode
	for _, node := range nodeMap {
		if node.Parent == nil {
			roots = append(roots, node)
		}
	}
	return roots
}

//type RichTextMention struct {
//	Type string `json:"type"`
//	Page struct {
//		ID string `json:"id"`
//	} `json:"page"`
//}
//
//type RichText struct {
//	Type    string           `json:"type"`
//	Mention *RichTextMention `json:"mention,omitempty"`
//}
//
//type Paragraph struct {
//	RichText []RichText `json:"rich_text"`
//}
//
//type Block struct {
//	Type      string     `json:"type"`
//	Paragraph *Paragraph `json:"paragraph,omitempty"`
//}
//
//type ChildrenResponse struct {
//	Results []Block `json:"results"`
//}
//
//// ExtractMentionedPageIDs parses a Notion children response and extracts the page IDs from mentions
//func ExtractMentionedPageIDs(body []byte) ([]string, error) {
//	var children ChildrenResponse
//	if err := json.Unmarshal(body, &children); err != nil {
//		return nil, err
//	}
//
//	var pageIDs []string
//	for _, block := range children.Results {
//		if block.Type == "paragraph" && block.Paragraph != nil {
//			for _, rt := range block.Paragraph.RichText {
//				if rt.Type == "mention" && rt.Mention != nil && rt.Mention.Type == "page" {
//					pageIDs = append(pageIDs, rt.Mention.Page.ID)
//				}
//			}
//		}
//	}
//
//	return pageIDs, nil
//}

//func main() {
//	err := fetchAllPages("")
//	if err != nil {
//		fmt.Println("Erreur:", err)
//	}
//}
