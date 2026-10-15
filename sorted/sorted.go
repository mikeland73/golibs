// Package sorted provides balanced ordered containers: Map (like Java's
// TreeMap / C++ std::map), Set (TreeSet / std::set) and List (a sorted
// multiset like Python's sortedcontainers.SortedList / std::multiset).
//
//	m := sorted.NewMap[string, int]()
//	m.Put("b", 2)
//	m.Put("a", 1)
//	k, v, _ := m.Min()         // "a", 1
//	k, _, _ = m.Ceiling("aa")  // "b"
//
//	l := sorted.NewList(5, 1, 3, 3)
//	l.At(l.Len() / 2)          // 3 (median)
//	l.BisectLeft(3)            // 1
//
// All three share one balanced binary search tree (a treap augmented with
// subtree sizes), so besides the usual lookups they support order statistics:
// Rank (number of elements smaller than a value) and At (i-th smallest
// element), both in O(log n). Unless noted otherwise every operation costs
// expected O(log n); iterating over all n elements costs O(n). The tree uses
// a deterministic priority sequence, so behavior is reproducible run to run
// (the flip side is that the O(log n) bound is not guaranteed against input
// crafted with knowledge of that sequence).
//
// Use NewMap / NewSet / NewList for cmp.Ordered types and NewMapFunc /
// NewSetFunc / NewListFunc with a comparator shaped like cmp.Compare for
// custom orderings. The zero values are not usable; always use a constructor.
//
// Mutating a container while iterating over it with All, Backward or Range
// has undefined results. The containers are not safe for concurrent use.
package sorted

import (
	"fmt"
	"strings"
)

// checkCmp panics if cmp is nil.
func checkCmp[T any](cmp func(a, b T) int, fn string) {
	if cmp == nil {
		panic("sorted: " + fn + " called with nil comparator")
	}
}

// checkIndex panics if i is not in [0, n).
func checkIndex(i, n int, method string) {
	if i < 0 || i >= n {
		panic(fmt.Sprintf("sorted: %s index %d out of range [0, %d)", method, i, n))
	}
}

// formatSeq formats elements as name{a, b, c}.
func formatSeq[K, V any](name string, t *treap[K, V]) string {
	var sb strings.Builder
	sb.WriteString(name)
	sb.WriteByte('{')
	first := true
	t.each(func(n *node[K, V]) bool {
		if !first {
			sb.WriteString(", ")
		}
		first = false
		fmt.Fprint(&sb, n.key)
		return true
	})
	sb.WriteByte('}')
	return sb.String()
}

// keyOf returns n's key and true, or the zero value and false if n is nil.
func keyOf[K, V any](n *node[K, V]) (K, bool) {
	if n == nil {
		var zero K
		return zero, false
	}
	return n.key, true
}

// entryOf returns n's key, value and true, or zero values and false if n is
// nil.
func entryOf[K, V any](n *node[K, V]) (K, V, bool) {
	if n == nil {
		var k K
		var v V
		return k, v, false
	}
	return n.key, n.val, true
}
