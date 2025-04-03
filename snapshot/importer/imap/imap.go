package imap

import (
	"fmt"
	"github.com/PlakarKorp/plakar/objects"
	"io"
	"log"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/PlakarKorp/plakar/snapshot/importer"

	"github.com/BrianLeishman/go-imap"
)

type IMAPImporter struct {
	server   string
	port     int
	username string
	password string
	client   *imap.Dialer
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

	return &IMAPImporter{
		server:   parsedURL.Hostname(),
		port:     port,
		username: config["username"],
		password: config["password"],
		client:   nil,
	}, nil
}

func (p IMAPImporter) Origin() string {
	return p.server
}

func (p IMAPImporter) Type() string {
	return "imap"
}

func (p IMAPImporter) Root() string {
	return p.server
}

func connectToIMAP(server string, port int, username string, password string) (*imap.Dialer, error) {
	imap.RetryCount = 2

	c, err := imap.New(username, password, server, port)
	if err != nil {
		return nil, err
	}

	log.Printf("Connected to IMAP server %s", server)
	return c, nil
}

func (p IMAPImporter) Scan() (<-chan *importer.ScanResult, error) {
	c, err := connectToIMAP(p.server, p.port, p.username, p.password)
	if err != nil {
		return nil, err
	}
	p.client = c

	results := make(chan *importer.ScanResult, 1000)
	//-------
	folders, err := c.GetFolders()
	if err != nil {
		results <- importer.NewScanError("/", err)
		return nil, err
	}

	results <- importer.NewScanRecord("/", "", imapFileInfo("/", 4096, time.Now(), true), []string{})

	for _, f := range folders {
		err = c.SelectFolder(f)

		pathname := fmt.Sprintf("/%s", strings.ReplaceAll(f, ".", "/"))

		if err != nil {
			results <- importer.NewScanError(pathname, err)
			continue
		}

		uids, err := c.GetUIDs("ALL")
		if err != nil {
			results <- importer.NewScanError(pathname, err)
			continue
		}

		emails, err := c.GetEmails(uids...)
		if err != nil {
			results <- importer.NewScanError(pathname, err)
			continue
		}

		fileinfo := imapFileInfo(path.Base(pathname), 4096, time.Now(), true)
		results <- importer.NewScanRecord(pathname, "", fileinfo, []string{})

		for _, email := range emails {
			filepath := strings.ReplaceAll(f, ".", "/")
			pathname := fmt.Sprintf("%s/%d", filepath, email.UID)
			fileinfo := imapFileInfo(path.Base(pathname), int64(email.Size), email.Sent, false)
			results <- importer.NewScanRecord(pathname, "", fileinfo, []string{})
		}
	}

	return results, nil
}

func imapFileInfo(name string, size int64, t time.Time, isFolder bool) objects.FileInfo {
	mode := os.FileMode(0)
	if isFolder {
		mode = os.ModeDir
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
	parts := strings.Split(pathname, "/")
	imappath := ""
	if len(parts) > 1 {
		imappath = strings.Join(parts[:len(parts)-1], ".")
	}
	mailuid := parts[len(parts)-1]

	p.client, _ = connectToIMAP(p.server, p.port, p.username, p.password)
	defer p.client.Close()
	err := p.client.SelectFolder(imappath)
	if err != nil {
		return nil, err
	}

	uid, err := strconv.Atoi(mailuid)
	if err != nil {
		return nil, err
	}

	email, err := p.client.GetEmails(uid)
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
	return nil, fmt.Errorf("IMAPImporter.NewExtendedAttributeReader not implemented")
}

func (p IMAPImporter) GetExtendedAttributes(s string) ([]importer.ExtendedAttributes, error) {
	return nil, fmt.Errorf("IMAPImporter.GetExtendedAttributes not implemented")
}

func (p IMAPImporter) Close() error {
	return p.client.Close()
}
