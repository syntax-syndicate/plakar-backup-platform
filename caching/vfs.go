package caching

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/bloom"
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

func makeKey(keyInput string) []byte {
	key := []byte(keyInput)

	key = append(key, 0)
	return key
}

func makeSuffixKey(keyInput string, suffixLen int) []byte {
	key := []byte(keyInput)

	return append(key, byte(suffixLen))
}

func extractPrefixKey(key []byte) []byte {
	suffixLen := int(key[len(key)-1])
	prefixEnd := len(key) - 1 - suffixLen
	return key[:prefixEnd]

}

func newVFSCache(cacheManager *Manager, repositoryID uuid.UUID, scheme string, origin string) (*_VFSCache, error) {
	cacheDir := filepath.Join(cacheManager.cacheDir, "vfs", repositoryID.String(), scheme, origin)

	opts := &pebble.Options{
		DisableWAL: true,
		Comparer: &pebble.Comparer{
			AbbreviatedKey: func(key []byte) uint64 {
				prefixKey := extractPrefixKey(key)
				return pebble.DefaultComparer.AbbreviatedKey(prefixKey)
			},
			Separator: func(dst, a, b []byte) []byte {
				aPrefix := extractPrefixKey(a)
				rPrefix := extractPrefixKey(b)

				return pebble.DefaultComparer.Separator(dst, aPrefix, rPrefix)
			},
			Successor: func(dst, a []byte) []byte {
				aPrefix := extractPrefixKey(a)
				return pebble.DefaultComparer.Successor(dst, aPrefix)
			},
			Split: func(key []byte) int {
				if len(key) == 0 {
					return 0
				}

				// Last byte of the key is the suffix len or zero if there are none.
				suffixLen := int(key[len(key)-1])
				return len(key) - suffixLen - 1
			},
			Name: "cache_comparer",
		},
	}

	opts.EnsureDefaults()
	for i := 0; i < len(opts.Levels); i++ {
		l := &opts.Levels[i]
		l.FilterPolicy = bloom.FilterPolicy(10)
		l.FilterType = pebble.TableFilter
	}

	db, err := pebble.Open(cacheDir, opts)
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
	key := makeKey(fmt.Sprintf("%s:%s", prefix, pathname))
	return c.db.Set(key, data, &pebble.WriteOptions{Sync: false})
}

func (c *_VFSCache) get(prefix, pathname string) (*ResourceCloser, error) {
	key := makeKey(fmt.Sprintf("%s:%s", prefix, pathname))
	data, del, err := c.db.Get(key)
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
