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
	interval  time.Duration
	stopCh    chan struct{}
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
	cache := &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     map[string]*list.Element{},
		OnEvicted: onEvicted,
		interval:  time.Second,
		stopCh:    make(chan struct{}),
	}
	go cache.evictExpiredItems()
	return cache
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
func (c *Cache) Add(key string, expire time.Duration, value Value) {
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
		ele := c.ll.PushFront(&entry{key: key, value: value, expire: time.Now().Add(expire)})
		c.cache[key] = ele
		c.nBytes += int64(len(key)) + int64(value.Len())
	}
	for c.nBytes > c.maxBytes && c.maxBytes != 0 {
		c.RemoveOldest()
	}
}

// 检测lru链表中过期的元素
func (c *Cache) removeElement(e *list.Element) {
	kv := e.Value.(*entry)
	c.ll.Remove(e)
	delete(c.cache, kv.key)
	c.nBytes -= int64(len(kv.key)) + int64(kv.value.Len())
	if c.OnEvicted != nil {
		c.OnEvicted(kv.key, kv.value)
	}
}
func (c *Cache) evictExpiredItems() {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			for e := c.ll.Back(); e != nil; e = e.Prev() {
				kv := e.Value.(*entry)
				if !kv.expire.IsZero() && time.Now().After(kv.expire) {
					c.removeElement(e)
				} else {
					break
				}
			}
		case <-c.stopCh:
			return
		}
	}
}
func (c *Cache) Stop() {
	close(c.stopCh)
}
