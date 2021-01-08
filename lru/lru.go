package lru

import (
	"container/list"
)

// Cache is a LRU cache. It is not safe for concurrent access.
type Cache struct {
	maxBytes  int64
	usedBytes int64
	ll        *list.List
	cache     map[string]*list.Element
	// optional and executed when an entry is purged
	OnEvicted func(key string, value Value)
}

type entry struct {
	key   string
	value Value
}

// Value use Len to count how many bytes it takes
type Value interface {
	Len() int
}

// New is the Constructor of Cache
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

// Get look ups a key's Value
func (c *Cache) Get(key string)(value Value, ok bool) {
	if elem, ok := c.cache[key]; ok {
		c.ll.MoveToBack(elem) // 约定back为队尾（双向链表，队首队尾是相对的）
		kv := elem.Value.(*entry)
		return kv.value, true
	}
	return nil, false
}

// RemoveOldest removes the oldest item
func (c *Cache) RemoveOldest() {
	elem := c.ll.Front()
	if elem != nil {
		c.ll.Remove(elem)
		kv := elem.Value.(*entry)
		delete(c.cache, kv.key)
		c.usedBytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// Add adds a value to the cache
func (c *Cache) Add(key string, value Value) {
	if elem, ok := c.cache[key]; ok {
		c.ll.MoveToBack(elem)
		kv := elem.Value.(*entry)
		c.usedBytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		elem := c.ll.PushBack(&entry{key, value})
		c.cache[key] = elem
		c.usedBytes += int64(len(key)) + int64(value.Len())
	}
	for c.maxBytes > 0 && c.maxBytes < c.usedBytes {
		c.RemoveOldest()
	}
}

// Len the number of cache entries
func (c *Cache) Len() int {
	return c.ll.Len()
}