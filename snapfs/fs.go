package snapfs

import (
	"io/fs"
	"time"
)

// FS is an usable superset of fs.FS and fs.StatFS.
type FS interface {
	Open(name string) (File, error)
	Stat(name string) (FileInfo, error)
}

// FileInfo is an usable superset of fs.FileInfo and fs.DirEntry
type FileInfo interface {
	Name() string
	Size() int64
	Mode() fs.FileMode
	ModTime() time.Time
	IsDir() bool
	Sys() any
}

// File is an usable superset of fs.File, fs.Seeker and http.File.
type File interface {
	Stat() (FileInfo, error)
	Read([]byte) (int, error)
	Seek(int64, int) (int64, error)
	Readdir(int) ([]FileInfo, error)
	Close() error
}

// ReadDirFile is an usable superset of fs.ReadDirFile
type ReadDirFile interface {
	File
	ReadDir(int) ([]FileInfo, error)
}
