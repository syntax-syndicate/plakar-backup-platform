package rclone

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	stdpath "path"
	"strings"
	"sync/atomic"
	"time"

	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/snapshot/importer"

	_ "github.com/rclone/rclone/backend/all" // import all backends
	"github.com/rclone/rclone/librclone/librclone"
)

type RcloneImporter struct {
	remote string
	base   string

	ino uint64
}

func init() {
	importer.Register("rclone", NewRcloneImporter)
}

// NewRcloneImporter creates a new RcloneImporter instance. It expects the location
// to be in the format "remote:path/to/dir". The path is optional, but the remote
// storage location is required, so the colon separator is always expected.
func NewRcloneImporter(config map[string]string) (importer.Importer, error) {
	location := strings.TrimPrefix(config["location"], "rclone://")
	remote, base, found := strings.Cut(location, ":")
	if !found {
		return nil, fmt.Errorf("invalid location: %s. Expected format: remote:path/to/dir", location)
	}

	librclone.Initialize()

	return &RcloneImporter{
		remote: remote,
		base:   base,
	}, nil
}

func (p *RcloneImporter) Scan() (<-chan *importer.ScanResult, error) {
	results := make(chan *importer.ScanResult)
	go func() {
		defer close(results)

		p.generateBaseDirectories(results)
		p.scanRecursive(results, "")
	}()
	return results, nil
}

// getPathInBackup returns the full normalized path of a file within the backup.
//
// The resulting path is constructed by joining the base path of the backup (p.base)
// with the provided relative path. If the base path (p.base) is not absolute,
// it is treated as relative to the root ("/").
func (p *RcloneImporter) getPathInBackup(path string) string {
	path = stdpath.Join(p.base, path)

	if !stdpath.IsAbs(p.base) {
		path = "/" + path
	}

	return stdpath.Clean(path)
}

// generateBaseDirectories sends all parent directories of the base path in
// reverse order to the provided results channel.
//
// For example, if the base is "remote:/path/to/dir", this function generates
// the directories "/path/to/dir", "/path/to", "/path", and "/".
func (p *RcloneImporter) generateBaseDirectories(results chan *importer.ScanResult) {
	parts := generatePathComponents(p.getPathInBackup(""))

	for _, part := range parts {
		results <- importer.NewScanRecord(
			part,
			"",
			objects.NewFileInfo(
				path.Base(part),
				0,
				0700|os.ModeDir,
				time.Unix(0, 0).UTC(),
				0,
				atomic.AddUint64(&p.ino, 1),
				0,
				0,
				0,
			),
			nil,
		)
	}
}

// generatePathComponents is a helper function that returns a slice of strings
// containing all the hierarchical components of an absolute path, starting
// from the full path down to the root.
//
// The path given as an argument must be an absolute clean path within the
// backup.
//
// Example:
//
//	Input:  "/path/to/dir"
//	Output: []string{"/path/to/dir", "/path/to", "/path", "/"}
//
//	Input:  "/relative/path"
//	Output: []string{"/relative/path", "/relative", "/"}
//
//	Input:  "/"
//	Output: []string{"/"}
func generatePathComponents(path string) []string {
	components := []string{}
	tmp := path

	for {
		components = append(components, tmp)
		parent := stdpath.Dir(tmp)
		if parent == tmp { // Reached the root
			break
		}
		tmp = parent
	}
	return components
}

