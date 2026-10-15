package sorted

import (
	"cmp"
	"fmt"
	"iter"
	"strings"
)

// Map is an ordered map from unique keys to values, like Java's TreeMap or
// C++'s std::map. Keys are kept sorted by the map's comparator.
//
// The zero value is not usable; create maps with NewMap or NewMapFunc.
type Map[K, V any] struct {
	t treap[K, V]
}

// NewMap returns an empty Map ordered by cmp.Compare.
func NewMap[K cmp.Ordered, V any]() *Map[K, V] {
	return NewMapFunc[K, V](cmp.Compare[K])
}

// NewMapFunc returns an empty Map ordered by cmp, which must return a negative
// number when a < b, zero when a == b and a positive number when a > b (the
// same shape as cmp.Compare). Keys for which cmp returns 0 are the same key.
func NewMapFunc[K, V any](cmp func(a, b K) int) *Map[K, V] {
	checkCmp(cmp, "NewMapFunc")
	return &Map[K, V]{t: newTreap[K, V](cmp, false)}
}

// Put sets the value for key k, replacing any existing value. O(log n).
func (m *Map[K, V]) Put(k K, v V) {
	m.t.put(k, v)
}

// Get returns the value stored for k and whether it was present. O(log n).
func (m *Map[K, V]) Get(k K) (V, bool) {
	if n := m.t.find(k); n != nil {
		return n.val, true
	}
	var zero V
	return zero, false
}

// Contains reports whether k is in the map. O(log n).
func (m *Map[K, V]) Contains(k K) bool {
	return m.t.find(k) != nil
}

// Delete removes k and reports whether it was present. O(log n).
func (m *Map[K, V]) Delete(k K) bool {
	return m.t.remove(k) != nil
}

// Len returns the number of keys. O(1).
func (m *Map[K, V]) Len() int {
	return m.t.len()
}

// IsEmpty reports whether the map has no keys. O(1).
func (m *Map[K, V]) IsEmpty() bool {
	return m.t.len() == 0
}

// Clear removes all keys. O(1).
func (m *Map[K, V]) Clear() {
	m.t.clear()
}

// Min returns the smallest key and its value, or ok == false if the map is
// empty. O(log n).
func (m *Map[K, V]) Min() (k K, v V, ok bool) {
	return entryOf(m.t.min())
}

// Max returns the largest key and its value, or ok == false if the map is
// empty. O(log n).
func (m *Map[K, V]) Max() (k K, v V, ok bool) {
	return entryOf(m.t.max())
}

// PopMin removes and returns the smallest key and its value, or ok == false
// if the map is empty. O(log n).
func (m *Map[K, V]) PopMin() (k K, v V, ok bool) {
	if m.t.len() == 0 {
		return entryOf[K, V](nil)
	}
	return entryOf(m.t.removeAt(0))
}

// PopMax removes and returns the largest key and its value, or ok == false
// if the map is empty. O(log n).
func (m *Map[K, V]) PopMax() (k K, v V, ok bool) {
	if m.t.len() == 0 {
		return entryOf[K, V](nil)
	}
	return entryOf(m.t.removeAt(m.t.len() - 1))
}

// Floor returns the greatest key <= k and its value, or ok == false if there
// is none. O(log n).
func (m *Map[K, V]) Floor(k K) (key K, v V, ok bool) {
	return entryOf(m.t.below(k, true))
}

// Ceiling returns the least key >= k and its value, or ok == false if there
// is none. O(log n).
func (m *Map[K, V]) Ceiling(k K) (key K, v V, ok bool) {
	return entryOf(m.t.above(k, true))
}

// Lower returns the greatest key < k and its value, or ok == false if there
// is none. O(log n).
func (m *Map[K, V]) Lower(k K) (key K, v V, ok bool) {
	return entryOf(m.t.below(k, false))
}

// Higher returns the least key > k and its value, or ok == false if there is
// none. O(log n).
func (m *Map[K, V]) Higher(k K) (key K, v V, ok bool) {
	return entryOf(m.t.above(k, false))
}

// Rank returns the number of keys strictly less than k. k need not be in the
// map. O(log n).
func (m *Map[K, V]) Rank(k K) int {
	return m.t.rank(k, false)
}

// At returns the i-th smallest key (0-based) and its value. It panics if i is
// not in [0, Len()). O(log n).
func (m *Map[K, V]) At(i int) (K, V) {
	checkIndex(i, m.t.len(), "Map.At")
	n := m.t.at(i)
	return n.key, n.val
}

// All returns an iterator over the key/value pairs in ascending key order.
// Each step costs amortized O(1).
func (m *Map[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		m.t.each(func(n *node[K, V]) bool {
			return yield(n.key, n.val)
		})
	}
}

// Backward returns an iterator over the key/value pairs in descending key
// order. Each step costs amortized O(1).
func (m *Map[K, V]) Backward() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		m.t.descend(func(n *node[K, V]) bool {
			return yield(n.key, n.val)
		})
	}
}

// Range returns an iterator over the key/value pairs with lo <= key < hi in
// ascending key order. Locating lo costs O(log n), then each step costs
// amortized O(1).
func (m *Map[K, V]) Range(lo, hi K) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		m.t.ascend(lo, true, hi, true, func(n *node[K, V]) bool {
			return yield(n.key, n.val)
		})
	}
}

// Keys returns the keys in ascending order as a new slice. O(n).
func (m *Map[K, V]) Keys() []K {
	return m.t.keys()
}

// String returns the map formatted as "Map{k1: v1, k2: v2}" in ascending key
// order. O(n).
func (m *Map[K, V]) String() string {
	var sb strings.Builder
	sb.WriteString("Map{")
	first := true
	m.t.each(func(n *node[K, V]) bool {
		if !first {
			sb.WriteString(", ")
		}
		first = false
		fmt.Fprintf(&sb, "%v: %v", n.key, n.val)
		return true
	})
	sb.WriteByte('}')
	return sb.String()
}
