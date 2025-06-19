//go:build windows || darwin

package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// NormalizePath returns a canonicalized version of the given absolute
// path.  macos and windows filesystems can be case-insensitive, so to
// disambiguate we have to walk the filesystem to get the right casing
// for each component.
func NormalizePath(path string) (string, error) {
	path = filepath.Clean(path)
	parts := strings.Split(path, string(filepath.Separator))[1:]

	if len(parts) == 0 || parts[0] == "" {
		if runtime.GOOS == "windows" {
			return "C:\\", nil
		}
		return "/", nil
	}

	normalizedPath := string(filepath.Separator)
	if runtime.GOOS == "windows" {
		normalizedPath = filepath.VolumeName(path) + "\\"
	}

	for _, part := range parts {
		if part == "" {
			continue
		}

		dirEntries, err := os.ReadDir(normalizedPath)
		if err != nil {
			return "", err
		}

		matched := false
		for _, entry := range dirEntries {
			if strings.EqualFold(entry.Name(), part) {
				normalizedPath = filepath.Join(normalizedPath, entry.Name())
				matched = true
				break
			}
		}

		if !matched {
			return "", fmt.Errorf("path not found: %s", path)
		}
	}

	return normalizedPath, nil
}
