package imap

import (
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
		maxConcurrency: make(chan struct{}, 4),
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

	//log.Printf("Connected to IMAP server %s", server)
	return c, nil
}

func (p IMAPImporter) Scan() (<-chan *importer.ScanResult, error) {
	//-------
	folders, err := p.client.GetFolders()
	if err != nil {
		return nil, err
	}

	results := make(chan *importer.ScanResult, 1000)
	results <- importer.NewScanRecord("/", "", imapFileInfo("/", 4096, time.Now(), true), []string{})

	go func() {
		for _, f := range folders {
			err = p.client.SelectFolder(f)

			pathname := fmt.Sprintf("/%s", strings.ReplaceAll(f, ".", "/"))

			if err != nil {
				results <- importer.NewScanError(pathname, err)
				continue
			}

			uids, err := p.client.GetUIDs("ALL")
			if err != nil {
				results <- importer.NewScanError(pathname, err)
				continue
			}

			emails, err := p.client.GetEmails(uids...)
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
	//return nil, fmt.Errorf("IMAP does not support reading files")
	p.maxConcurrency <- struct{}{}
	defer func() { <-p.maxConcurrency }()

	client, err := connectToIMAP(p.server, p.port, p.username, p.password) //not satisfied with connecting for every mails
	if err != nil {
		return nil, err
	}
	defer client.Close()

	parts := strings.Split(pathname, "/")
	//log.Printf("IMAPImporter.NewReader: %s", pathname)
	imappath := ""
	if len(parts) > 1 {
		imappath = strings.Join(parts[1:len(parts)-1], ".")
	}
	mailuid := parts[len(parts)-1]
	_ = imappath

	err = client.SelectFolder(imappath)
	if err != nil {
		return nil, err
	}

	uid, err := strconv.Atoi(mailuid)
	if err != nil {
		return nil, err
	}

	email, err := client.GetEmails(uid)
	if err != nil {
		return nil, err
	}
	if len(email) != 1 {
		return nil, fmt.Errorf("email not found")
	}

	content := fmt.Sprintf("%v", email)
	reader := strings.NewReader(content)
	return io.NopCloser(reader), nil

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
