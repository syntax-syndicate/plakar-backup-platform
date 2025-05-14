package testing

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/caching"
	"github.com/PlakarKorp/plakar/cookies"
	"github.com/PlakarKorp/plakar/encryption"
	"github.com/PlakarKorp/plakar/hashing"
	"github.com/PlakarKorp/plakar/logging"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/resources"
	"github.com/PlakarKorp/plakar/storage"
	bfs "github.com/PlakarKorp/plakar/storage/backends/fs"
	"github.com/PlakarKorp/plakar/versioning"
	"github.com/stretchr/testify/require"
)

func GenerateRepository(t *testing.T, bufout *bytes.Buffer, buferr *bytes.Buffer, passphrase *[]byte) *repository.Repository {
	// init temporary directories
	tmpRepoDirRoot, err := os.MkdirTemp("", "tmp_repo")
	require.NoError(t, err)
	tmpRepoDir := filepath.Join(tmpRepoDirRoot, "repo")
	tmpCacheDir, err := os.MkdirTemp("", "tmp_cache")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpRepoDir)
		os.RemoveAll(tmpCacheDir)
		os.RemoveAll(tmpRepoDirRoot)
	})

	cookies := cookies.NewManager(tmpCacheDir)

	ctx := appcontext.NewAppContext()
	ctx.SetCookies(cookies)

	ctx.Client = "plakar-test/1.0.0"

	// create a storage
	r, err := bfs.NewStore(ctx, "fs", map[string]string{"location": "fs://" + tmpRepoDir})
	require.NotNil(t, r)
	require.NoError(t, err)
	config := storage.NewConfiguration()
	hasher := hashing.GetHasher(hashing.DEFAULT_HASHING_ALGORITHM)

	var key []byte
	if passphrase != nil {
		key, err = encryption.DeriveKey(config.Encryption.KDFParams, *passphrase)
		require.NoError(t, err)

		canary, err := encryption.DeriveCanary(config.Encryption, key)
		require.NoError(t, err)

		config.Encryption.Canary = canary
		hasher = hashing.GetMACHasher(storage.DEFAULT_HASHING_ALGORITHM, key)
	} else {
		config.Encryption = nil
	}
	serialized, err := config.ToBytes()
	require.NoError(t, err)

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
	ctx.MaxConcurrency = 1
	if bufout != nil && buferr != nil {
		ctx.Stdout = bufout
		ctx.Stderr = buferr
	}
	cache := caching.NewManager(tmpCacheDir)
	ctx.SetCache(cache)

	if passphrase != nil {
		ctx.SetSecret(key)
	}

	// Create a new logger
	var logger *logging.Logger
	if bufout == nil || buferr == nil {
		logger = logging.NewLogger(os.Stdout, os.Stderr)
	} else {
		logger = logging.NewLogger(bufout, buferr)
	}
	if bufout != nil && buferr != nil {
		logger.EnableInfo()
	}
	// logger.EnableTrace("all")
	ctx.SetLogger(logger)
	repo, err := repository.New(ctx, r, serializedConfig)
	require.NoError(t, err, "creating repository")

	// override the homedir to avoid having test overwriting existing home configuration
	ctx.HomeDir = repo.Location()

	return repo
}

func GenerateRepositoryWithoutConfig(t *testing.T, bufout *bytes.Buffer, buferr *bytes.Buffer, passphrase *[]byte) *repository.Repository {
	// init temporary directories
	tmpRepoDirRoot, err := os.MkdirTemp("", "tmp_repo")
	require.NoError(t, err)
	tmpRepoDir := filepath.Join(tmpRepoDirRoot, "repo")
	tmpCacheDir, err := os.MkdirTemp("", "tmp_cache")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpRepoDir)
		os.RemoveAll(tmpCacheDir)
		os.RemoveAll(tmpRepoDirRoot)
	})

	cookies := cookies.NewManager(tmpCacheDir)

	ctx := appcontext.NewAppContext()
	ctx.SetCookies(cookies)
	ctx.Client = "plakar-test/1.0.0"
	ctx.MaxConcurrency = 1

	// create a storage
	r, err := bfs.NewStore(ctx, "fs", map[string]string{"location": tmpRepoDir})
	require.NotNil(t, r)
	require.NoError(t, err)
	config := storage.NewConfiguration()

	var key []byte
	if passphrase != nil {
		key, err = encryption.DeriveKey(config.Encryption.KDFParams, *passphrase)
		require.NoError(t, err)
	}

	// create a repository
	if bufout != nil && buferr != nil {
		ctx.Stdout = bufout
		ctx.Stderr = buferr
	}
	cache := caching.NewManager(tmpCacheDir)
	ctx.SetCache(cache)

	if passphrase != nil {
		ctx.SetSecret(key)
	}

	// Create a new logger
	var logger *logging.Logger
	if bufout == nil || buferr == nil {
		logger = logging.NewLogger(os.Stdout, os.Stderr)
	} else {
		logger = logging.NewLogger(bufout, buferr)
	}
	if bufout != nil && buferr != nil {
		logger.EnableInfo()
	}
	// logger.EnableTrace("all")
	ctx.SetLogger(logger)

	repo, err := repository.Inexistent(ctx, map[string]string{"location": tmpRepoDirRoot + "/repo"})
	require.NoError(t, err, "creating repository")

	return repo
}
