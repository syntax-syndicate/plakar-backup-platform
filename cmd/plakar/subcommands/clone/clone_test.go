package clone

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/PlakarKorp/plakar/snapshot"
	_ "github.com/PlakarKorp/plakar/snapshot/exporter/fs"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

func generateSnapshot(t *testing.T, bufOut *bytes.Buffer, bufErr *bytes.Buffer) *snapshot.Snapshot {
	return ptesting.GenerateSnapshot(t, bufOut, bufErr, nil, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockDir("another_subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
		ptesting.NewMockFile("subdir/to_exclude", 0644, "*/subdir/to_exclude\n"),
		ptesting.NewMockFile("another_subdir/bar.txt", 0644, "hello bar"),
	})
}

func TestExecuteCmdClone(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	snap := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	ctx := snap.AppContext()
	ctx.MaxConcurrency = 1

	tmpDestinationDir, err := os.MkdirTemp("", "clone_destination")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpDestinationDir)
	})

	repo := snap.Repository()
	// override the homedir to avoid having test overwriting existing home configuration
	ctx.HomeDir = repo.Location()
	outputDir := filepath.Join(tmpDestinationDir, "clone_test")
	args := []string{"to", outputDir}

	subcommand, err := parse_cmd_clone(ctx, snap.Repository(), args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)
	require.Equal(t, "clone", subcommand.(*Clone).Name())

	status, err := subcommand.Execute(ctx, snap.Repository())
	require.NoError(t, err)
	require.NotNil(t, status)

	_, err = os.Stat(outputDir)
	require.NoError(t, err)
}
