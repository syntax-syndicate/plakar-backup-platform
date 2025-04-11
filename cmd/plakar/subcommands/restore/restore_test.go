package restore

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/snapshot"
	_ "github.com/PlakarKorp/plakar/snapshot/exporter/fs"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

func generateSnapshot(t *testing.T, bufOut *bytes.Buffer, bufErr *bytes.Buffer) (*repository.Repository, *snapshot.Snapshot) {
	repo := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockDir("another_subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
		ptesting.NewMockFile("subdir/to_exclude", 0644, "*/subdir/to_exclude\n"),
		ptesting.NewMockFile("another_subdir/bar.txt", 0644, "hello bar"),
	})
	return repo, snap
}

func TestExecuteCmdRestoreDefault(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	ctx := repo.AppContext()

	tmpToRestoreDir, err := os.MkdirTemp("", "tmp_to_restore")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpToRestoreDir)
	})

	ctx.CWD = tmpToRestoreDir
	args := []string{}
	// args := []string{tmpBackupDir + "/subdir/dummy.txt"}

	subcommand, err := parse_cmd_restore(repo.AppContext(), args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)
	require.Equal(t, "restore", subcommand.(*Restore).Name())

	status, err := subcommand.Execute(repo.AppContext(), repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// output should be something like:
	// 2025-02-25T21:35:42Z info: 31b0d219: OK ✓ /tmp/tmp_to_backup1287588797/another_subdir/bar
	// 2025-02-25T21:35:42Z info: 31b0d219: OK ✓ /tmp/tmp_to_backup1287588797/another_subdir
	// 2025-02-25T21:35:42Z info: 31b0d219: OK ✓ /tmp/tmp_to_backup1287588797/subdir/dummy.txt
	// 2025-02-25T21:35:42Z info: 31b0d219: OK ✓ /tmp/tmp_to_backup1287588797/subdir/foo.txt
	// 2025-02-25T21:35:42Z info: 31b0d219: OK ✓ /tmp/tmp_to_backup1287588797/subdir/to_exclude
	// 2025-02-25T21:35:42Z info: 31b0d219: OK ✓ /tmp/tmp_to_backup1287588797/subdir
	// 2025-02-25T21:35:42Z info: 31b0d219: OK ✓ /tmp/tmp_to_backup1287588797
	// 2025-02-25T21:35:42Z info: restore: restoration of 31b0d219:/ at /tmp/tmp_to_restore3971085618/plakar-2025-02-25T21:35:42Z completed successfully

	output := bufOut.String()
	lines := strings.Split(strings.Trim(output, "\n"), "\n")
	require.Equal(t, 8, len(lines))
	// last line should have the summary
	lastline := lines[len(lines)-1]
	require.Contains(t, lastline, "info: restore: restoration of")
}

func TestExecuteCmdRestoreSpecificSnapshot(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	// create one snapshot
	repo, snap := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	ctx := repo.AppContext()

	tmpToRestoreDir, err := os.MkdirTemp("", "tmp_to_restore")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpToRestoreDir)
	})

	ctx.CWD = tmpToRestoreDir
	indexId := snap.Header.GetIndexID()
	args := []string{fmt.Sprintf("%s", hex.EncodeToString(indexId[:]))}
	subcommand, err := parse_cmd_restore(repo.AppContext(), args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)
	require.Equal(t, "restore", subcommand.(*Restore).Name())

	status, err := subcommand.Execute(repo.AppContext(), repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// output should be something like:
	// 2025-02-25T21:35:42Z info: 31b0d219: OK ✓ /tmp/tmp_to_backup1287588797/another_subdir/bar
	// 2025-02-25T21:35:42Z info: 31b0d219: OK ✓ /tmp/tmp_to_backup1287588797/another_subdir
	// 2025-02-25T21:35:42Z info: 31b0d219: OK ✓ /tmp/tmp_to_backup1287588797/subdir/dummy.txt
	// 2025-02-25T21:35:42Z info: 31b0d219: OK ✓ /tmp/tmp_to_backup1287588797/subdir/foo.txt
	// 2025-02-25T21:35:42Z info: 31b0d219: OK ✓ /tmp/tmp_to_backup1287588797/subdir/to_exclude
	// 2025-02-25T21:35:42Z info: 31b0d219: OK ✓ /tmp/tmp_to_backup1287588797/subdir
	// 2025-02-25T21:35:42Z info: 31b0d219: OK ✓ /tmp/tmp_to_backup1287588797
	// 2025-02-25T21:35:42Z info: restore: restoration of 31b0d219:/ at /tmp/tmp_to_restore3971085618/plakar-2025-02-25T21:35:42Z completed successfully

	output := bufOut.String()
	lines := strings.Split(strings.Trim(output, "\n"), "\n")
	if len(lines) != 8 {
		t.Fatalf("Expected 8 lines; got %d; the content is:\n%s\n", len(lines), output)
	}
	require.Equal(t, 8, len(lines))
	// last line should have the summary
	lastline := lines[len(lines)-1]
	require.Contains(t, lastline, "info: restore: restoration of")
}
