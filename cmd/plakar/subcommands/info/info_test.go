package info

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

func TestExecuteCmdInfoDefault(t *testing.T) {
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

	subcommand, err := parse_cmd_info(ctx, repo, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// output should look like this
	// Version: 1.0.0
	// Timestamp: 2025-03-05 21:48:39.742132699 +0000 UTC
	// RepositoryID: 79650133-b57c-46a9-aff9-7dcaf7829033
	// Packfile:
	// - MaxSize: 21 MB (20971520 bytes)
	// Chunking:
	// - Algorithm: FASTCDC
	// - MinSize: 66 kB (65536 bytes)
	// - NormalSize: 1.0 MB (1048576 bytes)
	// - MaxSize: 4.2 MB (4194304 bytes)
	// Hashing:
	// - Algorithm: BLAKE3
	// - Bits: 256
	// Compression:
	// - Algorithm: LZ4
	// - Level: 131072
	// Encryption:
	// - SubkeyAlgorithm: AES256-KW
	// - DataAlgorithm: AES256-GCM-SIV
	// - ChunkSize: 65536
	// - Canary:
	// - KDF: ARGON2ID
	// - Salt: 1f9d5fbf813e81066d863c77d2093612
	// - SaltSize: 16
	// - KeyLen: 32
	// - Time: 4
	// - Memory: 262144
	// - Thread: 1
	// Snapshots: 1
	// Size: 49 B (49 bytes)

	output := bufOut.String()
	require.Contains(t, output, "Snapshots: 1")
}

func _TestExecuteCmdInfoSnapshot(t *testing.T) {
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
	args := []string{hex.EncodeToString(indexId[:])}

	subcommand, err := parse_cmd_info(ctx, repo, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// output should look like this
	// Version: 1.0.0
	// SnapshotID: 9ed843b00c92b6a69eb68b1f691032587a31e102fab7c82b64ad2589585883f1
	// Timestamp: 2025-03-05 21:49:22.875334881 +0000 UTC
	// Duration: 5.000531ms
	// Name: test_backup
	// Environment: default
	// Perimeter: default
	// Category: default
	// VFS: {649537a22c67f1b29bb4f59632e104653c3a01088605ad7fc25af5fc96b6e7cb d358b0f4fcab617140ff49cd0bec5f8ff3c35deca65f4754a320245f50238e00 d358b0f4fcab617140ff49cd0bec5f8ff3c35deca65f4754a320245f50238e00}
	// Importer:
	// - Type: fs
	// - Origin: grumpf
	// - Directory: /tmp/tmp_to_backup3285582724
	// Context:
	// - MachineID:
	// - Hostname:
	// - Username:
	// - OperatingSystem:
	// - Architecture:
	// - NumCPU: 16
	// - GOMAXPROCS:
	// - ProcessID: 0
	// - Client:
	// - CommandLine:
	// Summary:
	// - Directories: 0
	// - Files: 4
	// - Symlinks: 0
	// - Devices: 0
	// - Pipes: 0
	// - Sockets: 0
	// - Setuid: 0
	// - Setgid: 0
	// - Sticky: 0
	// - Objects: 4
	// - Chunks: 4
	// - MinSize: 0 B (0 bytes)
	// - MaxSize: 20 B (20 bytes)
	// - Size: 49 B (49 bytes)
	// - MinModTime: 1970-01-01 00:00:00 +0000 UTC
	// - MaxModTime: 2025-03-05 21:49:22 +0000 UTC
	// - MinEntropy: 0.000000
	// - MaxEntropy: 3.921928
	// - HiEntropy: 0
	// - LoEntropy: 0
	// - MIMEAudio: 0
	// - MIMEVideo: 0
	// - MIMEImage: 0
	// - MIMEText: 4
	// - MIMEApplication: 0
	// - MIMEOther: 0
	// - Errors: 0

	output := bufOut.String()
	fmt.Println(output)
	require.Contains(t, output, "Name: test_backup")
	require.Contains(t, output, "Files: 4")
	require.Contains(t, output, fmt.Sprintf("Directory: %s", snap.Header.GetSource(0).Importer.Directory))
	require.Contains(t, output, fmt.Sprintf("SnapshotID: %s", hex.EncodeToString(indexId[:])))
}

func TestExecuteCmdInfoSnapshotPath(t *testing.T) {
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
	args := []string{fmt.Sprintf("%s:subdir/dummy.txt", hex.EncodeToString(indexId[:]))}

	subcommand, err := parse_cmd_info(ctx, repo, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// output should look like this
	// [FileEntry]
	// Version: 65536
	// ParentPath: /tmp/tmp_to_backup1755905950/subdir
	// Name: dummy.txt
	// Type: regular
	// Size: 11 B (11 bytes)
	// Permissions: -rw-r--r--
	// ModTime: 2025-03-06 07:51:06.716971661 +0000 UTC
	// DeviceID: 64768
	// InodeID: 22314615
	// UserID: 1000
	// GroupID: 1000
	// Username: sayoun
	// Groupname: sayoun
	// NumLinks: 1
	// ExtendedAttributes: []
	// FileAttributes: 0
	// Classification:
	// CustomMetadata: []
	// Tags: []

	output := bufOut.String()
	require.Contains(t, output, "[FileEntry]")
	require.Contains(t, output, "Name: dummy.txt")
}
