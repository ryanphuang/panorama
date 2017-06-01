package store

import (
	"sync"
	"time"

	du "deephealth/util"
)

type CacheItem struct {
	expires time.Time
	Value   interface{}
}

type CacheBase struct {
	sync.RWMutex
	ttl time.Duration
}

type Cache struct {
	CacheBase
	items map[string]*CacheItem
}

type CacheListItem struct {
	chain []*CacheItem
}

type CacheList struct {
	CacheBase
	items        map[string]*CacheListItem
	max_list_len int
}

const (
	RETIRE_LIST_LEN  = 100
	EXPIRE_WRITE_LEN = 50
	ctag             = "cache"
)

var (
	retired = make([]string, RETIRE_LIST_LEN)
)

func NewCacheList(ttl time.Duration, max_list_len int) *CacheList {
	return &CacheList{
		CacheBase:    CacheBase{ttl: ttl},
		items:        make(map[string]*CacheListItem),
		max_list_len: max_list_len,
	}
}

func NewCache(ttl time.Duration) *Cache {
	c := &Cache{
		CacheBase: CacheBase{ttl: ttl},
		items:     make(map[string]*CacheItem),
	}
	return c
}

func (i *CacheItem) Expired() bool {
	d := time.Now().Sub(i.expires)
	return d >= 0
}

func (i *CacheItem) TTL() time.Duration {
	return time.Now().Sub(i.expires)
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
	return item.Value
}

func (c *Cache) Set(key string, value interface{}) {
	item := &CacheItem{
		expires: time.Now().Add(c.ttl),
		Value:   value,
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

func (c *CacheList) Process(key string, fn func(*CacheItem) bool) int {
	c.Lock()
	defer c.Unlock()
	litem, ok := c.items[key]
	if !ok {
		return 0
	}
	var i, j, processed int
	var item *CacheItem
	newchain := make([]*CacheItem, 0, c.max_list_len+1)
	j = 0
	processed = 0
	for i = 0; i < len(litem.chain); i++ {
		item = litem.chain[i]
		if !item.Expired() {
			continue
		}
		processed++
		if !fn(item) {
			// only retain them if the processing function says so
			newchain[j] = item
			j++
		}
	}
	du.LogD(ctag, "%d entries remaining after expiring and processing the list", j)
	litem.chain = newchain
	return processed
}

func (c *CacheList) Get(key string) []*CacheItem {
	c.Lock()
	defer c.Unlock()
	litem, ok := c.items[key]
	if !ok {
		return nil
	}
	var i int
	var item *CacheItem
	// the items in the chain are in chronological order
	// so once we find an unexpired item, we are done
	for i = 0; i < len(litem.chain); i++ {
		item = litem.chain[i]
		if !item.Expired() {
			break
		}
	}
	if i > 0 {
		du.LogD(ctag, "%d/%d entires for %s has been expired", i, len(litem.chain), key)
		litem.chain = litem.chain[i:] // discard the expired items
	}
	return litem.chain
}

func (c *CacheList) Set(key string, value interface{}) {
	item := &CacheItem{
		expires: time.Now().Add(c.ttl),
		Value:   value,
	}
	c.Lock()
	litem, ok := c.items[key]
	if !ok {
		lst := make([]*CacheItem, 0, c.max_list_len+1)
		litem = &CacheListItem{lst}
		c.items[key] = litem
	}
	litem.chain = append(litem.chain, item)
	if len(litem.chain) > c.max_list_len {
		litem.chain = litem.chain[1:]
		du.LogD(ctag, "truncating cache list for %s to make room for %v", key, value)
	}
	c.Unlock()
}

func (c *CacheList) Delete(key string) {
	c.Lock()
	delete(c.items, key)
	c.Unlock()
}

func (c *CacheList) Empty(key string) {
	c.Lock()
	litem, ok := c.items[key]
	if ok {
		litem.chain = make([]*CacheItem, 0, c.max_list_len+1)
	}
	c.Unlock()
}
