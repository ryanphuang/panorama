package store

import (
	"sync"
	"time"

	du "deephealth/util"
)

type CacheItem struct {
	expires time.Time
	value   interface{}
}

type Cache struct {
	sync.RWMutex
	items map[string]*CacheItem
	ttl   time.Duration
}

const (
	RETIRE_LIST_LEN  = 100
	EXPIRE_WRITE_LEN = 50
	ctag             = "cache"
)

var (
	retired = make([]string, RETIRE_LIST_LEN)
)

func NewCache(ttl time.Duration) *Cache {
	c := &Cache{
		ttl:   ttl,
		items: make(map[string]*CacheItem),
	}
	return c
}

func (i *CacheItem) Expired() bool {
	d := time.Now().Sub(i.expires)
	return d >= 0
}

func (c *Cache) Get(key string) interface{} {
	c.RLock()
	item, ok := c.items[key]
	c.RUnlock()
	if !ok {
		return nil
	}
	if item.Expired() {
		c.Lock()
		delete(c.items, key) // delete expired cache item from cache
		c.Unlock()
		du.LogD(ctag, "Entry for %s has expired (added at %s)", key, item.expires.Add(-1*c.ttl))
		return nil
	}
	return item.value
}

func (c *Cache) Set(key string, value interface{}) {
	item := &CacheItem{
		expires: time.Now().Add(c.ttl),
		value:   value,
	}
	c.Lock()
	c.items[key] = item
	c.Unlock()
	if len(c.items) >= EXPIRE_WRITE_LEN {
		go c.reap() // trigger reaping when there are too many entries
	}
}

func (c *Cache) Delete(key string) {
	c.Lock()
	delete(c.items, key)
	c.Unlock()
}

func (c *Cache) Clear() {
	c.Lock()
	c.items = make(map[string]*CacheItem)
	c.Unlock()
}

func (c *Cache) reap() {
	c.RLock()
	i := 0
	for key, item := range c.items {
		if item.Expired() {
			if i >= RETIRE_LIST_LEN {
				break
			}
			retired[i] = key
			i++
		}
	}
	c.Unlock()
	if i == 0 { // no item has expired yet
		return
	}
	c.Lock()
	for j := 0; j < i; j++ {
		delete(c.items, retired[j])
	}
	c.Unlock()
}
