package cache

import (
	"github.com/treeforest/cache/lru"
	"sync"
)

// 实现对lru的并发控制
type cache struct {
	mu         sync.Mutex
	lru        *lru.Cache
	cacheBytes int64
}

func (c *cache) add(key string, value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.lru == nil {
		c.lru = lru.New(c.cacheBytes, nil)
	}

	c.lru.Add(key, value)
}

func (c *cache) get(key string) (value ByteView, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.lru == nil {
		return ByteView{}, false
	}

	if v, ok := c.lru.Get(key); ok {
		return v.(ByteView), true
	}

	return ByteView{}, false
}
