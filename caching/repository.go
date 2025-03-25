package caching

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/resources"
	"github.com/cockroachdb/pebble"
	"github.com/google/uuid"
)

var ErrInUse = fmt.Errorf("cache in use")

type _RepositoryCache struct {
	manager    *Manager
	cookiesDir string
	db         *pebble.DB
}

func newRepositoryCache(cacheManager *Manager, repositoryID uuid.UUID) (*_RepositoryCache, error) {
	cookiesDir := filepath.Join(cacheManager.cacheDir, "cookies", repositoryID.String())
	if err := os.MkdirAll(cookiesDir, 0700); err != nil {
		return nil, err
	}

	cacheDir := filepath.Join(cacheManager.cacheDir, "repository", repositoryID.String())
	db, err := pebble.Open(cacheDir, &pebble.Options{DisableWAL: true})
	if err != nil {
		if errors.Is(err, syscall.EAGAIN) {
			return nil, ErrInUse
		}
		return nil, err
	}

	return &_RepositoryCache{
		manager:    cacheManager,
		cookiesDir: cookiesDir,
		db:         db,
	}, nil
}

func (c *_RepositoryCache) Close() error {
	return c.db.Close()
}

func (c *_RepositoryCache) HasCookie(name string) bool {
	name = strings.ReplaceAll(name, "/", "_")
	_, err := os.Stat(filepath.Join(c.cookiesDir, name))
	return err == nil
}

func (c *_RepositoryCache) PutCookie(name string) error {
	name = strings.ReplaceAll(name, "/", "_")
	_, err := os.Create(filepath.Join(c.cookiesDir, name))
	return err
}

func (c *_RepositoryCache) put(prefix string, key string, data []byte) error {
	return c.db.Set([]byte(fmt.Sprintf("%s:%s", prefix, key)), data, &pebble.WriteOptions{Sync: false})
}

func (c *_RepositoryCache) has(prefix, key string) (bool, error) {
	_, del, err := c.db.Get([]byte(fmt.Sprintf("%s:%s", prefix, key)))

	if err != nil {
		if err == pebble.ErrNotFound {
			return false, nil
		}
		return false, err
	}

	del.Close()
	return true, nil
}

func (c *_RepositoryCache) get(prefix, key string) ([]byte, error) {
	data, del, err := c.db.Get([]byte(fmt.Sprintf("%s:%s", prefix, key)))
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	ret := make([]byte, len(data))
	copy(ret, data)
	del.Close()

	return ret, nil
}

func (c *_RepositoryCache) getObjects(keyPrefix string) iter.Seq2[objects.MAC, []byte] {
	return func(yield func(objects.MAC, []byte) bool) {
		keyUpperBound := func(b []byte) []byte {
			end := make([]byte, len(b))
			copy(end, b)
			for i := len(end) - 1; i >= 0; i-- {
				end[i] = end[i] + 1
				if end[i] != 0 {
					return end[:i+1]
				}
			}
			return nil // no upper-bound
		}

		prefixIterOptions := func(prefix []byte) *pebble.IterOptions {
			return &pebble.IterOptions{
				LowerBound: prefix,
				UpperBound: keyUpperBound(prefix),
			}
		}
		iter, _ := c.db.NewIter(prefixIterOptions([]byte(keyPrefix)))
		defer iter.Close()

		for iter.First(); iter.Valid(); iter.Next() {
			if !strings.HasPrefix(string(iter.Key()), keyPrefix) {
				break
			}

			/* Extract the csum part of the key, this avoids decoding the full
			 * entry later on if that's the only thing we need */
			key := iter.Key()
			hex_csum := string(key[bytes.LastIndexByte(key, byte(':'))+1:])
			csum, _ := hex.DecodeString(hex_csum)

			if !yield(objects.MAC(csum), iter.Value()) {
				return
			}
		}
	}
}

func (c *_RepositoryCache) delete(prefix, key string) error {
	return c.db.Delete([]byte(fmt.Sprintf("%s:%s", prefix, key)), nil)
}

