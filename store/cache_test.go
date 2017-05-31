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
