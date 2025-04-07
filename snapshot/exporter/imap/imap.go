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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/PlakarKorp/plakar/objects"

	"github.com/PlakarKorp/plakar/snapshot/exporter"

	imap2 "github.com/BrianLeishman/go-imap"
	"github.com/emersion/go-imap/client"
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
	p.maxConcurrency <- struct{}{}
	defer func() { <-p.maxConcurrency }()
	if pathname == "/" {
		return nil
	}
	err := p.client.Create(strings.ReplaceAll(pathname[1:], "/", "."))
	if err != nil {
		log.Printf("Failed to create directory %s: %v", strings.ReplaceAll(pathname[1:], "/", "."), err)
		return nil
	}
	return nil //todo: check error if err == "Mailbox already exists" return nil else return err
}

func (p *IMAPExporter) StoreFile(pathname string, fp io.Reader) error {

	p.maxConcurrency <- struct{}{}
	defer func() { <-p.maxConcurrency }()

	pathh, _ := strings.CutSuffix(pathname, "/"+path.Base(pathname))
	pathhh, _ := strings.CutPrefix(pathh, "/")
	folderName := strings.ReplaceAll(pathhh, "/", ".")
	log.Printf("Folder name: %s", folderName)
	_, err := p.client.Select(folderName, false)
	if err != nil {
		return fmt.Errorf("error selecting folder: %w", err)
	}

	emailRaw, err := io.ReadAll(fp)
	if err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}
	var email imap2.Email
	err = json.Unmarshal(emailRaw, &email)
	if err != nil {
		return fmt.Errorf("error unmarshalling email: %w", err)
	}

	var b bytes.Buffer
	b.WriteString("From: " + email.From.String() + "\r\n")
	b.WriteString("To: " + email.To.String() + "\r\n")
	b.WriteString("ReplyTo: " + email.ReplyTo.String() + "\r\n")
	b.WriteString("Subject: " + email.Subject + "\r\n")
	b.WriteString("Received: " + email.Received.String() + "\r\n")
	b.WriteString("Sent: " + email.Sent.String() + "\r\n")
	b.WriteString("CC: " + email.CC.String() + "\r\n")
	b.WriteString("BCC: " + email.BCC.String() + "\r\n")
	b.WriteString("\r\n")
	b.WriteString(email.Text)

	err = p.client.Append(folderName, email.Flags, email.Received, &b)
	if err != nil {
		return fmt.Errorf("error appending email to folder: %w", err)
	}
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
