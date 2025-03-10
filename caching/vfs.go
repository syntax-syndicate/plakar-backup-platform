package caching

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/linxGnu/grocksdb"
)

type _VFSCache struct {
	manager *Manager
	db      *grocksdb.DB
}

func newVFSCache(cacheManager *Manager, scheme string, origin string) (*_VFSCache, error) {
	cacheDir := filepath.Join(cacheManager.cacheDir, "vfs", scheme, origin)

	err := os.MkdirAll(cacheDir, os.ModePerm)

	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	db, err := grocksdb.OpenDb(opts, cacheDir)
	if err != nil {
		return nil, err
	}

	return &_VFSCache{
		manager: cacheManager,
		db:      db,
	}, nil
}

func (c *_VFSCache) Close() {
	c.db.Close()
}

func (c *_VFSCache) put(prefix string, pathname string, data []byte) error {
	return c.db.Put(grocksdb.NewDefaultWriteOptions(), []byte(fmt.Sprintf("%s:%s", prefix, pathname)), data)
}

func (c *_VFSCache) get(prefix, pathname string) ([]byte, error) {
	data, err := c.db.GetBytes(grocksdb.NewDefaultReadOptions(), []byte(fmt.Sprintf("%s:%s", prefix, pathname)))
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (c *_VFSCache) PutDirectory(pathname string, data []byte) error {
	return c.put("__directory__", pathname, data)
}

func (c *_VFSCache) GetDirectory(pathname string) ([]byte, error) {
	return c.get("__directory__", pathname)
}

func (c *_VFSCache) PutFilename(pathname string, data []byte) error {
	return c.put("__filename__", pathname, data)
}

func (c *_VFSCache) GetFilename(pathname string) ([]byte, error) {
	return c.get("__filename__", pathname)
}

func (c *_VFSCache) PutFileSummary(pathname string, data []byte) error {
	return c.put("__file_summary__", pathname, data)
}

func (c *_VFSCache) GetFileSummary(pathname string) ([]byte, error) {
	return c.get("__file_summary__", pathname)
}

func (c *_VFSCache) PutObject(mac [32]byte, data []byte) error {
	return c.put("__object__", fmt.Sprintf("%x", mac), data)
}

func (c *_VFSCache) GetObject(mac [32]byte) ([]byte, error) {
	return c.get("__object__", fmt.Sprintf("%x", mac))
}
