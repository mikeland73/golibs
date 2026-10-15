package conc

import (
	"fmt"
	"iter"
	"maps"
	"strings"
	"sync"
)

// SafeMap is a map guarded by a sync.RWMutex, safe for concurrent use. Unlike
// sync.Map it is typed and supports atomic read-modify-write via Update and
// GetOrSet.
//
// The zero value is an empty map ready to use. A SafeMap must not be copied
// after first use.
//
// Complexity: Get, Set, Delete, GetOrSet, Update, Len and IsEmpty are O(1)
// (amortized); Keys, All, Clear and String are O(n).
type SafeMap[K comparable, V any] struct {
	mu sync.RWMutex
	m  map[K]V
}

// NewSafeMap returns an empty SafeMap. It is equivalent to new(SafeMap[K, V]).
func NewSafeMap[K comparable, V any]() *SafeMap[K, V] {
	return &SafeMap[K, V]{}
}

// Get returns the value stored for k and whether it was present.
func (s *SafeMap[K, V]) Get(k K) (V, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.m[k]
	return v, ok
}

// Set stores v for k.
func (s *SafeMap[K, V]) Set(k K, v V) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.m == nil {
		s.m = make(map[K]V)
	}
	s.m[k] = v
}

// Delete removes k. It is a no-op if k is not present.
func (s *SafeMap[K, V]) Delete(k K) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, k)
}

// GetOrSet returns the existing value for k if present (loaded is true).
// Otherwise it stores v and returns it (loaded is false). The check and the
// store happen atomically.
func (s *SafeMap[K, V]) GetOrSet(k K, v V) (actual V, loaded bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if old, ok := s.m[k]; ok {
		return old, true
	}
	if s.m == nil {
		s.m = make(map[K]V)
	}
	s.m[k] = v
	return v, false
}

// Update atomically replaces the value for k with fn(old, ok), where old is
// the current value (the zero value if absent) and ok reports whether k was
// present. It returns the new value. fn runs under the write lock and must not
// call methods on s.
//
//	m.Update("hits", func(n int, _ bool) int { return n + 1 }) // counter
func (s *SafeMap[K, V]) Update(k K, fn func(old V, ok bool) V) V {
	s.mu.Lock()
	defer s.mu.Unlock()
	old, ok := s.m[k]
	v := fn(old, ok)
	if s.m == nil {
		s.m = make(map[K]V)
	}
	s.m[k] = v
	return v
}

// Len returns the number of entries.
func (s *SafeMap[K, V]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.m)
}

// IsEmpty reports whether the map has no entries.
func (s *SafeMap[K, V]) IsEmpty() bool {
	return s.Len() == 0
}

// Keys returns a snapshot of the keys in unspecified order.
func (s *SafeMap[K, V]) Keys() []K {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]K, 0, len(s.m))
	for k := range s.m {
		keys = append(keys, k)
	}
	return keys
}

// All returns an iterator over a snapshot of the entries, taken when
// iteration starts, in unspecified order. The lock is not held while the loop
// body runs, so the body may freely call methods on s; changes it makes are
// not reflected in the ongoing iteration.
func (s *SafeMap[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for k, v := range s.snapshot() {
			if !yield(k, v) {
				return
			}
		}
	}
}

// Clear removes all entries.
func (s *SafeMap[K, V]) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	clear(s.m)
}

// String returns a representation like "SafeMap[a:1 b:2]" with keys sorted
// as fmt sorts map keys.
func (s *SafeMap[K, V]) String() string {
	return "SafeMap" + strings.TrimPrefix(fmt.Sprint(s.snapshot()), "map")
}

func (s *SafeMap[K, V]) snapshot() map[K]V {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return maps.Clone(s.m)
}
