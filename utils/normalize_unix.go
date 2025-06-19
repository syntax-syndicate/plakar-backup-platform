//go:build !windows && !darwin

package utils

import "path/filepath"

func NormalizePath(p string) (string, error) {
	return filepath.Clean(p), nil
}
