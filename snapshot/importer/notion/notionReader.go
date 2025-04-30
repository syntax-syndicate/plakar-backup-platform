package notion

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type NotionReader struct {
	token           string
	pageID          string
	cursor          string
	buf             *bytes.Buffer
	done            bool
	blockOpen       bool
	first           bool
	wroteFirstBlock bool
}

func NewNotionReader(token, pageID string) *NotionReader {
	return &NotionReader{
		token:           token,
		pageID:          pageID,
		cursor:          "",
		buf:             new(bytes.Buffer),
		done:            false,
		blockOpen:       false,
		first:           true,
		wroteFirstBlock: false,
	}
}

func (nr *NotionReader) Read(p []byte) (int, error) {
	for nr.buf.Len() < len(p) && !nr.done {
		// First-time call: open the JSON array
		if nr.first {
			nr.buf.WriteByte('[')
			nr.blockOpen = true
			nr.first = false
		}

		// Fetch `pagesize` blocks
		url := fmt.Sprintf("%s/blocks/%s/children?page_size=%d", notionURL, nr.pageID, pageSize)
		if nr.cursor != "" {
			url += fmt.Sprintf("&start_cursor=%s", nr.cursor)
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return 0, err
		}
		req.Header.Set("Authorization", "Bearer "+nr.token)
		req.Header.Set("Notion-Version", "2022-06-28")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return 0, err
		}
		defer resp.Body.Close()

		var blockResp struct {
			Results    []json.RawMessage `json:"results"`
			HasMore    bool              `json:"has_more"`
			NextCursor string            `json:"next_cursor"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&blockResp); err != nil {
			return 0, err
		}

		//Write blocks to buffer
		for _, block := range blockResp.Results {
			// If this is not the first block, add a comma
			if nr.wroteFirstBlock {
				nr.buf.WriteByte(',')
			}
			nr.buf.Write(block)
			nr.wroteFirstBlock = true
		}
		//Prettified version (less efficient, more readable)
		//for _, block := range blockResp.Results {
		//	if nr.wroteFirstBlock {
		//		nr.buf.WriteByte(',')
		//		nr.buf.WriteByte('\n')
		//	}
		//	var prettyBlock bytes.Buffer
		//	if err := json.Indent(&prettyBlock, block, "  ", "  "); err != nil {
		//		return 0, fmt.Errorf("failed to prettify block: %w", err)
		//	}
		//	nr.buf.Write(prettyBlock.Bytes())
		//	nr.wroteFirstBlock = true
		//}

		// Check if there are more blocks
		if !blockResp.HasMore {
			nr.done = true
		} else {
			nr.cursor = blockResp.NextCursor
		}
	}

	// Close the JSON array if done
	if nr.done && nr.blockOpen {
		nr.buf.WriteByte(']')
		nr.blockOpen = false
	}

	// Read from the buffer
	n, err := nr.buf.Read(p)
	if n == 0 && nr.done {
		return 0, io.EOF
	}
	return n, err
}
