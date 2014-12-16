// This package provides a simple LRU cache. It is based on the
// LRU implementation in groupcache:
// https://github.com/golang/groupcache/tree/master/lru
package lru

import (
	"container/list"
	//"errors"
	"fmt"
	"sync"
)

var (
	//ErrKeyExists = fmt.Errorf("item already exists")
	ErrInvalidSize = fmt.Errorf("Must provide a positive size")

//ErrCacheMiss = fmt.Errorf("item not found")
)

// Cache is a thread-safe fixed size LRU cache.
type Cache struct {
	maxEntries int
	evictList  *list.List
	items      map[interface{}]*list.Element

	// OnEvicted optionally specificies a callback function to be
	// executed when an entry is purged from the cache.
	OnEvicted func(key Key, value interface{})

	// todo: experiment with a full Mutex and RWMutex
	//lock sync.Mutex
	lock sync.RWMutex
}

// A Key may be any value that is comparable. See http://golang.org/ref/spec#Comparison_operators
type Key interface{}

// entry is used to hold a value in the evictList
type entry struct {
	key   Key
	value interface{}
}

// New creates an LRU of the given size
func New(size int) (*Cache, error) {
	if size < 0 {
		return nil, ErrInvalidSize
	}
	c := &Cache{
		maxEntries: size,
		evictList:  list.New(),
		items:      make(map[interface{}]*list.Element, size),
	}
	return c, nil
}

// Purge is used to completely clear the cache
func (c *Cache) Purge() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.evictList = list.New()
	c.items = make(map[interface{}]*list.Element, c.maxEntries)
}

// Add adds a value to the cache.
func (c *Cache) Add(key Key, value interface{}) {
	c.lock.Lock()
	defer c.lock.Unlock()

	// Check for existing item
	if ent, ok := c.items[key]; ok {
		c.evictList.MoveToFront(ent)
		ent.Value.(*entry).value = value
		return
	}

	// Add new item
	entry := c.evictList.PushFront(&entry{key, value})
	c.items[key] = entry

	// Verify size not exceeded
	if c.maxEntries != 0 && c.evictList.Len() > c.maxEntries {
		c.removeOldest()
	}
}

// Get looks up a key's value from the cache.
func (c *Cache) Get(key Key) (value interface{}, ok bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if ent, ok := c.items[key]; ok {
		c.evictList.MoveToFront(ent)
		return ent.Value.(*entry).value, true
	}
	return
}

// Remove removes the provided key from the cache.
func (c *Cache) Remove(key Key) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if ent, ok := c.items[key]; ok {
		c.removeElement(ent)
	}
}

// RemoveOldest removes the oldest item from the cache.
func (c *Cache) RemoveOldest() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.removeOldest()
}

// Keys returns a slice of the keys in the cache.
func (c *Cache) Keys() []interface{} {
	c.lock.Lock()
	defer c.lock.Unlock()

	keys := make([]interface{}, len(c.items))
	i := 0
	for k := range c.items {
		keys[i] = k
		i++
	}

	return keys
}

// removeOldest removes the oldest item from the cache.
func (c *Cache) removeOldest() {
	ent := c.evictList.Back()
	if ent != nil {
		c.removeElement(ent)
	}
}

// removeElement is used to remove a given list element from the cache
func (c *Cache) removeElement(e *list.Element) {
	c.evictList.Remove(e)
	kv := e.Value.(*entry)
	delete(c.items, kv.key)
	if c.OnEvicted != nil {
		c.OnEvicted(kv.key, kv.value)
	}
}

// Len returns the number of items in the cache.
func (c *Cache) Len() int {
	c.lock.RLock()
	defer c.lock.RUnlock()
	//c.lock.Lock()
	//defer c.lock.Unlock()
	return c.evictList.Len()
}
