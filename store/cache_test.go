package store

import (
	"testing"
	"time"
)

func TestCacheExpiration(t *testing.T) {
	cache := NewCache(3 * time.Second)
	cache.Set("hello", 32)
	cache.Set("world", 3.2)
	if cache.Get("hello").(int) != 32 {
		t.Error("Expecting 32 for hello key")
	}
	if cache.Get("world").(float64) != 3.2 {
		t.Error("Expecting 3.2 for world key")
	}
	t.Log("Sleeping for 2 seconds to test expiration")
	time.Sleep(2 * time.Second)
	if cache.Get("hello") == nil {
		t.Error("hello entry should not be expired")
	}
	t.Log("Sleeping for 2 seconds to test expiration")
	time.Sleep(2 * time.Second)
	if cache.Get("world") != nil {
		t.Error("world entry should be expired by now")
	}
}

func TestCacheList(t *testing.T) {
	cache := NewCacheList(2*time.Second, 5)
	for i := 0; i < 3; i++ {
		cache.Set("hello", i+1)
	}
	items := cache.Get("hello")
	for i, item := range items {
		if item.Value != i+1 {
			t.Errorf("expecting %d, got %v\n", i+1, item.Value)
		}
	}
	for i := 3; i < 10; i++ {
		cache.Set("hello", i+1)
	}
	items = cache.Get("hello")
	for i, item := range items {
		if item.Value != i+6 {
			t.Errorf("expecting %d, got %v\n", i+6, item.Value)
		}
	}
	time.Sleep(3 * time.Second)
	t.Log("Sleeping for 3 seconds to test list expiration")
	items = cache.Get("hello")
	if len(items) != 0 {
		t.Errorf("the cache list for hello should be empty now\n")
	}
	for i := 3; i < 10; i++ {
		cache.Set("hello", i+1)
	}
	cache.Process("hello", func(item *CacheItem) bool {
		t.Logf("process entry %v\n", item.Value)
		return true
	})
}
