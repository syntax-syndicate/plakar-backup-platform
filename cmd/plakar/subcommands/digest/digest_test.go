package digest

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/caching"
	"github.com/PlakarKorp/plakar/hashing"
	"github.com/PlakarKorp/plakar/logging"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/resources"
	"github.com/PlakarKorp/plakar/snapshot"
	_ "github.com/PlakarKorp/plakar/snapshot/exporter/fs"
	"github.com/PlakarKorp/plakar/snapshot/importer/fs"
	"github.com/PlakarKorp/plakar/storage"
	bfs "github.com/PlakarKorp/plakar/storage/backends/fs"
	"github.com/PlakarKorp/plakar/versioning"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

func generateSnapshot(t *testing.T, bufOut *bytes.Buffer, bufErr *bytes.Buffer) *snapshot.Snapshot {
	// init temporary directories
	tmpRepoDirRoot, err := os.MkdirTemp("", "tmp_repo")
	require.NoError(t, err)
	tmpRepoDir := fmt.Sprintf("%s/repo", tmpRepoDirRoot)
	tmpCacheDir, err := os.MkdirTemp("", "tmp_cache")
	require.NoError(t, err)
	tmpBackupDir, err := os.MkdirTemp("", "tmp_to_backup")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpRepoDir)
		os.RemoveAll(tmpCacheDir)
		os.RemoveAll(tmpBackupDir)
		os.RemoveAll(tmpRepoDirRoot)
	})
	// create temporary files to backup
	err = os.MkdirAll(tmpBackupDir+"/subdir", 0755)
	require.NoError(t, err)
	err = os.MkdirAll(tmpBackupDir+"/another_subdir", 0755)
	require.NoError(t, err)
	err = os.WriteFile(tmpBackupDir+"/subdir/dummy.txt", []byte("hello dummy"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(tmpBackupDir+"/subdir/foo.txt", []byte("hello foo"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(tmpBackupDir+"/subdir/to_exclude", []byte("*/subdir/to_exclude\n"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(tmpBackupDir+"/another_subdir/bar", []byte("hello bar"), 0644)
	require.NoError(t, err)

	// create a storage
	r, err := bfs.NewStore(map[string]string{"location": "fs://" + tmpRepoDir})
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

	err = r.Create(wrappedConfig)
	require.NoError(t, err)

	// open the storage to load the configuration
	r, serializedConfig, err := storage.Open(map[string]string{"location": tmpRepoDir})
	require.NoError(t, err)

	// create a repository
	ctx := appcontext.NewAppContext()
	ctx.Stdout = bufOut
	ctx.Stderr = bufErr
	cache := caching.NewManager(tmpCacheDir)
	ctx.SetCache(cache)

	// Create a new logger
	logger := logging.NewLogger(bufOut, bufErr)
	logger.EnableInfo()
	ctx.SetLogger(logger)
	repo, err := repository.New(ctx, r, serializedConfig)
	require.NoError(t, err, "creating repository")

	// create a snapshot
	snap, err := snapshot.New(repo)
	require.NoError(t, err)
	require.NotNil(t, snap)

	imp, err := fs.NewFSImporter(map[string]string{"location": tmpBackupDir})
	require.NoError(t, err)
	snap.Backup(imp, &snapshot.BackupOptions{Name: "test_backup", MaxConcurrency: 1})

	err = snap.Repository().RebuildState()
	require.NoError(t, err)

	return snap
}

func TestExecuteCmdDigestDefault(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	snap := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	ctx := snap.AppContext()
	ctx.MaxConcurrency = 1

	repo := snap.Repository()
	// override the homedir to avoid having test overwriting existing home configuration
	ctx.HomeDir = repo.Location()
	indexId := snap.Header.GetIndexID()
	args := []string{fmt.Sprintf("%s", hex.EncodeToString(indexId[:]))}

	subcommand, err := parse_cmd_digest(ctx, repo, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)
	require.Equal(t, "digest", subcommand.(*Digest).Name())

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// output should look like this
	// SHA256 (/tmp/tmp_to_backup3363028982/another_subdir/bar) = b585d5afa0d0a97a7c217eeb9d9adf08fc63188d4204fc7d537a178224b477e6
	// SHA256 (/tmp/tmp_to_backup3363028982/subdir/dummy.txt) = f4da3ebff9dbd21cfb270054dee6948f96de93f68f525e0bf4067ce2f9e2d639
	// SHA256 (/tmp/tmp_to_backup3363028982/subdir/foo.txt) = 6c8aa524fae27a3607f9c4204567b65d48341b3bcc0e36e9e50856aaaf073d21
	// SHA256 (/tmp/tmp_to_backup3363028982/subdir/to_exclude) = dd7117865f65a87aba1e171b82e073914a2cdffb1b34407dea682f62c3dc72e0

	output := bufOut.String()
	require.Contains(t, output, "dummy.txt")
	lines := strings.Split(strings.Trim(output, "\n"), "\n")
	for _, line := range lines {
		require.Contains(t, line, "SHA256 (")
	}
}

func TestExecuteCmdDigestNoParam(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	snap := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	ctx := snap.AppContext()
	ctx.MaxConcurrency = 1

	repo := snap.Repository()
	// override the homedir to avoid having test overwriting existing home configuration
	ctx.HomeDir = repo.Location()
	args := []string{}

	subcommand, err := parse_cmd_digest(ctx, repo, args)
	require.Error(t, err, "at least one parameter is required")
	require.Nil(t, subcommand)
}

func TestExecuteCmdDigestWrongHashing(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	snap := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	ctx := snap.AppContext()
	ctx.MaxConcurrency = 1

	repo := snap.Repository()
	// override the homedir to avoid having test overwriting existing home configuration
	ctx.HomeDir = repo.Location()
	indexId := snap.Header.GetIndexID()
	args := []string{"-hashing", "md5", fmt.Sprintf("%s", hex.EncodeToString(indexId[:]))}

	subcommand, err := parse_cmd_digest(ctx, repo, args)
	require.Error(t, err, "at least one parameter is required")
	require.Nil(t, subcommand)
}
