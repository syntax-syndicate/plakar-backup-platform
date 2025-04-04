package imap

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/PlakarKorp/plakar/objects"

	"github.com/PlakarKorp/plakar/snapshot/importer"

	"github.com/BrianLeishman/go-imap"
)

type IMAPImporter struct {
	server   string
	port     int
	username string
	password string
	client   *imap.Dialer

	maxConcurrency chan struct{}
}

func init() {
	importer.Register("imap", NewIMAPImporter)
}

func NewIMAPImporter(config map[string]string) (importer.Importer, error) {
	port, err := strconv.Atoi(config["port"])
	if err != nil {
		return nil, fmt.Errorf("invalid port: %w", err)
	}
	parsedURL, err := url.Parse(config["location"])
	if err != nil {
		return nil, fmt.Errorf("invalid location: %w", err)
	}

	client, err := connectToIMAP(parsedURL.Hostname(), port, config["username"], config["password"])
	if err != nil {
		return nil, fmt.Errorf("failed to connect to IMAP server: %w", err)
	}

	return &IMAPImporter{
		server:         parsedURL.Hostname(),
		port:           port,
		username:       config["username"],
		password:       config["password"],
		client:         client,
		maxConcurrency: make(chan struct{}, 1),
	}, nil
}

func (p IMAPImporter) Origin() string {
	return p.server
}

func (p IMAPImporter) Type() string {
	return "imap"
}

func (p IMAPImporter) Root() string {
	return "/"
}

func connectToIMAP(server string, port int, username string, password string) (*imap.Dialer, error) {
	imap.RetryCount = 2

	c, err := imap.New(username, password, server, port)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (p IMAPImporter) Scan() (<-chan *importer.ScanResult, error) {
	//-------
	p.maxConcurrency <- struct{}{}

	folders, err := p.client.GetFolders()
	func() { <-p.maxConcurrency }()
	if err != nil {
		return nil, err
	}

	results := make(chan *importer.ScanResult, 1000)
	results <- importer.NewScanRecord("/", "", imapFileInfo("/", 4096, time.Now(), true), []string{})

	go func() {
		for _, f := range folders {
			p.maxConcurrency <- struct{}{}
			err = p.client.SelectFolder(f)
			func() { <-p.maxConcurrency }()

			pathname := fmt.Sprintf("/%s", strings.ReplaceAll(f, ".", "/"))

			if err != nil {
				results <- importer.NewScanError(pathname, err)
				continue
			}

			p.maxConcurrency <- struct{}{}
			uids, err := p.client.GetUIDs("ALL")
			func() { <-p.maxConcurrency }()
			if err != nil {
				results <- importer.NewScanError(pathname, err)
				continue
			}

			p.maxConcurrency <- struct{}{}
			emails, err := p.client.GetEmails(uids...)
			func() { <-p.maxConcurrency }()
			if err != nil {
				results <- importer.NewScanError(pathname, err)
				continue
			}

			fileinfo := imapFileInfo(path.Base(pathname), 4096, time.Now(), true)
			results <- importer.NewScanRecord(pathname, "", fileinfo, []string{})

			for _, email := range emails {
				filepath := strings.ReplaceAll(f, ".", "/")
				pathname := fmt.Sprintf("/%s/%d", filepath, email.UID)
				fileinfo := imapFileInfo(path.Base(pathname), int64(email.Size), email.Sent, false)
				results <- importer.NewScanRecord(pathname, "", fileinfo, []string{})
			}
		}
		close(results)
	}()

	return results, nil
}

func imapFileInfo(name string, size int64, t time.Time, isFolder bool) objects.FileInfo {
	mode := os.FileMode(0)
	if isFolder {
		mode = os.ModeDir | 0644
	} else {
		mode = os.FileMode(0644)
	}
	ssize := size
	if isFolder {
		ssize = 4096
	}

	return objects.FileInfo{
		Lname:      name,
		Lsize:      ssize,
		Lmode:      mode,
		LmodTime:   t,
		Ldev:       0,
		Lino:       0,
		Luid:       0,
		Lgid:       0,
		Lnlink:     0,
		Lusername:  "",
		Lgroupname: "",
		Flags:      0,
	}
}

func (p IMAPImporter) NewReader(pathname string) (io.ReadCloser, error) {
	p.maxConcurrency <- struct{}{}
	client := p.client
	func() { <-p.maxConcurrency }()

	parts := strings.Split(pathname, "/")
	imappath := ""
	if len(parts) > 1 {
		imappath = strings.Join(parts[1:len(parts)-1], ".")
	}
	mailuid := parts[len(parts)-1]
	_ = imappath

	p.maxConcurrency <- struct{}{}
	err := client.SelectFolder(imappath)
	func() { <-p.maxConcurrency }()

	if err != nil {
		return nil, err
	}

	uid, err := strconv.Atoi(mailuid)
	if err != nil {
		return nil, err
	}

	p.maxConcurrency <- struct{}{}
	emails, err := client.GetEmails(uid)
	func() { <-p.maxConcurrency }()

	if err != nil {
		return nil, err
	}

	for _, email := range emails {
		content, err := json.MarshalIndent(email, "", "  ")
		if content == nil {
			continue
		}
		if err != nil {
			return nil, err
		}
		scontent := fmt.Sprintf("%s", content)
		reader := strings.NewReader(scontent)
		return io.NopCloser(reader), nil
	}
	return nil, fmt.Errorf("no email found with UID %s", mailuid)
}

func (p IMAPImporter) NewExtendedAttributeReader(s string, s2 string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("extended attributes are not supported on IMAP")
}

func (p IMAPImporter) GetExtendedAttributes(s string) ([]importer.ExtendedAttributes, error) {
	return nil, fmt.Errorf("extended attributes are not supported on IMAP")
}

func (p IMAPImporter) Close() error {
	log.Printf("Disconnected to IMAP server")
	return p.client.Close()
}
