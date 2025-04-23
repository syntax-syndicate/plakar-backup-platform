package clone

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/PlakarKorp/plakar/snapshot/exporter/fs"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

func TestExecuteCmdClone(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockDir("another_subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
		ptesting.NewMockFile("subdir/to_exclude", 0644, "*/subdir/to_exclude\n"),
		ptesting.NewMockFile("another_subdir/bar.txt", 0644, "hello bar"),
	})
	defer snap.Close()

	tmpDestinationDir, err := os.MkdirTemp("", "clone_destination")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpDestinationDir)
	})

	outputDir := filepath.Join(tmpDestinationDir, "clone_test")
	args := []string{"to", outputDir}

	subcommand := &Clone{}
	err = subcommand.Parse(repo.AppContext(), args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)
	require.Equal(t, "clone", subcommand.Name())

	status, err := subcommand.Execute(repo.AppContext(), repo)
	require.NoError(t, err)
	require.NotNil(t, status)

	_, err = os.Stat(outputDir)
	require.NoError(t, err)
}
