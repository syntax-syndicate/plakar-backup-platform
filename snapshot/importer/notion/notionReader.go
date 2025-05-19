package notion

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	NotionURL           = "https://api.notion.com/v1"
	PageSize            = 1 // Number of pages to fetch at once default is 100
	NotionVersionHeader = "2022-06-28"
)

type NotionReader struct {
	buf             *bytes.Buffer
	token           string
	pageID          string
	path            string
	isBlock         bool
	cursor          string
	done            bool
	blockOpen       bool
	first           bool
	wroteFirstBlock bool
	recordChan      chan<- notionRecord // Channel to send records
}

type notionRecord struct {
	Block  json.RawMessage
	pathTo string
	EOF    bool
}

type BlockResponse struct {
	Results    []json.RawMessage `json:"results"`
	HasMore    bool              `json:"has_more"`
	NextCursor string            `json:"next_cursor"`
}

func fetchFromURL[T any](url, token string) (*T, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Notion-Version", NotionVersionHeader)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	var result T
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

//---------------
// lets break down the current NotionReader into two:
// 1. NotionReaderHeader: This will be used to read the header of the page
// 2. NotionReaderBlocks: This will be used to read the blocks of the page

type NotionReaderHeader struct {
	buf    *bytes.Buffer
	token  string
	pageID string
}

func NewNotionReaderHeader(token, pageID string) (*NotionReaderHeader, error) {
	nr := &NotionReaderHeader{
		buf:    new(bytes.Buffer),
		token:  token,
		pageID: pageID,
	}
	pageHeader, err := nr.fetchPageHeader()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page header: %w", err)
	}
	var mapp map[string]interface{}
	if err = json.Unmarshal(*pageHeader, &mapp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal page header: %w", err)
	}
	delete(mapp, "request_id")
	b, err := json.Marshal(mapp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal page header: %w", err)
	}
	nr.buf.Write(b)
	return nr, nil
}

func (nr *NotionReaderHeader) fetchPageHeader() (*json.RawMessage, error) {
	url := fmt.Sprintf("%s/pages/%s", NotionURL, nr.pageID)
	return fetchFromURL[json.RawMessage](url, nr.token)
}

func (nr *NotionReaderHeader) Read(p []byte) (int, error) {
	n, err := nr.buf.Read(p)
	if n == 0 && err == io.EOF {
		return 0, io.EOF
	}
	return n, err
}

type NotionReaderBlocks struct {
	buf             *bytes.Buffer
	token           string
	pageID          string
	path            string
	cursor          string
	done            bool
	blockOpen       bool
	first           bool
	wroteFirstBlock bool
	recordChan      chan<- notionRecord // Channel to send records
}

func NewNotionReaderBlocks(token, pageID, path string, recordChan chan<- notionRecord) (*NotionReaderBlocks, error) {
	nRd := &NotionReaderBlocks{
		buf:             new(bytes.Buffer),
		token:           token,
		pageID:          pageID,
		path:            path,
		cursor:          "",
		done:            false,
		blockOpen:       false,
		first:           true,
		wroteFirstBlock: false,
		recordChan:      recordChan,
	}

	nRd.buf.WriteString("[")
	return nRd, nil
}

func (nr *NotionReaderBlocks) fetchBlocks() (*BlockResponse, error) {
	url := fmt.Sprintf("%s/blocks/%s/children?page_size=%d", NotionURL, nr.pageID, PageSize)
	if nr.cursor != "" {
		url += fmt.Sprintf("&start_cursor=%s", nr.cursor)
	}
	return fetchFromURL[BlockResponse](url, nr.token)
}

func (nr *NotionReaderBlocks) Read(p []byte) (int, error) {
	for nr.buf.Len() < len(p) && !nr.done {
		if nr.first {
			nr.openJSONArray()
		}

		blockResp, err := nr.fetchBlocks()
		if err != nil {
			return 0, fmt.Errorf("failed to fetch blocks: %w", err)
		}

		nr.writeBlocksToBuffer(blockResp.Results)

		// Send each block as a notionRecord to the channel
		for _, block := range blockResp.Results {
			nr.recordChan <- notionRecord{Block: block, EOF: false, pathTo: nr.path}
		}

		if !blockResp.HasMore {
			nr.done = true
			// Send EOF record to indicate completion
			nr.recordChan <- notionRecord{EOF: true}
		} else {
			nr.cursor = blockResp.NextCursor
		}
	}

	if nr.done && nr.blockOpen {
		nr.closeJSONArray()
	}

	n, err := nr.buf.Read(p)
	if n == 0 && nr.done {
		return 0, io.EOF
	}
	return n, err
}

func (nr *NotionReaderBlocks) openJSONArray() {
	nr.blockOpen = true
	nr.first = false
}

func (nr *NotionReaderBlocks) closeJSONArray() {
	// Close the array and add the last brace that was removed
	nr.buf.Write([]byte("]"))
	nr.blockOpen = false
}

func (nr *NotionReaderBlocks) writeBlocksToBuffer(blocks []json.RawMessage) {
	for _, block := range blocks {
		if nr.wroteFirstBlock {
			nr.buf.WriteByte(',')
		}
		nr.buf.Write(block)
		nr.wroteFirstBlock = true
	}
}

type notionReaderFile struct {
	headerReader *NotionReaderHeader
	blockReader  *NotionReaderBlocks
}

// NewNotionReaderFile creates a new notionReaderFile instance
func NewNotionReaderFile(token, pageID, path string, recordChan chan<- notionRecord) (*notionReaderFile, error) {
	// Create the header reader
	headerReader, err := NewNotionReaderHeader(token, pageID)
	if err != nil {
		return nil, fmt.Errorf("failed to create NotionReaderHeader: %w", err)
	}
	//remove the closing bracket from the header and add the 'children' key
	headerReader.buf.Truncate(headerReader.buf.Len() - 1)
	headerReader.buf.WriteString(",\"children\":")

	// Create the block reader
	blockReader, err := NewNotionReaderBlocks(token, pageID, path, recordChan)
	if err != nil {
		return nil, fmt.Errorf("failed to create NotionReaderBlocks: %w", err)
	}

	return &notionReaderFile{
		headerReader: headerReader,
		blockReader:  blockReader,
	}, nil
}

// Read reads from the header first, then falls back to the block reader
func (nrf *notionReaderFile) Read(p []byte) (int, error) {
	// Attempt to read from the header reader
	n, err := nrf.headerReader.Read(p)
	if n > 0 || err != io.EOF {
		return n, err
	}

	// Fallback to the block reader
	n, err = nrf.blockReader.Read(p)
	if n > 0 || err != io.EOF {
		return n, err
	}

	if err == io.EOF {
		p[0] = '}'
		return 1, io.EOF
	}

	return 0, err
}
