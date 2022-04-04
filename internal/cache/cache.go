package cache

import (
	"time"

	"github.com/dgraph-io/ristretto"
)

const _TTL = time.Minute * 10

type Cache struct {
	cache *ristretto.Cache
}

func NewCache() (*Cache, error) {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // number of keys to track frequency of (10M).
		MaxCost:     1 << 20, // maximum cost of cache (1GB).
		BufferItems: 64,      // number of keys per Get buffer.
	})
	if err != nil {
		panic(err)
	}

	return &Cache{
		cache: cache,
	}, nil
}

func (c *Cache) Set(key interface{}, value interface{}) {
	c.cache.SetWithTTL(key, value, 1, _TTL)
}

func (c *Cache) Get(key interface{}) (interface{}, bool) {
	return c.cache.Get(key)
}

func (c *Cache) SetWithTTL(key interface{}, value interface{}, ttl time.Duration) {
	c.cache.SetWithTTL(key, value, 1, ttl)
}

func (c *Cache) Clear() {
	c.cache.Close()
}