func (c *_RepositoryCache) PutState(stateID objects.MAC, data []byte) error {
	return c.put("__state__", fmt.Sprintf("%x", stateID), data)
}

func (c *_RepositoryCache) HasState(stateID objects.MAC) (bool, error) {
	return c.has("__state__", fmt.Sprintf("%x", stateID))
}

func (c *_RepositoryCache) GetState(stateID objects.MAC) ([]byte, error) {
	return c.get("__state__", fmt.Sprintf("%x", stateID))
}

func (c *_RepositoryCache) DelState(stateID objects.MAC) error {
	return c.delete("__state__", fmt.Sprintf("%x", stateID))
}

func (c *_RepositoryCache) GetStates() (map[objects.MAC][]byte, error) {
	ret := make(map[objects.MAC][]byte, 0)

	keyPrefix := "__state__:"
	keyUpperBound := func(b []byte) []byte {
		end := make([]byte, len(b))
		copy(end, b)
		for i := len(end) - 1; i >= 0; i-- {
			end[i] = end[i] + 1
			if end[i] != 0 {
				return end[:i+1]
			}
		}
		return nil // no upper-bound
	}

	prefixIterOptions := func(prefix []byte) *pebble.IterOptions {
		return &pebble.IterOptions{
			LowerBound: prefix,
			UpperBound: keyUpperBound(prefix),
		}
	}
	iter, _ := c.db.NewIter(prefixIterOptions([]byte(keyPrefix)))
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		if !strings.HasPrefix(string(iter.Key()), keyPrefix) {
			break
		}

		var stateID objects.MAC
		_, err := hex.Decode(stateID[:], iter.Key()[len(keyPrefix):])
		if err != nil {
			fmt.Printf("Error decoding state ID: %v\n", err)
			return nil, err
		}
		ret[stateID] = iter.Value()
	}

	return ret, nil
}

func (c *_RepositoryCache) GetDelta(blobType resources.Type, blobCsum objects.MAC) iter.Seq2[objects.MAC, []byte] {
	return c.getObjects(fmt.Sprintf("__delta__:%d:%x:", blobType, blobCsum))
}

func (c *_RepositoryCache) HasDelta(blobType resources.Type, blobCsum objects.MAC) (bool, error) {
	return c.has("__delta__", fmt.Sprintf("%d:%x", blobType, blobCsum))
}

func (c *_RepositoryCache) PutDelta(blobType resources.Type, blobCsum, packfile objects.MAC, data []byte) error {
	return c.put("__delta__", fmt.Sprintf("%d:%x:%x", blobType, blobCsum, packfile), data)
}

func (c *_RepositoryCache) GetDeltasByType(blobType resources.Type) iter.Seq2[objects.MAC, []byte] {
	return func(yield func(objects.MAC, []byte) bool) {
		keyPrefix := fmt.Sprintf("__delta__:%d:", blobType)
		keyUpperBound := func(b []byte) []byte {
			end := make([]byte, len(b))
			copy(end, b)
			for i := len(end) - 1; i >= 0; i-- {
				end[i] = end[i] + 1
				if end[i] != 0 {
					return end[:i+1]
				}
			}
			return nil // no upper-bound
		}

		prefixIterOptions := func(prefix []byte) *pebble.IterOptions {
			return &pebble.IterOptions{
				LowerBound: prefix,
				UpperBound: keyUpperBound(prefix),
			}
		}
		iter, _ := c.db.NewIter(prefixIterOptions([]byte(keyPrefix)))
		defer iter.Close()

		for iter.First(); iter.Valid(); iter.Next() {
			if !strings.HasPrefix(string(iter.Key()), keyPrefix) {
				break
			}

			/* Extract the csum part of the key, this avoids decoding the full
			 * entry later on if that's the only thing we need */
			key := iter.Key()
			hex_csum := string(key[bytes.LastIndexByte(key, byte(':'))+1:])
			csum, _ := hex.DecodeString(hex_csum)

			if !yield(objects.MAC(csum), iter.Value()) {
				return
			}
		}
	}
}

