package snapshot

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/caching"
	"github.com/PlakarKorp/plakar/logging"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/snapshot/importer/fs"
	"github.com/PlakarKorp/plakar/storage"
	bfs "github.com/PlakarKorp/plakar/storage/backends/fs"
	"github.com/stretchr/testify/require"
)

func randFileName(prefix string) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	suffix := make([]byte, 8)
	for i := range suffix {
		suffix[i] = chars[rand.Intn(len(chars))]
	}
	return prefix + string(suffix)
}

func TestSnapshot(t *testing.T) {
	// init temporary directories
	tmpRepoDir := fmt.Sprintf("/tmp/%s", randFileName("tmp_repo_"))
	tmpCacheDir, err := os.MkdirTemp("", "tmp_cache")
	require.NoError(t, err)
	tmpBackupDir, err := os.MkdirTemp("", "tmp_to_backup")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpRepoDir)
		os.RemoveAll(tmpCacheDir)
		os.RemoveAll(tmpBackupDir)
	})
	// create a temporary file to backup later
	err = os.WriteFile(tmpBackupDir+"/dummy.txt", []byte("hello"), 0644)
	require.NoError(t, err)

	// create a storage
	r := bfs.NewRepository("fs://" + tmpRepoDir)
	require.NotNil(t, r)
	config := storage.NewConfiguration()
	err = r.Create("fs://"+tmpRepoDir, *config)
	require.NoError(t, err)

	// open the storage to load the configuration
	err = r.Open("fs://" + tmpRepoDir)
	require.NoError(t, err)

	// crate a repository
	ctx := appcontext.NewAppContext()
	cache := caching.NewManager(tmpCacheDir)
	defer cache.Close()
	ctx.SetCache(cache)
	logger := logging.NewLogger(os.Stdout, os.Stderr)
	logger.EnableTrace("all")
	ctx.SetLogger(logger)
	repo, err := repository.New(ctx, r, nil)
	require.NoError(t, err, "creating repository")

	// create a snapshot
	snap, err := New(repo)
	require.NoError(t, err)
	require.NotNil(t, snap)
	defer snap.Close()

	appCtx := snap.AppContext()
	require.NotNil(t, appCtx)

	imp, err := fs.NewFSImporter(tmpBackupDir)
	require.NoError(t, err)
	snap.Backup(tmpBackupDir, imp, &BackupOptions{Name: "test_backup", MaxConcurrency: 1})

	events := appCtx.Events()
	require.NotNil(t, events)

	a, err := r.GetPackfiles()
	require.NoError(t, err)
	require.NotNil(t, a)

	err = repo.RebuildState()
	require.NoError(t, err)

	for d, err := range snap.ListDatas() {
		require.NoError(t, err)
		require.NotNil(t, d)
	}

	snapFs, err := snap.Filesystem()
	require.NoError(t, err)
	require.NotNil(t, snapFs)

	iter, err := snap.ListObjects()
	require.NoError(t, err)
	require.NotNil(t, iter)
	for o, err := range iter {
		require.NoError(t, err)
		require.NotNil(t, o)
		lo, err := snap.LookupObject(o)
		require.NoError(t, err)
		require.NotNil(t, lo)
	}

	iter2, err := snap.ListChunks()
	require.NoError(t, err)
	require.NotNil(t, iter2)
	for o, err := range iter2 {
		require.NoError(t, err)
		require.NotNil(t, o)
	}

	snap2, err := Load(repo, snap.Header.Identifier)
	require.NoError(t, err)
	require.NotNil(t, snap2)

	require.Equal(t, snap.Header.Identifier, snap2.Header.Identifier)
	require.Equal(t, snap.Header.Timestamp.Truncate(time.Nanosecond), snap2.Header.Timestamp.Truncate(time.Nanosecond))

	snap3, err := Clone(repo, snap.Header.Identifier)
	require.NoError(t, err)
	require.NotNil(t, snap3)

	require.NotEqual(t, snap.Header.Identifier, snap3.Header.Identifier)

	snap4, err := Fork(repo, snap.Header.Identifier)
	require.NoError(t, err)
	require.NotNil(t, snap4)

	require.NotEqual(t, snap.Header.Identifier, snap4.Header.Identifier)
}
