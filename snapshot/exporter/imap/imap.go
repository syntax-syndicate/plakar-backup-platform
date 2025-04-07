/*
 * Copyright (c) 2025 Gilles Chehade <gilles@plakar.io>
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

package imap

import (
	//"encoding/json"
	"fmt"
	"io"
	"net/url"
	//"log"
	"strconv"
	//"bytes"

	"github.com/PlakarKorp/plakar/objects"

	"github.com/PlakarKorp/plakar/snapshot/exporter"

	//"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	//"github.com/emersion/go-message"
	//"github.com/emersion/go-message/textproto"
)

type IMAPExporter struct {
	server   string
	port     int
	username string
	password string
	client   *client.Client

	maxConcurrency chan struct{}
}

func init() {
	exporter.Register("imap", NewIMAPexporter)
}

func connectToIMAP(server string, port int, username string, password string) (*client.Client, error) {
	// Connect to the server

	c, err := client.DialTLS(fmt.Sprintf("%s:%d", server, port), nil)
	if err != nil {
		return nil, err
	}

	// Login
	err = c.Login(username, password)
	if err != nil {
		c.Close()
		return nil, err
	}

	//log.Printf("Connected to IMAP server %s", server)
	return c, nil
}

func NewIMAPexporter(config map[string]string) (exporter.Exporter, error) {
	port, err := strconv.Atoi(config["port"])
	if err != nil {
		return nil, fmt.Errorf("invalid port: %w", err)
	}
	parsedURL, err := url.Parse(config["location"])
	if err != nil {
		return nil, fmt.Errorf("invalid location: %w", err)
	}

	c, err := connectToIMAP(parsedURL.Hostname(), port, config["username"], config["password"])
	if err != nil {
		return nil, fmt.Errorf("failed to connect to IMAP server: %w", err)
	}

	return &IMAPExporter{
		server:         parsedURL.Hostname(),
		port:           port,
		username:       config["username"],
		password:       config["password"],
		client:         c,
		maxConcurrency: make(chan struct{}, 1),
	}, nil
}

func (p *IMAPExporter) Root() string {
	return "/"
}

func (p *IMAPExporter) CreateDirectory(pathname string) error {
	/*err := p.client.Create(pathname)
	if err != nil {
		fmt.Println("Trouble creating folder :", err)
	} else {
		fmt.Println("Folder created :", pathname)
	}*/
	return nil
}

func (p *IMAPExporter) StoreFile(pathname string, fp io.Reader) error {
	/*folderName := pathname
	_, err := p.client.Select(folderName, false)
	if err != nil {
		return fmt.Errorf("error selecting folder: %w", err)
	}

	var buf bytes.Buffer
	header := textproto.Header{}
	header.Set("From", "user@example.com")
	header.Set("To", "recipient@example.com")
	header.Set("Subject", "Email added to IMAP")

	writer, err := message.CreateWriter(&buf, "text/plain")
	if err != nil {
		return fmt.Errorf("error creating message writer: %w", err)
	}

	part, err := writer.CreatePart(message.Header{Header: header})
	if err != nil {
		return fmt.Errorf("error creating message part: %w", err)
	}

	emailContent, err := json.Marshal(fp) // TODO: pqrse the json
	if emailContent != nil {
		_, err = io.Copy(part, emailContent)
		if err != nil {
			return fmt.Errorf("error writing email content: %w", err)
		}
	}

	part.Close()
	writer.Close()

	appendFlags := []string{imap.SeenFlag}
	err = p.client.Append(folderName, appendFlags, bytes.NewReader(buf.Bytes()), nil)
	if err != nil {
		return fmt.Errorf("error adding email: %w", err)
	}*/
	return nil
}



func (p *IMAPExporter) SetPermissions(pathname string, fileinfo *objects.FileInfo) error {
	return nil
}

func (p *IMAPExporter) Close() error {
	if p.client != nil {
		return p.client.Close()
	}
	return nil
}
