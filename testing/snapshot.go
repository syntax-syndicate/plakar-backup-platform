package testing

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/snapshot"
	"github.com/PlakarKorp/plakar/snapshot/importer/fs"
	"github.com/stretchr/testify/require"
)

type MockFile struct {
	Path    string
	IsDir   bool
	Mode    os.FileMode
	Content []byte
}

func NewMockDir(path string) MockFile {
	return MockFile{
		Path:  path,
		IsDir: true,
		Mode:  0755,
	}
}

func NewMockFile(path string, mode os.FileMode, content string) MockFile {
	return MockFile{
		Path:    path,
		Mode:    mode,
		Content: []byte(content),
	}
}

type testingOptions struct {
	name string
}

func newTestingOptions() *testingOptions {
	return &testingOptions{
		name: "test_backup",
	}
}

type TestingOptions func(o *testingOptions)

func WithName(name string) TestingOptions {
	return func(o *testingOptions) {
		o.name = name
	}
}

func GenerateFiles(t *testing.T, files []MockFile) string {
	tmpBackupDir, err := os.MkdirTemp("", "tmp_to_backup")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpBackupDir)
	})

	for _, file := range files {
		dest := filepath.Join(tmpBackupDir, filepath.FromSlash(file.Path))
		if file.IsDir {
			err = os.MkdirAll(dest, file.Mode)
		} else {
			err = os.WriteFile(dest, file.Content, file.Mode)
		}
	}

	return tmpBackupDir
}

func GenerateSnapshot(t *testing.T, repo *repository.Repository, files []MockFile, opts ...TestingOptions) *snapshot.Snapshot {
	o := newTestingOptions()
	for _, f := range opts {
		f(o)
	}

	tmpBackupDir := GenerateFiles(t, files)
	repo.AppContext().CWD = tmpBackupDir

	// create a snapshot
	builder, err := snapshot.Create(repo, repository.DefaultType)
	require.NoError(t, err)
	require.NotNil(t, builder)

	imp, err := fs.NewFSImporter(map[string]string{"location": tmpBackupDir})
	require.NoError(t, err)
	builder.Backup(imp, &snapshot.BackupOptions{Name: o.name, MaxConcurrency: 1})

	err = builder.Repository().RebuildState()
	require.NoError(t, err)

	// reopen it
	snap, err := snapshot.Load(repo, builder.Header.Identifier)
	require.NoError(t, err)
	require.NotNil(t, snap)

	checkCache, err := repo.AppContext().GetCache().Check()
	require.NoError(t, err)
	snap.SetCheckCache(checkCache)

	return snap
}
