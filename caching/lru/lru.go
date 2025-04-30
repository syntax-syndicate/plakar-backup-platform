package lru

import (
	"sync"
	"sync/atomic"
)

type node[K comparable] struct {
	key  K
	next *node[K]
}

type Cache[K comparable, V any] struct {
	mtx sync.RWMutex

	size   int
	target int

	onevict func(K, V) error

	items map[K]V
	head  *node[K]
	tail  *node[K]

	hits   atomic.Uint64
	misses atomic.Uint64
}

func New[K comparable, V any](target int, onevict func(K, V) error) *Cache[K, V] {
	return &Cache[K, V]{
		target:  target,
		onevict: onevict,
		items:   make(map[K]V, target),
	}
}

func (c *Cache[K, V]) put(key K, val V) {
	c.size++
	c.items[key] = val

	n := &node[K]{key: key}
	if c.head == nil {
		c.head = n
		c.tail = n
	} else {
		c.tail.next = n
		c.tail = n
	}
}

// assume that the item was just removed from the linked list
func (c *Cache[K, V]) flush(key K) error {
	val := c.items[key]
	if c.onevict != nil {
		if err := c.onevict(key, val); err != nil {
			return err
		}
	}

	delete(c.items, key)
	c.size--
	return nil
}

func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mtx.RLock()
	val, ok := c.items[key]
	c.mtx.RUnlock()

	if ok {
		c.hits.Add(1)
	} else {
		c.misses.Add(1)
	}

	return val, ok
}

// Put adds or overrides an element in the cache.
func (c *Cache[K, V]) Put(key K, val V) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if _, ok := c.items[key]; ok {
		c.items[key] = val
		return nil
	}

	if c.size == c.target {
		if err := c.flush(c.head.key); err != nil {
			return err
		}
		c.head = c.head.next
	}

	c.put(key, val)
	return nil
}

func (c *Cache[K, V]) Close() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	var err error
	for n := c.head; n != nil; n = n.next {
		if e := c.flush(n.key); e != nil {
			err = e
		}
	}
	c.head = nil
	c.tail = nil
	return err
}

func (c *Cache[K, V]) Stats() (hit, miss, size uint64) {
	return c.hits.Load(), c.misses.Load(), uint64(c.size)
}