func (p *RcloneImporter) scanRecursive(results chan *importer.ScanResult, path string) {
	payload := map[string]interface{}{
		"fs":     fmt.Sprintf("%s:%s", p.remote, p.base),
		"remote": path,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		results <- importer.NewScanError(p.getPathInBackup(path), err)
		return
	}

	output, status := librclone.RPC("operations/list", string(jsonPayload))
	if status != http.StatusOK {
		results <- importer.NewScanError(p.getPathInBackup(path), fmt.Errorf("failed to list directory: %s", output))
		return
	}

	var response struct {
		List []struct {
			Path     string `json:"Path"`
			Name     string `json:"Name"`
			Size     int64  `json:"Size"`
			MimeType string `json:"MimeType"`
			ModTime  string `json:"ModTime"`
			IsDir    bool   `json:"isDir"`
			ID       string `json:"ID"`
		} `json:"list"`
	}

	err = json.Unmarshal([]byte(output), &response)
	if err != nil {
		results <- importer.NewScanError(p.getPathInBackup(path), err)
		return
	}

	for _, file := range response.List {
		// Should never happen, but just in case let's fallback to the Unix epoch
		parsedTime, err := time.Parse(time.RFC3339, file.ModTime)
		if err != nil {
			parsedTime = time.Unix(0, 0).UTC()
		}

		if file.IsDir {
			p.scanRecursive(results, file.Path)

			results <- importer.NewScanRecord(
				p.getPathInBackup(file.Path+"|||"+file.ID),
				"",
				objects.NewFileInfo(
					stdpath.Base(file.Name),
					0,
					0700|os.ModeDir,
					parsedTime,
					0,
					atomic.AddUint64(&p.ino, 1),
					0,
					0,
					0,
				),
				nil,
			)
		} else {
			filesize := file.Size

			// Hack: the importer is required to provide a size for the file. If
			// the size is not available when listing the directory, let's
			// attempt to download the file to retrieve it. This is a hack until
			// it becomes possible to return -1 as the size in the importer. It
			// has to be removed eventually, because it requires to download the
			// file twice when performing the backup.
			if file.Size < 0 {
				handle, err := p.NewReader(p.getPathInBackup(file.Path))
				if err != nil {
					results <- importer.NewScanError(p.getPathInBackup(path), err)
					continue
				}
				name := handle.(*AutoremoveTmpFile).Name()
				size, err := os.Stat(name)
				if err != nil {
					results <- importer.NewScanError(p.getPathInBackup(path), err)
					continue
				}

				handle.Close()

				filesize = size.Size()
			}

			fi := objects.NewFileInfo(
				stdpath.Base(file.Path),
				filesize,
				0600,
				parsedTime,
				1,
				atomic.AddUint64(&p.ino, 1),
				0,
				0,
				0,
			)

			results <- importer.NewScanRecord(
				p.getPathInBackup(file.Path+"|||"+file.ID),
				"",
				fi,
				nil,
			)
		}
	}
}

// AutoremoveTmpFile is a wrapper around an os.File that removes the file when it's closed.
type AutoremoveTmpFile struct {
	*os.File
}

func (file *AutoremoveTmpFile) Close() error {
	librclone.Finalize()
	defer os.Remove(file.Name())
	return file.File.Close()
}

func (p *RcloneImporter) NewReader(pathname string) (io.ReadCloser, error) {
	// pathname is an absolute path within the backup. Let's convert it to a
	// relative path to the base path.
	pathname, _, _ = strings.Cut(pathname, "|||")
	relativePath := strings.TrimPrefix(pathname, p.getPathInBackup(""))

	tmpFile, err := os.CreateTemp("", fmt.Sprintf("plakar_temp_*%s", path.Ext(relativePath)))
	if err != nil {
		return nil, err
	}
	tmpFile.Close()

	payload := map[string]string{
		"srcFs":     fmt.Sprintf("%s:%s", p.remote, p.base),
		"srcRemote": relativePath,

		"dstFs":     strings.TrimSuffix(tmpFile.Name(), path.Base(tmpFile.Name())),
		"dstRemote": path.Base(tmpFile.Name()),
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	body, status := librclone.RPC("operations/copyfile", string(jsonPayload))
	if status != http.StatusOK {
		return nil, fmt.Errorf("failed to copy file: %s", body)
	}

	tmpFile, err = os.Open(tmpFile.Name())
	if err != nil {
		return nil, err
	}

	return &AutoremoveTmpFile{tmpFile}, nil
}

func (p *RcloneImporter) Close() error {
	return nil
}

func (p *RcloneImporter) Root() string {
	return p.getPathInBackup("")
}

func (p *RcloneImporter) Origin() string {
	return p.remote
}

func (p *RcloneImporter) Type() string {
	return "rclone"
}

func (p *RcloneImporter) NewExtendedAttributeReader(pathname string, attribute string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("extended attributes are not supported on rclone")
}

func (p *RcloneImporter) GetExtendedAttributes(pathname string) ([]importer.ExtendedAttributes, error) {
	return nil, fmt.Errorf("extended attributes are not supported on rclone")
}
