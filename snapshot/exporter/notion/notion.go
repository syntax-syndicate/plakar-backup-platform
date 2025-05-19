package notion

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/snapshot/exporter"
	notionConst "github.com/PlakarKorp/plakar/snapshot/importer/notion"
	"io"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"sync"
	"time"
)

type NotionExporter struct {
	token  string
	rootID string //TODO : change this to a user friendly name (e.g. "My Notion Page" instead of "1234567890abcdef")
	sync.Mutex
	mapMutex sync.RWMutex // Added RWMutex for protecting pageIDMap
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

	log.Printf("NotionExporter: token: %s, rootID: %s", token, rootID)

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

func (p *NotionExporter) SetPermissions(pathname string, fileinfo *objects.FileInfo) error {
	return nil
}

var pageIDMap = map[string]string{}

func (p *NotionExporter) Close() error {
	pageIDMap = map[string]string{}
	return nil
}

func DebugResponse(resp *http.Response) {
	// debug
	log.Printf("failed to upload file: %d", resp.StatusCode)
	// Read the response body for more details
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	var prettyJSON bytes.Buffer
	err = json.Indent(&prettyJSON, b, "", "\t")
	if err != nil {
		return
	}
	log.Printf("Error response: %s\n", prettyJSON.String())
	// end
}

func (p *NotionExporter) AddBlock(payload []byte, pageID string) (string, error) {
	url := fmt.Sprintf("%s/blocks/%s/children", notionConst.NotionURL, pageID)
	req, err := http.NewRequest("PATCH", url, io.NopCloser(bytes.NewReader(payload)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", notionConst.NotionVersionHeader)

	p.Lock()
	resp, err := http.DefaultClient.Do(req)
	p.Unlock()
	if err != nil {
		log.Printf(url)
		DebugResponse(resp)
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		DebugResponse(resp)
		return "", fmt.Errorf("failed to store file: status code %d", resp.StatusCode)
	}

	jsonData := map[string]any{}
	err = json.NewDecoder(resp.Body).Decode(&jsonData)
	if err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}
	blockID := jsonData["results"].([]any)[0].(map[string]any)["id"].(string)
	return blockID, nil
}

func (p *NotionExporter) PostPage(payload []byte) (string, error) {
	url := fmt.Sprintf("%s/pages", notionConst.NotionURL)
	req, err := http.NewRequest("POST", url, io.NopCloser(bytes.NewReader(payload)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", notionConst.NotionVersionHeader)

	p.Lock()
	resp, err := http.DefaultClient.Do(req)
	p.Unlock()
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf(url)
		DebugResponse(resp)
		return "", fmt.Errorf("failed to store file: status code %d", resp.StatusCode)
	}

	jsonData := map[string]any{}
	err = json.NewDecoder(resp.Body).Decode(&jsonData)
	if err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return jsonData["id"].(string), nil
}

func (p *NotionExporter) PatchPage(payload []byte, pageID string) error {
	url := fmt.Sprintf("%s/pages/%s", notionConst.NotionURL, pageID)
	req, err := http.NewRequest("PATCH", url, io.NopCloser(bytes.NewReader(payload)))
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
		log.Printf(url)
		DebugResponse(resp)
		return fmt.Errorf("failed to store file: status code %d", resp.StatusCode)
	}

	return nil
}

// StoreFile2 is a temporary function to handle files in notion, it will replace StoreFile.
// StoreFile is outdated and will be removed in the future
func (p *NotionExporter) StoreFile(pathname string, fp io.Reader, size int64) error {

	filetype := filepath.Base(pathname)
	OldID := path.Base(path.Dir(pathname))

	if filetype == "content.json" { //POST empty pages to the root page

		var jsonData []map[string]any
		err := json.NewDecoder(fp).Decode(&jsonData)
		if err != nil {
			return fmt.Errorf("failed to unmarshal JSON: %w", err)
		}

		// Create a new page for each entry in the JSON array
		for _, entry := range jsonData {
			payload := map[string]any{
				"parent": map[string]any{
					"type":    "page_id",
					"page_id": p.rootID,
				},
				"properties": map[string]any{},
				"children":   []any{},
			}

			data, err := json.Marshal(payload)
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}

			pageID, err := p.PostPage(data)
			if err != nil {
				return fmt.Errorf("failed to post page: %w", err)
			}

			p.mapMutex.Lock()
			pageIDMap[entry["id"].(string)] = pageID
			p.mapMutex.Unlock()
		}

	} else if filetype == "header.json" { //PATCH header to the page OldID
		newID := func() string {
			timeout := time.After(5 * time.Second)
			for {
				select {
				case <-timeout:
					return "" // Timeout reached
				default:
					p.mapMutex.RLock()
					id, ok := pageIDMap[OldID]
					p.mapMutex.RUnlock()
					if ok {
						return id
					}
				}
			}
		}()
		if newID == "" {
			return fmt.Errorf("failed to find new ID for page %s", OldID)
		}

		var jsonData map[string]any
		err := json.NewDecoder(fp).Decode(&jsonData)
		if err != nil {
			return fmt.Errorf("failed to unmarshal JSON: %w", err)
		}
		jsonData2 := map[string]any{
			"properties": jsonData["properties"],
			"icon":       jsonData["icon"],
			"cover":      jsonData["cover"],
		}
		payload, err := json.Marshal(jsonData2)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		err = p.PatchPage(payload, newID)
		if err != nil {
			return fmt.Errorf("failed to patch page: %w", err)
		}

	} else if filetype == "blocks.json" { //PATCH blocks to the page OldID

		newID := func() string {
			timeout := time.After(5 * time.Second)
			for {
				select {
				case <-timeout:
					return "" // Timeout reached
				default:
					p.mapMutex.RLock()
					id, ok := pageIDMap[OldID]
					p.mapMutex.RUnlock()
					if ok {
						return id
					}
					time.Sleep(10 * time.Millisecond)
				}
			}
		}()

		var jsonData []map[string]any
		err := json.NewDecoder(fp).Decode(&jsonData)
		if err != nil {
			return fmt.Errorf("failed to unmarshal JSON: %w", err)
		}

		for _, block := range jsonData { //PATCH each block to the page
			if block["type"] == "child_page" {
				payload := map[string]any{}
				payload["parent"] = map[string]any{
					"type":    "page_id",
					"page_id": newID,
				}
				payload["properties"] = map[string]any{}
				payload["children"] = []any{}
				data, err := json.Marshal(payload)
				if err != nil {
					return fmt.Errorf("failed to marshal JSON: %w", err)
				}
				newPageID, err := p.PostPage(data)
				if err != nil {
					return fmt.Errorf("failed to post page: %w", err)
				}
				p.mapMutex.Lock()
				pageIDMap[block["id"].(string)] = newPageID
				p.mapMutex.Unlock()
			} else { //standard block
				payload := map[string]any{
					"children": []any{
						block,
					},
				}
				data, err := json.Marshal(payload)
				if err != nil {
					return fmt.Errorf("failed to marshal JSON: %w", err)
				}
				newBlockID, err := p.AddBlock(data, newID)
				if err != nil {
					return fmt.Errorf("failed to patch block: %w", err)
				}
				p.mapMutex.Lock()
				pageIDMap[block["id"].(string)] = newBlockID
				p.mapMutex.Unlock()
			}
		}

	} else {
		return fmt.Errorf("unsupported file type: %s", filetype)
	}

	return nil
}
