package rclone

import (
	"encoding/json"
	"fmt"
	"github.com/PlakarKorp/plakar/snapshot/exporter"
	"io"
	"net/http"
	"os"
	stdpath "path"
	"strings"

	"github.com/PlakarKorp/plakar/objects"
	"github.com/rclone/rclone/librclone/librclone"
)

type GooglePhotoExporter struct {
	remote   string
	base     string
	provider string
}

func NewGooglePhotoExporter(config map[string]string) (exporter.Exporter, error) {
	provider, location, _ := strings.Cut(config["location"], "://")
	remote, base, found := strings.Cut(location, ":")

	if !found {
		return nil, fmt.Errorf("invalid location: %s. Expected format: remote:path/to/dir", location)
	}

	librclone.Initialize()

	return &GooglePhotoExporter{
		remote:   remote,
		base:     base,
		provider: provider,
	}, nil
}

func (p *GooglePhotoExporter) getPathInBackup(path string) string {
	path = stdpath.Join(p.base, path)

	if !stdpath.IsAbs(p.base) {
		path = "/" + path
	}

	return stdpath.Clean(path)
}

func (p *GooglePhotoExporter) Root() string {
	return p.getPathInBackup("")
}

// The operation mkdir is a no-op for Google Photos
func (p *GooglePhotoExporter) CreateDirectory(pathname string) error {
	return nil
}

func (p *GooglePhotoExporter) StoreFile(pathname string, fp io.Reader) error {
	tmpFile, err := os.CreateTemp("", "tempfile-*.tmp")
	if err != nil {
		return err
	}
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	_, err = io.Copy(tmpFile, fp)
	if err != nil {
		return err
	}

	relativePath := strings.TrimPrefix(pathname, p.getPathInBackup(""))
	if p.base != "" {
		relativePath = p.base + "/" + relativePath
	}

	payload := map[string]string{
		"srcFs":     "/",
		"srcRemote": tmpFile.Name(),
		"dstFs":     p.remote + ":",
		"dstRemote": func() string {
			if strings.HasPrefix(relativePath, "media/") {
				return "upload/" + stdpath.Base(relativePath)
			}
			if strings.HasPrefix(relativePath, "feature/") {
				return "album/FAVORITE/" + stdpath.Base(relativePath)
			}
			return relativePath
		}(),
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	body, resp := librclone.RPC("operations/copyfile", string(jsonPayload))

	if resp != http.StatusOK {
		return fmt.Errorf("failed to copy file: %s", body)
	}

	return nil
}

func (p *GooglePhotoExporter) SetPermissions(pathname string, fileinfo *objects.FileInfo) error {
	return nil
}

func (p *GooglePhotoExporter) Close() error {
	return nil
}
