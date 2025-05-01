package notion

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const notionURL = "https://api.notion.com/v1"
const pageSize = 1 // Number of pages to fetch at once default is 100
const notionVersionHeader = "2022-06-28"

type NotionReader struct {
	buf             *bytes.Buffer
	token           string
	pageID          string
	cursor          string
	done            bool
	blockOpen       bool
	first           bool
	wroteFirstBlock bool
}

func NewNotionReader(token, pageID string) *NotionReader {
	return &NotionReader{
		buf:             new(bytes.Buffer),
		token:           token,
		pageID:          pageID,
		cursor:          "",
		done:            false,
		blockOpen:       false,
		first:           true,
		wroteFirstBlock: false,
	}
}

type BlockResponse struct {
	Results    []json.RawMessage `json:"results"`
	HasMore    bool              `json:"has_more"`
	NextCursor string            `json:"next_cursor"`
}

func (nr *NotionReader) Read(p []byte) (int, error) {
	for nr.buf.Len() < len(p) && !nr.done {
		if nr.first {
			nr.openJSONArray()
		}

		blockResp, err := nr.fetchBlocks()
		if err != nil {
			return 0, fmt.Errorf("failed to fetch blocks: %w", err)
		}

		nr.writeBlocksToBuffer(blockResp.Results)

		if !blockResp.HasMore {
			nr.done = true
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

func (nr *NotionReader) openJSONArray() {
	nr.buf.WriteByte('[')
	nr.blockOpen = true
	nr.first = false
}

func (nr *NotionReader) closeJSONArray() {
	nr.buf.WriteByte(']')
	nr.blockOpen = false
}

func (nr *NotionReader) fetchBlocks() (*BlockResponse, error) {
	url := fmt.Sprintf("%s/blocks/%s/children?page_size=%d", notionURL, nr.pageID, pageSize)
	if nr.cursor != "" {
		url += fmt.Sprintf("&start_cursor=%s", nr.cursor)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+nr.token)
	req.Header.Set("Notion-Version", notionVersionHeader)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	var blockResp BlockResponse
	if err := json.NewDecoder(resp.Body).Decode(&blockResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &blockResp, nil
}

func (nr *NotionReader) writeBlocksToBuffer(blocks []json.RawMessage) {
	for _, block := range blocks {
		if nr.wroteFirstBlock {
			nr.buf.WriteByte(',')
		}
		nr.buf.Write(block)
		nr.wroteFirstBlock = true
	}
}
