package service

import (
	"sync"
	"time"
)

type cacheEntry struct {
	data      *SalePagePublishData
	createdAt time.Time
}

type salePageCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	ttl     time.Duration
}

func newSalePageCache(ttl time.Duration) *salePageCache {
	return &salePageCache{
		entries: make(map[string]cacheEntry),
		ttl:     ttl,
	}
}

func (c *salePageCache) Get(slug string) (*SalePagePublishData, bool) {
	c.mu.RLock()
	entry, ok := c.entries[slug]
	c.mu.RUnlock()
	if !ok || time.Since(entry.createdAt) > c.ttl {
		return nil, false
	}
	return entry.data, true
}

func (c *salePageCache) Set(slug string, data *SalePagePublishData) {
	c.mu.Lock()
	c.entries[slug] = cacheEntry{data: data, createdAt: time.Now()}
	c.mu.Unlock()
}

func (c *salePageCache) Invalidate(slug string) {
	c.mu.Lock()
	delete(c.entries, slug)
	c.mu.Unlock()
}