func (c *_RepositoryCache) GetDeltas() iter.Seq2[objects.MAC, []byte] {
	return c.getObjects("__delta__:")
}

func (c *_RepositoryCache) DelDelta(blobType resources.Type, blobCsum, packfileMAC objects.MAC) error {
	return c.delete("__delta__", fmt.Sprintf("%d:%x:%x", blobType, blobCsum, packfileMAC))
}

func (c *_RepositoryCache) PutDeleted(blobType resources.Type, blobCsum objects.MAC, data []byte) error {
	return c.put("__deleted__", fmt.Sprintf("%d:%x", blobType, blobCsum), data)
}

func (c *_RepositoryCache) HasDeleted(blobType resources.Type, blobCsum objects.MAC) (bool, error) {
	return c.has("__deleted__", fmt.Sprintf("%d:%x", blobType, blobCsum))
}

func (c *_RepositoryCache) GetDeleteds() iter.Seq2[objects.MAC, []byte] {
	return c.getObjects(fmt.Sprintf("__deleted__:"))
}

func (c *_RepositoryCache) GetDeletedsByType(blobType resources.Type) iter.Seq2[objects.MAC, []byte] {
	return c.getObjects(fmt.Sprintf("__deleted__:%d:", blobType))
}

func (c *_RepositoryCache) DelDeleted(blobType resources.Type, blobCsum objects.MAC) error {
	return c.delete("__deleted__", fmt.Sprintf("%d:%x", blobType, blobCsum))
}

func (c *_RepositoryCache) PutPackfile(packfile objects.MAC, data []byte) error {
	return c.put("__packfile__", fmt.Sprintf("%x", packfile), data)
}

func (c *_RepositoryCache) HasPackfile(packfile objects.MAC) (bool, error) {
	return c.has("__packfile__", fmt.Sprintf("%x", packfile))
}

func (c *_RepositoryCache) DelPackfile(packfile objects.MAC) error {
	return c.delete("__packfile__", fmt.Sprintf("%x", packfile))
}

func (c *_RepositoryCache) GetPackfiles() iter.Seq2[objects.MAC, []byte] {
	return c.getObjects("__packfile__:")
}

func (c *_RepositoryCache) PutConfiguration(key string, data []byte) error {
	return c.put("__configuration__", key, data)
}

func (c *_RepositoryCache) GetConfiguration(key string) ([]byte, error) {
	return c.get("__configuration__", key)
}

func (c *_RepositoryCache) GetConfigurations() iter.Seq[[]byte] {
	return func(yield func([]byte) bool) {
		keyPrefix := "__configuration__"
		iter, _ := c.db.NewIter(&pebble.IterOptions{
			LowerBound: []byte(keyPrefix + ":"),
			UpperBound: []byte(keyPrefix + ";"),
		})
		defer iter.Close()

		for iter.First(); iter.Valid(); iter.Next() {
			if !strings.HasPrefix(string(iter.Key()), keyPrefix) {
				break
			}

			if !yield(iter.Value()) {
				return
			}
		}
	}
}

func (c *_RepositoryCache) PutSnapshot(stateID objects.MAC, data []byte) error {
	return c.put("__snapshot__", fmt.Sprintf("%x", stateID), data)
}

func (c *_RepositoryCache) HasSnapshot(stateID objects.MAC) (bool, error) {
	return c.has("__snapshot__", fmt.Sprintf("%x", stateID))
}

func (c *_RepositoryCache) GetSnapshot(stateID objects.MAC) ([]byte, error) {
	return c.get("__snapshot__", fmt.Sprintf("%x", stateID))
}

func (c *_RepositoryCache) DelSnapshot(stateID objects.MAC) error {
	return c.delete("__snapshot__", fmt.Sprintf("%x", stateID))
}
