package lru

import (
	"container/list"
	"time"
)

// Cache is an LRU cache. It is not safe for concurrent access.
// OnEvicted is an optional callback function.
// ll is a doubly linked list.
type Cache struct {
	maxBytes  int64
	nBytes    int64
	ll        *list.List
	cache     map[string]*list.Element
	OnEvicted func(key string, value Value)
}

type Value interface {
	Len() int
}

type entry struct {
	key    string
	value  Value
	expire time.Time
}

// New creates a new Cache.
func New(maxBytes int64, onEvicted func(key string, value Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     map[string]*list.Element{},
		OnEvicted: onEvicted,
	}
}

// Get looks up a key's value from the cache.
func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, hit := c.cache[key]; hit {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	return nil, false
}

// RemoveOldest removes the oldest item from the cache.
func (c *Cache) RemoveOldest() {
	prev := c.ll.Back()
	if prev != nil {
		c.ll.Remove(prev)
		key := prev.Value.(*entry).key
		delete(c.cache, key)
		c.nBytes -= int64(len(key)) + int64(prev.Value.(*entry).value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(key, prev.Value.(*entry).value)
		}
	}
}

// Add adds a value to the cache.
func (c *Cache) Add(key string, value Value) {
	if c.cache == nil {
		c.cache = make(map[string]*list.Element)
		c.ll = list.New()
	}
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		c.nBytes += int64(len(key)) + int64(kv.value.Len())
		kv.value = value
	} else {
		ele := c.ll.PushFront(&entry{key: key, value: value})
		c.cache[key] = ele
		c.nBytes += int64(len(key)) + int64(value.Len())
	}
	for c.nBytes > c.maxBytes && c.maxBytes != 0 {
		c.RemoveOldest()
	}
}
