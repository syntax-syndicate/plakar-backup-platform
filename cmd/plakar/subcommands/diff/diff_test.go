package diff

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"os"
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

func generateFixtures(t *testing.T, bufOut *bytes.Buffer, bufErr *bytes.Buffer) (*repository.Repository, string) {
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
	r, serializedConfig, err := storage.Open(map[string]string{"location": "fs://" + tmpRepoDir})
	require.NoError(t, err)

	// create a repository
	ctx := appcontext.NewAppContext()
	ctx.Stdout = bufOut
	ctx.Stderr = bufErr
	cache := caching.NewManager(tmpCacheDir)
	ctx.SetCache(cache)

	// Create a new logger)
	logger := logging.NewLogger(bufOut, bufErr)
	logger.EnableInfo()
	ctx.SetLogger(logger)
	repo, err := repository.New(ctx, r, serializedConfig)
	require.NoError(t, err, "creating repository")

	return repo, tmpBackupDir
}

func TestExecuteCmdDiffIdentical(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, tmpBackupDir := generateFixtures(t, bufOut, bufErr)

	// create one snapshot
	snap, err := snapshot.New(repo)
	require.NoError(t, err)
	require.NotNil(t, snap)

	imp, err := fs.NewFSImporter(map[string]string{"location": tmpBackupDir})
	require.NoError(t, err)
	snap.Backup(imp, &snapshot.BackupOptions{Name: "test_backup1", MaxConcurrency: 1})

	err = snap.Repository().RebuildState()
	require.NoError(t, err)

	// create second snapshot
	snap2, err := snapshot.New(repo)
	require.NoError(t, err)
	require.NotNil(t, snap2)

	snap2.Backup(imp, &snapshot.BackupOptions{Name: "test_backup2", MaxConcurrency: 1})

	err = snap2.Repository().RebuildState()
	require.NoError(t, err)

	ctx := repo.AppContext()
	ctx.MaxConcurrency = 1
	// override the homedir to avoid having test overwriting existing home configuration
	ctx.HomeDir = repo.Location()
	indexId1 := snap.Header.GetIndexShortID()
	indexId2 := snap2.Header.GetIndexShortID()
	backupDir := snap.Header.GetSource(0).Importer.Directory
	snapPath1 := fmt.Sprintf("%s:%s/subdir/dummy.txt", hex.EncodeToString(indexId1[:]), backupDir)
	snapPath2 := fmt.Sprintf("%s:%s/subdir/dummy.txt", hex.EncodeToString(indexId2[:]), backupDir)
	args := []string{snapPath1, snapPath2}

	subcommand, err := parse_cmd_diff(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)
	require.Equal(t, "diff", subcommand.(*Diff).Name())

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	outputErr := bufErr.String()
	require.Contains(t, outputErr, fmt.Sprintf("%s and %s are identical\n", snapPath1, snapPath2))
}

func TestExecuteCmdDiffFiles(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, tmpBackupDir := generateFixtures(t, bufOut, bufErr)

	// create one snapshot
	snap, err := snapshot.New(repo)
	require.NoError(t, err)
	require.NotNil(t, snap)

	imp, err := fs.NewFSImporter(map[string]string{"location": tmpBackupDir})
	require.NoError(t, err)
	snap.Backup(imp, &snapshot.BackupOptions{Name: "test_backup1", MaxConcurrency: 1})

	err = snap.Repository().RebuildState()
	require.NoError(t, err)

	// modify content before second backup
	err = os.WriteFile(tmpBackupDir+"/subdir/dummy.txt", []byte("hello dumpy"), 0644)
	require.NoError(t, err)

	// create second snapshot
	snap2, err := snapshot.New(repo)
	require.NoError(t, err)
	require.NotNil(t, snap2)

	snap2.Backup(imp, &snapshot.BackupOptions{Name: "test_backup2", MaxConcurrency: 1})

	err = snap2.Repository().RebuildState()
	require.NoError(t, err)

	ctx := repo.AppContext()
	ctx.MaxConcurrency = 1
	// override the homedir to avoid having test overwriting existing home configuration
	ctx.HomeDir = repo.Location()
	indexId1 := snap.Header.GetIndexShortID()
	indexId2 := snap2.Header.GetIndexShortID()
	backupDir := snap.Header.GetSource(0).Importer.Directory
	snapPath1 := fmt.Sprintf("%s:%s/subdir/dummy.txt", hex.EncodeToString(indexId1[:]), backupDir)
	snapPath2 := fmt.Sprintf("%s:%s/subdir/dummy.txt", hex.EncodeToString(indexId2[:]), backupDir)
	args := []string{snapPath1, snapPath2}

	subcommand, err := parse_cmd_diff(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)
	require.Equal(t, "diff", subcommand.(*Diff).Name())

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	output := bufOut.String()
	require.Contains(t, output, `
@@ -1 +1 @@
-hello dummy
+hello dumpy`)
}
