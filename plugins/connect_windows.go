package plugins

import (
	"errors"

	"google.golang.org/grpc"
)

func connectPlugin(pluginPath string, config map[string]string) (grpc.ClientConnInterface, error) {
	return nil, errors.ErrUnsupported
}
