package snapfs

import (
	"encoding/hex"
	"errors"
	"io/fs"
	"time"
)

type rootinfo struct{ name string }

func (ri *rootinfo) Name() string               { return ri.name }
func (ri *rootinfo) Size() int64                { return 0 }
func (ri *rootinfo) Mode() fs.FileMode          { return 0750 }
func (ri *rootinfo) ModTime() time.Time         { return time.Time{} }
func (ri *rootinfo) IsDir() bool                { return true }
func (ri *rootinfo) Sys() any                   { return nil }
func (ri *rootinfo) Type() fs.FileMode          { return fs.ModeDir }
func (ri *rootinfo) Info() (fs.FileInfo, error) { return ri, nil }

type rootdir struct {
	sfs    *snapfs
	offset int
}

func (rd *rootdir) Stat() (FileInfo, error)                   { return &rootinfo{"/"}, nil }
func (rd *rootdir) Read([]byte) (int, error)                  { return 0, fs.ErrInvalid }
func (rd *rootdir) Seek(int64, int) (int64, error)            { return 0, errors.ErrUnsupported }
func (rd *rootdir) Close() error                              { return nil }
func (rd *rootdir) Readdir(n int) (ret []FileInfo, err error) { return rd.ReadDir(n) }

func (rd *rootdir) ReadDir(n int) (entries []FileInfo, err error) {
	i := -1
	for id := range rd.sfs.repo.ListSnapshots() {
		i++
		if i < rd.offset {
			continue
		}

		entries = append(entries, &rootinfo{hex.EncodeToString(id[:])})
		rd.offset++
		if n > 0 && len(entries) == n {
			break
		}
	}
	return
}
