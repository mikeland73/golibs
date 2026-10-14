// Package lru provides a fixed-capacity least-recently-used cache.
//
//	c := lru.New[string, int](2)
//	c.Put("a", 1)
//	c.Put("b", 2)
//	c.Get("a")    // 1, true — "a" is now most recently used
//	c.Put("c", 3) // evicts "b", the least recently used
//	c.Keys()      // [c a]
//
// It is implemented with a map plus an intrusive doubly linked list, so Get,
// Peek, Put, Remove and Contains are all O(1).
//
// A Cache is NOT safe for concurrent use; guard it with a sync.Mutex if it is
// shared between goroutines.
package lru

import (
	"fmt"
	"iter"
	"strings"
)

type node[K comparable, V any] struct {
	key        K
	val        V
	prev, next *node[K, V]
}

// Cache is a fixed-capacity LRU cache mapping keys of type K to values of type V.
// Create one with New; the zero value is not usable.
type Cache[K comparable, V any] struct {
	capacity int
	items    map[K]*node[K, V]
	// root is a sentinel of a circular list: root.next is the most recently
	// used entry and root.prev the least recently used.
	root node[K, V]
}

// New returns an empty cache that holds at most capacity entries.
// It panics if capacity <= 0.
func New[K comparable, V any](capacity int) *Cache[K, V] {
	if capacity <= 0 {
		panic(fmt.Sprintf("lru: New called with non-positive capacity %d", capacity))
	}
	c := &Cache[K, V]{capacity: capacity}
	c.Clear()
	return c
}

// Get returns the value for k and true, marking k as most recently used.
// If k is not present it returns the zero value and false. O(1).
func (c *Cache[K, V]) Get(k K) (V, bool) {
	n, ok := c.items[k]
	if !ok {
		var zero V
		return zero, false
	}
	c.moveToFront(n)
	return n.val, true
}

// Peek returns the value for k and true without changing its recency.
// If k is not present it returns the zero value and false. O(1).
func (c *Cache[K, V]) Peek(k K) (V, bool) {
	n, ok := c.items[k]
	if !ok {
		var zero V
		return zero, false
	}
	return n.val, true
}

// Put sets the value for k and marks it as most recently used. If k is new
// and the cache is full, the least recently used entry is evicted and
// returned with evicted == true. Updating an existing key never evicts.
// Callers that don't care about evictions can ignore the results. O(1).
func (c *Cache[K, V]) Put(k K, v V) (evictedKey K, evictedVal V, evicted bool) {
	if n, ok := c.items[k]; ok {
		n.val = v
		c.moveToFront(n)
		return evictedKey, evictedVal, false
	}
	var n *node[K, V]
	if len(c.items) >= c.capacity {
		// Reuse the least recently used node.
		n = c.root.prev
		evictedKey, evictedVal, evicted = n.key, n.val, true
		delete(c.items, n.key)
		unlink(n)
	} else {
		n = &node[K, V]{}
	}
	n.key, n.val = k, v
	c.items[k] = n
	c.pushFront(n)
	return evictedKey, evictedVal, evicted
}

// Remove deletes k from the cache and reports whether it was present. O(1).
func (c *Cache[K, V]) Remove(k K) bool {
	n, ok := c.items[k]
	if !ok {
		return false
	}
	delete(c.items, k)
	unlink(n)
	return true
}

// Contains reports whether k is in the cache without changing its recency. O(1).
func (c *Cache[K, V]) Contains(k K) bool {
	_, ok := c.items[k]
	return ok
}

// Len returns the number of entries in the cache. O(1).
func (c *Cache[K, V]) Len() int { return len(c.items) }

// Cap returns the maximum number of entries the cache can hold. O(1).
func (c *Cache[K, V]) Cap() int { return c.capacity }

// Clear removes all entries. O(1) (old entries are left to the garbage collector).
func (c *Cache[K, V]) Clear() {
	c.items = make(map[K]*node[K, V])
	c.root.next = &c.root
	c.root.prev = &c.root
}

// All returns an iterator over the entries from most to least recently used.
// Iterating does not change recency. The cache must not be modified during
// iteration.
func (c *Cache[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for n := c.root.next; n != &c.root; n = n.next {
			if !yield(n.key, n.val) {
				return
			}
		}
	}
}

// Keys returns the keys from most to least recently used. O(n).
func (c *Cache[K, V]) Keys() []K {
	keys := make([]K, 0, len(c.items))
	for k := range c.All() {
		keys = append(keys, k)
	}
	return keys
}

// String returns the entries from most to least recently used, e.g.
// "LRU[c:3 a:1]". O(n).
func (c *Cache[K, V]) String() string {
	var b strings.Builder
	b.WriteString("LRU[")
	first := true
	for k, v := range c.All() {
		if !first {
			b.WriteByte(' ')
		}
		first = false
		fmt.Fprintf(&b, "%v:%v", k, v)
	}
	b.WriteByte(']')
	return b.String()
}

func (c *Cache[K, V]) pushFront(n *node[K, V]) {
	n.prev = &c.root
	n.next = c.root.next
	c.root.next.prev = n
	c.root.next = n
}

func (c *Cache[K, V]) moveToFront(n *node[K, V]) {
	if c.root.next == n {
		return
	}
	unlink(n)
	c.pushFront(n)
}

func unlink[K comparable, V any](n *node[K, V]) {
	n.prev.next = n.next
	n.next.prev = n.prev
	n.prev, n.next = nil, nil
}
