package notion

import (
	"bytes"
	"encoding/json"
	"fmt"
	notionConst "github.com/PlakarKorp/plakar/snapshot/importer/notion"
	"io"
	"log"
	"net/http"
	"path"
	"sync"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/snapshot/exporter"
)

type NotionExporter struct {
	token  string
	rootID string //TODO : change this to a user friendly name (e.g. "My Notion Page" instead of "1234567890abcdef")
	sync.Mutex
}

func init() {
	exporter.Register("notion", NewNotionExporter)
}

func NewNotionExporter(appCtx *appcontext.AppContext, name string, config map[string]string) (exporter.Exporter, error) {
	token, ok := config["token"]
	if !ok {
		return nil, fmt.Errorf("missing token in config")
	}
	rootID, ok := config["rootID"]
	if !ok {
		return nil, fmt.Errorf("missing rootID in config")
	}
	return &NotionExporter{
		token:  token,
		rootID: rootID, //rootID must be an existing page ID, this is the page where the files will be exported
	}, nil
}

func (p *NotionExporter) Root() string {
	return p.rootID
}

func (p *NotionExporter) CreateDirectory(pathname string) error {
	// Notion does not support creating directories
	return nil
}

var pageIDMap = map[string]string{}

func (p *NotionExporter) StoreFile(pathname string, fp io.Reader, size int64) error {

	var jsonData map[string]interface{}
	err := json.NewDecoder(fp).Decode(&jsonData)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Link the new page to the root page with new page ID
	if parent, ok := jsonData["parent"].(map[string]interface{}); ok && parent["type"] == "workspace" {
		// Link top-level pages to the root page
		delete(jsonData, "parent")
		jsonData["parent"] = map[string]interface{}{
			"type":    "page_id",
			"page_id": p.rootID,
		}
	} else if parent["type"] == "page_id" {
		// Replace the page_id with the new one
		parentID := parent["page_id"].(string)
		if id, ok := pageIDMap[parentID]; ok {
			parent["page_id"] = id
		} else {
			log.Printf("page ID %s not found in map", parentID)
			return fmt.Errorf("page ID %s not found in map", parentID)
		}
	} else if parent["type"] == "block_id" {
		// Replace the block_id with the new one
		blockID := parent["block_id"].(string)
		if id, ok := pageIDMap[blockID]; ok {
			parent["block_id"] = id
		} else {
			log.Printf("block ID %s not found in map", blockID)
			return fmt.Errorf("block ID %s not found in map", blockID)
		}
	} else { //TODO: add database parent type
		return fmt.Errorf("invalid parent type: %s", parent["type"])
	}

	var children []map[string]interface{}
	if c, ok := jsonData["children"].([]interface{}); ok {
		for _, child := range c {
			if childMap, ok := child.(map[string]interface{}); ok {
				children = append(children, childMap)
			}
		}
	} else {
		return fmt.Errorf("invalid type for children: %T", jsonData["children"])
	}

	newChildren := make([]map[string]interface{}, 0)
	for _, child := range children {
		newChild := make(map[string]interface{})
		newChild["object"] = child["object"]
		newChild["type"] = child["type"]
		newChild[newChild["type"].(string)] = child[child["type"].(string)]
		newChildren = append(newChildren, newChild)
	}

	jsonData["children"] = []interface{}{} // Clear the children array to handle it separately

	// Marshal the modified JSON data back to bytes
	data, err := json.Marshal(jsonData)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	//----------

	url := fmt.Sprintf("%s/pages", notionConst.NotionURL)
	req, err := http.NewRequest("POST", url, io.NopCloser(bytes.NewReader(data)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", notionConst.NotionVersionHeader)

	p.Lock()
	resp, err := http.DefaultClient.Do(req)
	p.Unlock()
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// debug
		log.Printf("failed to upload file: %d", resp.StatusCode)
		// Read the response body for more details
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}
		var prettyJSON bytes.Buffer
		err = json.Indent(&prettyJSON, b, "", "\t")
		if err != nil {
			return fmt.Errorf("failed to format error response: %w", err)
		}
		log.Printf("Error response: %s\n", prettyJSON.String())
		// end debug
		return fmt.Errorf("failed to store file: status code %d", resp.StatusCode)
	}

	jsonData = map[string]interface{}{}
	err = json.NewDecoder(resp.Body).Decode(&jsonData)
	if err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	newPageID := jsonData["id"].(string)
	pageIDMap[path.Base(path.Dir(pathname))] = newPageID

	//=============

	for _, child := range newChildren {
		if child["type"] == "child_page" {
			continue //TODO: add support for child pages
		}
		var payload = map[string]interface{}{
			"children": []interface{}{
				child,
			},
		}
		data, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}

		url = fmt.Sprintf("%s/blocks/%s/children", notionConst.NotionURL, newPageID)
		req, err = http.NewRequest("PATCH", url, io.NopCloser(bytes.NewReader(data)))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+p.token)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Notion-Version", notionConst.NotionVersionHeader)

		p.Lock()
		resp, err = http.DefaultClient.Do(req)
		p.Unlock()
		if err != nil {
			return fmt.Errorf("failed to execute request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			// debug
			log.Printf("failed to upload children: %d", resp.StatusCode)
			b, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to read response body: %w", err)
			}
			var prettyJSON bytes.Buffer
			err = json.Indent(&prettyJSON, b, "", "\t")
			if err != nil {
				return fmt.Errorf("failed to format error response: %w", err)
			}
			log.Printf("Error response: %s\n", prettyJSON.String())
			// end debug
			return fmt.Errorf("failed to store file: status code %d", resp.StatusCode)
		}
	}

	return nil
}

func (p *NotionExporter) SetPermissions(pathname string, fileinfo *objects.FileInfo) error {
	return nil
}

func (p *NotionExporter) Close() error {
	pageIDMap = map[string]string{}
	return nil
}
