//go:build windows || darwin

package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizePath(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a nested directory structure inside the temp directory
	nestedDir := filepath.Join(tempDir, "TestDir", "SubDir")
	err := os.MkdirAll(nestedDir, 0755)
	require.NoError(t, err)

	// Create a file inside the nested directory
	filePath := filepath.Join(nestedDir, "testfile.txt")
	err = os.WriteFile(filePath, []byte("test content"), 0644)
	require.NoError(t, err)

	// Test case: Normalize an existing path
	normalizedPath, err := NormalizePath(filepath.Join(tempDir, "testdir", "subdir", "testfile.txt"))
	require.NoError(t, err)
	require.Equal(t, filePath, normalizedPath)

	// Test case: Normalize a non-existing path
	_, err = NormalizePath(filepath.Join(tempDir, "nonexistent", "path"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "path not found")

	// Test case: Normalize the root of the temp directory
	normalizedPath, err = NormalizePath(tempDir)
	require.NoError(t, err)
	require.Equal(t, tempDir, normalizedPath)
}
