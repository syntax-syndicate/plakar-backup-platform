package ptar

import (
	"os"
	"path/filepath"
	"testing"

	_ "github.com/PlakarKorp/plakar/storage/backends/ptar"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

func TestExecuteCmdPtarDefault(t *testing.T) {
	repo := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	tmpSourceDir := ptesting.GenerateFiles(t, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockDir("another_subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
		ptesting.NewMockFile("subdir/to_exclude", 0644, "*/subdir/to_exclude\n"),
		ptesting.NewMockFile("another_subdir/bar.txt", 0644, "hello bar"),
	})

	args := []string{"--no-encryption", "--no-compression", filepath.Join(tmpSourceDir, "subdir")}

	subcommand := &Ptar{}
	err := subcommand.Parse(repo.AppContext(), args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(repo.AppContext(), repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

func TestExecuteCmdPtarWithSync(t *testing.T) {
	// Create source repository
	srcRepo := ptesting.GenerateRepository(t, nil, nil, nil)
	srcSnap := ptesting.GenerateSnapshot(t, srcRepo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockDir("another_subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
		ptesting.NewMockFile("subdir/to_exclude", 0644, "*/subdir/to_exclude\n"),
		ptesting.NewMockFile("another_subdir/bar.txt", 0644, "hello bar"),
	})
	defer srcSnap.Close()

	// Create destination repository
	dstRepo := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)

	args := []string{"--no-encryption", "--no-compression", "--sync-from", srcRepo.Location()}

	subcommand := &Ptar{}
	err := subcommand.Parse(dstRepo.AppContext(), args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(dstRepo.AppContext(), dstRepo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}
