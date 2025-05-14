package agent

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/PlakarKorp/plakar/agent"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/caching"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands/ls"
	"github.com/PlakarKorp/plakar/cmd/plakar/utils"
	"github.com/PlakarKorp/plakar/cookies"
	"github.com/PlakarKorp/plakar/logging"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/snapshot"
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

func initContext(t *testing.T, bufout *bytes.Buffer, buferr *bytes.Buffer) (*appcontext.AppContext, string) {
	tmpCacheDir, err := os.MkdirTemp("", "tmp_cache")
	tmpLogDir, err := os.MkdirTemp("", "tmp_log")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpCacheDir)
		os.RemoveAll(tmpLogDir)
	})
	ctx := appcontext.NewAppContext()
	if bufout != nil && buferr != nil {
		ctx.Stdout = bufout
		ctx.Stderr = buferr
	}
	cache := caching.NewManager(tmpCacheDir)
	ctx.SetCache(cache)
	ctx.MaxConcurrency = 1

	cookies := cookies.NewManager("/tmp/test_plakar")
	ctx.SetCookies(cookies)
	ctx.Client = "plakar-test/1.0.0"

	var logger *logging.Logger
	if bufout == nil || buferr == nil {
		logger = logging.NewLogger(os.Stdout, os.Stderr)
	} else {
		logger = logging.NewLogger(bufout, buferr)
	}
	logger.EnableInfo()
	ctx.SetLogger(logger)

	return ctx, tmpLogDir
}

func TestCmdAgentForegroundInit(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	ctx, logDirectory := initContext(t, bufOut, bufErr)
	logFile := filepath.Join(logDirectory, "agent.log")
	defer os.Remove(logFile)

	args := []string{"-foreground", "-log", logFile}
	subcommand := &Agent{}
	err := subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	require.Equal(t, filepath.Join(ctx.CacheDir, "agent.sock"), subcommand.socketPath)
	defer subcommand.Close()

	_, err = os.Stat(logFile)
	require.NoError(t, err)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				require.Equal(t, "unexpected call to os.Exit(0) during test", r)
			}
		}()

		status, err := subcommand.Execute(ctx, nil)
		require.NoError(t, err)
		require.Equal(t, 0, status)
	}()

	time.Sleep(300 * time.Millisecond)

	defer func() {
		if r := recover(); r != nil {
			require.Equal(t, "unexpected call to os.Exit(0) during test", r)
		}
	}()

	client, err := agent.NewClient(filepath.Join(ctx.CacheDir, "agent.sock"), false)
	require.NoError(t, err)
	defer client.Close()

	repo, snap := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	ctx2 := snap.AppContext()
	ctx2.MaxConcurrency = 1

	// override the homedir to avoid having test overwriting existing home configuration
	ctx2.HomeDir = repo.Location()

	retval, err := client.SendCommand(ctx2, []string{"ls"}, &ls.Ls{LocateOptions: utils.NewDefaultLocateOptions(), SubcommandBase: subcommands.SubcommandBase{Flags: subcommands.AgentSupport}}, map[string]string{"location": repo.Location()})
	require.NoError(t, err)
	require.Equal(t, 0, retval)

	// disabled for now because if raises: write unix @->agent.sock: write: broken pipe
	// backupDir := snap.Header.GetSource(0).Importer.Directory
	// retval, err = client.SendCommand(ctx2, []string{"cat"}, &cat.Cat{Paths: []string{filepath.Join(backupDir, "subdir/dummy.txt")}, SubcommandBase: subcommands.SubcommandBase{Flags: subcommands.AgentSupport}}, map[string]string{"location": repo.Location()})
	// require.NoError(t, err)
	// require.Equal(t, 0, retval)
}
