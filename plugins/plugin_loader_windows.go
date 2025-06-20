//go:build windows

package plugins

import (
	"errors"

	"github.com/PlakarKorp/plakar/appcontext"
)

func Load(ctx *appcontext.AppContext, pluginPath, cacheDir string) error {
	return errors.ErrUnsupported
}
