package help

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
	"testing"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/caching"
	"github.com/PlakarKorp/plakar/cookies"
	"github.com/PlakarKorp/plakar/hashing"
	"github.com/PlakarKorp/plakar/logging"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/resources"
	"github.com/PlakarKorp/plakar/storage"
	bfs "github.com/PlakarKorp/plakar/storage/backends/fs"
	"github.com/PlakarKorp/plakar/versioning"
	"github.com/stretchr/testify/require"
)

func TestParseCmdHelpDefault(t *testing.T) {
	// Create a pipe to capture stdout
	old := os.Stdout
	r1, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	// Restore stdout after the test
	defer func() {
		os.Stdout = old
		w.Close()
	}()

	// Use a WaitGroup to wait for the read to complete
	var wg sync.WaitGroup
	wg.Add(1)

	// Capture the output in a buffer
	var buf bytes.Buffer
	go func() {
		defer wg.Done()
		io.Copy(&buf, r1)
	}()

	tmpRepoDirRoot, err := os.MkdirTemp("", "tmp_repo")
	require.NoError(t, err)
	tmpRepoDir := fmt.Sprintf("%s/repo", tmpRepoDirRoot)
	tmpCacheDir, err := os.MkdirTemp("", "tmp_cache")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpCacheDir)
		os.RemoveAll(tmpRepoDir)
		os.RemoveAll(tmpRepoDirRoot)
	})

	ctx := appcontext.NewAppContext()

	// create a storage
	r, err := bfs.NewStore(ctx, "fs", map[string]string{"location": tmpRepoDir})
	require.NotNil(t, r)
	require.NoError(t, err)
	config := storage.NewConfiguration()
	serialized, err := config.ToBytes()
	require.NoError(t, err)

	hasher := hashing.GetHasher(hashing.DEFAULT_HASHING_ALGORITHM)
	wrappedConfigRd, err := storage.Serialize(hasher, resources.RT_CONFIG, versioning.GetCurrentVersion(resources.RT_CONFIG), bytes.NewReader(serialized))
	require.NoError(t, err)

	wrappedConfig, err := io.ReadAll(wrappedConfigRd)
	require.NoError(t, err)

	err = r.Create(ctx, wrappedConfig)
	require.NoError(t, err)

	// open the storage to load the configuration
	r, serializedConfig, err := storage.Open(ctx, map[string]string{"location": tmpRepoDir})
	require.NoError(t, err)

	// create a repository
	cache := caching.NewManager(tmpCacheDir)
	ctx.SetCache(cache)

	cookies := cookies.NewManager(tmpCacheDir)
	ctx.SetCookies(cookies)
	ctx.Client = "plakar-test/1.0.0"

	// Create a new logger
	logger := logging.NewLogger(bytes.NewBuffer(nil), bytes.NewBuffer(nil))
	logger.EnableInfo()
	ctx.SetLogger(logger)
	repo, err := repository.New(ctx, r, serializedConfig)
	// override the homedir to avoid having test overwriting existing home configuration
	ctx.HomeDir = repo.Location()
	args := []string{"-style", "notty"}

	subcommand := &Help{}
	err = subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// Close the write end of the pipe to signal EOF to the reader
	w.Close()

	// Wait for the read to complete
	wg.Wait()

	// Restore stdout
	os.Stdout = old

	output := buf.String()
	require.Contains(t, output, "PLAKAR(1) - General Commands Manual")
	require.Contains(t, output, "# ENVIRONMENT")
	require.Contains(t, output, "# FILES")
}

func TestParseCmdHelpCommand(t *testing.T) {
	// Create a pipe to capture stdout
	old := os.Stdout
	r1, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	// Restore stdout after the test
	defer func() {
		os.Stdout = old
		w.Close()
	}()

	// Use a WaitGroup to wait for the read to complete
	var wg sync.WaitGroup
	wg.Add(1)

	// Capture the output in a buffer
	var buf bytes.Buffer
	go func() {
		defer wg.Done()
		io.Copy(&buf, r1)
	}()

	tmpRepoDirRoot, err := os.MkdirTemp("", "tmp_repo")
	require.NoError(t, err)
	tmpRepoDir := fmt.Sprintf("%s/repo", tmpRepoDirRoot)
	tmpCacheDir, err := os.MkdirTemp("", "tmp_cache")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpCacheDir)
		os.RemoveAll(tmpRepoDir)
		os.RemoveAll(tmpRepoDirRoot)
	})

	ctx := appcontext.NewAppContext()

	// create a storage
	r, err := bfs.NewStore(ctx, "fs", map[string]string{"location": tmpRepoDir})
	require.NotNil(t, r)
	require.NoError(t, err)
	config := storage.NewConfiguration()
	serialized, err := config.ToBytes()
	require.NoError(t, err)

	hasher := hashing.GetHasher(hashing.DEFAULT_HASHING_ALGORITHM)
	wrappedConfigRd, err := storage.Serialize(hasher, resources.RT_CONFIG, versioning.GetCurrentVersion(resources.RT_CONFIG), bytes.NewReader(serialized))
	require.NoError(t, err)

	wrappedConfig, err := io.ReadAll(wrappedConfigRd)
	require.NoError(t, err)

	err = r.Create(ctx, wrappedConfig)
	require.NoError(t, err)

	// open the storage to load the configuration
	r, serializedConfig, err := storage.Open(ctx, map[string]string{"location": tmpRepoDir})
	require.NoError(t, err)

	// create a repository
	cache := caching.NewManager(tmpCacheDir)
	ctx.SetCache(cache)

	cookies := cookies.NewManager(tmpCacheDir)
	ctx.SetCookies(cookies)
	ctx.Client = "plakar-test/1.0.0"

	// Create a new logger
	logger := logging.NewLogger(bytes.NewBuffer(nil), bytes.NewBuffer(nil))
	logger.EnableInfo()
	ctx.SetLogger(logger)
	repo, err := repository.New(ctx, r, serializedConfig)
	// override the homedir to avoid having test overwriting existing home configuration
	ctx.HomeDir = repo.Location()
	args := []string{"-style", "notty", "version"}

	subcommand := &Help{}
	err = subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// Close the write end of the pipe to signal EOF to the reader
	w.Close()

	// Wait for the read to complete
	wg.Wait()

	// Restore stdout
	os.Stdout = old

	output := buf.String()
	require.Contains(t, output, "PLAKAR-VERSION(1) - General Commands Manual")
}
