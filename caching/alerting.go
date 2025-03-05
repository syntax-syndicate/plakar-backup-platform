package caching

import (
	"fmt"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/syndtr/goleveldb/leveldb"
)

type AlertingCache struct {
	manager *Manager
	db      *leveldb.DB
}

func newAlertingCache(cacheManager *Manager, repositoryID uuid.UUID) (*AlertingCache, error) {
	cacheDir := filepath.Join(cacheManager.cacheDir, "alerting", repositoryID.String())

	db, err := leveldb.OpenFile(cacheDir, nil)
	if err != nil {
		return nil, err
	}

	return &AlertingCache{
		manager: cacheManager,
		db:      db,
	}, nil
}

func (c *AlertingCache) Close() error {
	return c.db.Close()
}

func (c *AlertingCache) put(prefix string, pathname string, data []byte) error {
	return c.db.Put([]byte(fmt.Sprintf("%s:%s", prefix, pathname)), data, nil)
}

func (c *AlertingCache) has(prefix, key string) (bool, error) {
	return c.db.Has([]byte(fmt.Sprintf("%s:%s", prefix, key)), nil)
}

func (c *AlertingCache) get(prefix, pathname string) ([]byte, error) {
	data, err := c.db.Get([]byte(fmt.Sprintf("%s:%s", prefix, pathname)), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	return data, nil
}

func (c *AlertingCache) delete(prefix, key string) error {
	return c.db.Delete([]byte(fmt.Sprintf("%s:%s", prefix, key)), nil)
}
