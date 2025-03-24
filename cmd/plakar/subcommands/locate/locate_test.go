package locate

import (
	"bytes"
	"encoding/hex"
	"os"
	"strings"
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

func TestExecuteCmdLocateDefault(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	snap := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	ctx := snap.AppContext()
	ctx.MaxConcurrency = 1

	// override the homedir to avoid having test overwriting existing home configuration
	ctx.HomeDir = snap.Repository().Location()
	args := []string{"dummy.txt"}

	subcommand, err := parse_cmd_locate(ctx, snap.Repository(), args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)
	require.Equal(t, "locate", subcommand.(*Locate).Name())

	status, err := subcommand.Execute(ctx, snap.Repository())
	require.NoError(t, err)
	require.NotNil(t, status)

	// output should look like this
	// d92a4c73:/tmp/tmp_to_backup1424943315/subdir/dummy.txt

	output := bufOut.String()
	lines := strings.Split(strings.Trim(output, "\n"), "\n")
	require.Equal(t, 1, len(lines))
}

func TestExecuteCmdLocateWithSnapshotId(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	snap := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	ctx := snap.AppContext()
	ctx.MaxConcurrency = 1

	// override the homedir to avoid having test overwriting existing home configuration
	ctx.HomeDir = snap.Repository().Location()
	args := []string{"-snapshot", hex.EncodeToString(snap.Header.GetIndexShortID()), "dummy.txt"}

	subcommand, err := parse_cmd_locate(ctx, snap.Repository(), args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)
	require.Equal(t, "locate", subcommand.(*Locate).Name())

	status, err := subcommand.Execute(ctx, snap.Repository())
	require.NoError(t, err)
	require.NotNil(t, status)

	// output should look like this
	// d92a4c73:/tmp/tmp_to_backup1424943315/subdir/dummy.txt

	output := bufOut.String()
	lines := strings.Split(strings.Trim(output, "\n"), "\n")
	require.Equal(t, 1, len(lines))
}
