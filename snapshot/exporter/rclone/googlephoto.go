package rclone

import (
	"encoding/json"
	"fmt"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/snapshot/exporter"
	"io"
	"net/http"
	"os"
	stdpath "path"
	"strings"

	"github.com/rclone/rclone/librclone/librclone"
)

type GooglePhotoExporter struct {
	*RcloneExporter
}

func NewGooglePhotoExporter(appCtx *appcontext.AppContext, name string, config map[string]string) (exporter.Exporter, error) {
	exp, err := NewRcloneExporter(appCtx, name, config)
	if err != nil {
		return nil, err
	}
	return &GooglePhotoExporter{RcloneExporter: exp.(*RcloneExporter)}, nil
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
