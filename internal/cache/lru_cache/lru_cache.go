// Package lru_cache implements a lru cache data structure
package lru_cache

import (
	"fmt"
	"l0/internal/cache/lru_cache/list"
	"sync"
)

// A LRUCache is a thread-safe implementation of least recently used cache
type LRUCache[K comparable, V any] struct {
	lruList  *list.LRUList[K, V]
	cache    map[K]*list.LRUListNode[K, V]
	capacity int
	mu       sync.Mutex
}

// NewLRUCache create empty cache. It should be created only using this command
func NewLRUCache[K comparable, V any](capacity int) (*LRUCache[K, V], error) {
	if capacity < 0 {
		return nil, fmt.Errorf("expected positive number for capacity, got: %d", capacity)
	}
	return &LRUCache[K, V]{lruList: list.NewLRUList[K, V](), cache: make(map[K]*list.LRUListNode[K, V]), capacity: capacity}, nil

}

// Set add a new key-value pair to cache, might evict some old pairs
func (c *LRUCache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	node, ok := c.cache[key]
	if ok {
		c.lruList.Remove(node)
	}

	if c.lruList.Size() == c.capacity {
		node, _ := c.lruList.PopBack()
		delete(c.cache, node.Key)
	}

	node = c.lruList.PushFront(key, value)
	c.cache[key] = node
}

// Get return a value by key and moves this pair to front
func (c *LRUCache[K, V]) Get(key K) (value V, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	node, ok := c.cache[key]
	if ok {
		c.lruList.MoveToFront(node)
		return node.Value, true
	}
	return

}

// Delete removes node by key
func (c *LRUCache[K, V]) Delete(key K) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	node, ok := c.cache[key]
	if !ok {
		return fmt.Errorf("can't delete node as no node has key %v", key)
	}
	_, err := c.lruList.Remove(node)
	delete(c.cache, key)
	return err
}

// Contains return if key is present in cache
func (c *LRUCache[K, V]) Contains(key K) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, ok := c.cache[key]
	return ok
}

// Flush clears a cache
func (c *LRUCache[K, V]) Flush() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, node := range c.cache {
		c.lruList.Remove(node)
	}
	clear(c.cache)
}

// Size returns how many elements are currently cashed
func (c *LRUCache[K, V]) Size() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.lruList.Size()
}

// Capacity returns the maximum capacity of the cache
func (c *LRUCache[K, V]) Capacity() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.capacity
}

// Empty returns if there are no elements in cache
func (c *LRUCache[K, V]) Empty() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.lruList.Size() == 0
}