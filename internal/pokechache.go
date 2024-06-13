package pokechache

import (
	"sync"
	"time"
)

type cacheEntry struct {
	createdAt time.Time
	val       []byte
}

type Cache struct {
	cacheData map[string]cacheEntry
	mu        *sync.RWMutex
}

func NewCache(interval time.Duration) *Cache {
	newCache := &Cache{
		cacheData: make(map[string]cacheEntry),
		mu:        &sync.RWMutex{},
	}
	go newCache.reapLoop(interval)
	return newCache
}

func (c *Cache) Add(key string, val []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cacheData[key] = cacheEntry{time.Now(), val}
}

func (c *Cache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if entry, ok := c.cacheData[key]; ok {
		return entry.val, true
	}
	return nil, false
}

func (c *Cache) reapLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			for key, entry := range c.cacheData {
				if time.Since(entry.createdAt) >= interval {
					delete(c.cacheData, key)
				}
			}
			c.mu.Unlock()
		}
	}
}
