//go:build windows

package plugins

import (
	"context"
	"errors"
)

func LoadBackends(ctx context.Context, pluginPath string) error {
	return errors.ErrUnsupported
}
