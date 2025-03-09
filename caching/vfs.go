package caching

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/cockroachdb/pebble"
	"github.com/google/uuid"
)

type _VFSCache struct {
	manager *Manager
	db      *pebble.DB
}

type ResourceCloser struct {
	Buf    []byte
	Closer io.Closer
}

func newVFSCache(cacheManager *Manager, repositoryID uuid.UUID, scheme string, origin string) (*_VFSCache, error) {
	cacheDir := filepath.Join(cacheManager.cacheDir, "vfs", repositoryID.String(), scheme, origin)

	db, err := pebble.Open(cacheDir, &pebble.Options{DisableWAL: true})
	if err != nil {
		return nil, err
	}

	return &_VFSCache{
		manager: cacheManager,
		db:      db,
	}, nil
}

func (c *_VFSCache) Close() error {
	return c.db.Close()
}

func (c *_VFSCache) put(prefix string, pathname string, data []byte) error {
	return c.db.Set([]byte(fmt.Sprintf("%s:%s", prefix, pathname)), data, &pebble.WriteOptions{Sync: false})
}

func (c *_VFSCache) get(prefix, pathname string) (*ResourceCloser, error) {
	data, del, err := c.db.Get([]byte(fmt.Sprintf("%s:%s", prefix, pathname)))
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &ResourceCloser{Buf: data, Closer: del}, nil
}

func (c *_VFSCache) PutDirectory(pathname string, data []byte) error {
	return c.put("__directory__", pathname, data)
}

func (c *_VFSCache) GetDirectory(pathname string) (*ResourceCloser, error) {
	return c.get("__directory__", pathname)
}

func (c *_VFSCache) PutFilename(pathname string, data []byte) error {
	return c.put("__filename__", pathname, data)
}

func (c *_VFSCache) GetFilename(pathname string) (*ResourceCloser, error) {
	return c.get("__filename__", pathname)
}

func (c *_VFSCache) PutFileSummary(pathname string, data []byte) error {
	return c.put("__file_summary__", pathname, data)
}

func (c *_VFSCache) GetFileSummary(pathname string) (*ResourceCloser, error) {
	return c.get("__file_summary__", pathname)
}

func (c *_VFSCache) PutObject(mac [32]byte, data []byte) error {
	return c.put("__object__", fmt.Sprintf("%x", mac), data)
}

func (c *_VFSCache) GetObject(mac [32]byte) (*ResourceCloser, error) {
	return c.get("__object__", fmt.Sprintf("%x", mac))
}
