package exporter

import (
	"fmt"
	"io"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/objects"
)

type Exporter interface {
	Root() string
	CreateDirectory(pathname string) error
	StoreFile(pathname string, fp io.Reader) error
	SetPermissions(pathname string, fileinfo *objects.FileInfo) error
	Close() error
}

var muBackends sync.Mutex
var backends map[string]func(appCtx *appcontext.AppContext, name string, config map[string]string) (Exporter, error) = make(map[string]func(appCtx *appcontext.AppContext, name string, config map[string]string) (Exporter, error))

func Register(name string, backend func(appCtx *appcontext.AppContext, name string, config map[string]string) (Exporter, error)) {
	muBackends.Lock()
	defer muBackends.Unlock()

	if _, ok := backends[name]; ok {
		log.Fatalf("backend '%s' registered twice", name)
	}
	backends[name] = backend
}

func Backends() []string {
	muBackends.Lock()
	defer muBackends.Unlock()

	ret := make([]string, 0)
	for backendName := range backends {
		ret = append(ret, backendName)
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i] < ret[j]
	})
	return ret
}

func NewExporter(ctx *appcontext.AppContext, config map[string]string) (Exporter, error) {
	location, ok := config["location"]
	if !ok {
		return nil, fmt.Errorf("missing location")
	}

	if strings.HasPrefix(location, "/") {
		location = "fs://" + location
	}

	muBackends.Lock()
	defer muBackends.Unlock()
	for name, backend := range backends {
		if strings.HasPrefix(location, name+":") {

			location = strings.TrimPrefix(location, name+"://")
			location = strings.TrimPrefix(location, name+":")
			config["location"] = location
			backendInstance, err := backend(ctx, name, config)
			if err != nil && name != "fs" {
				return nil, err
			} else if err != nil {
				if !filepath.IsAbs(location) {
					config["location"] = filepath.Join(ctx.CWD, location)
					if backendInstance, err = backend(ctx, name, config); err == nil {
						return backendInstance, nil
					}
				}
				return nil, err
			}
			return backendInstance, nil
		}
	}
	return nil, fmt.Errorf("unsupported exporter protocol")
}
