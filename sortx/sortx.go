// Package sortx provides sort-by-key helpers in the spirit of Python's
// sorted(xs, key=..., reverse=...), plus comparator combinators that plug
// into slices.SortFunc, slices.BinarySearchFunc, priority queues and anything
// else that takes a cmp func(a, b T) int.
//
//	sortx.SortBy(people, func(p Person) int { return p.Age })        // in place, stable
//	byName := sortx.SortedBy(people, func(p Person) string { return p.Name }) // copy
//	slices.SortFunc(people, sortx.Chain(
//		sortx.Desc(sortx.By(func(p Person) int { return p.Age })), // oldest first
//		sortx.By(func(p Person) string { return p.Name }),         // then by name
//	))
//	idx := sortx.ArgSort([]int{30, 10, 20}) // [1 2 0]
//
// For plain in-place sorting use slices.Sort / slices.SortFunc /
// slices.SortStableFunc; to sort the values of an iterator use slices.Sorted.
package sortx

import (
	"cmp"
	"slices"
)

// By returns a comparator that orders values by key(v) in ascending order
// using cmp.Compare (so NaNs sort first). The result can be passed to
// slices.SortFunc, slices.BinarySearchFunc, priority queues, etc.
//
// key is called twice per comparison; if it is expensive, prefer SortBy,
// which computes each key only once.
func By[T any, K cmp.Ordered](key func(T) K) func(a, b T) int {
	return func(a, b T) int {
		return cmp.Compare(key(a), key(b))
	}
}

// Desc returns a comparator that reverses the order of cmp:
// Desc(cmp)(a, b) == cmp(b, a).
func Desc[T any](cmp func(a, b T) int) func(a, b T) int {
	return func(a, b T) int {
		return cmp(b, a)
	}
}

// Chain returns a comparator that applies cmps in order and returns the first
// non-zero result, i.e. later comparators break ties of earlier ones (like
// sorting by a tuple key in Python). With no comparators every pair compares
// equal.
//
// For one-off comparators, cmp.Or achieves the same inline:
//
//	slices.SortFunc(s, func(a, b P) int {
//		return cmp.Or(cmp.Compare(a.Age, b.Age), cmp.Compare(a.Name, b.Name))
//	})
//
// Chain is lazier (later comparators only run on ties) and composes with By.
func Chain[T any](cmps ...func(a, b T) int) func(a, b T) int {
	return func(a, b T) int {
		for _, c := range cmps {
			if r := c(a, b); r != 0 {
				return r
			}
		}
		return 0
	}
}

// keyed pairs a value with its precomputed sort key.
type keyed[E any, K cmp.Ordered] struct {
	key K
	val E
}

// sortByKey sorts s in place by key, stably, calling key exactly once per
// element (decorate-sort-undecorate, like Python).
func sortByKey[S ~[]E, E any, K cmp.Ordered](s S, key func(E) K, desc bool) {
	tmp := make([]keyed[E, K], len(s))
	for i, v := range s {
		tmp[i] = keyed[E, K]{key(v), v}
	}
	if desc {
		slices.SortStableFunc(tmp, func(a, b keyed[E, K]) int { return cmp.Compare(b.key, a.key) })
	} else {
		slices.SortStableFunc(tmp, func(a, b keyed[E, K]) int { return cmp.Compare(a.key, b.key) })
	}
	for i := range tmp {
		s[i] = tmp[i].val
	}
}

// SortBy sorts s in place in ascending order of key(e). The sort is stable:
// elements with equal keys keep their original relative order. key is called
// exactly once per element, like Python's list.sort(key=...).
// It runs in O(n log n) time and O(n) extra space.
func SortBy[S ~[]E, E any, K cmp.Ordered](s S, key func(E) K) {
	sortByKey(s, key, false)
}

// SortByDesc sorts s in place in descending order of key(e). Like Python's
// list.sort(key=..., reverse=True), the sort is stable: elements with equal
// keys keep their original relative order (they are not reversed).
// It runs in O(n log n) time and O(n) extra space.
func SortByDesc[S ~[]E, E any, K cmp.Ordered](s S, key func(E) K) {
	sortByKey(s, key, true)
}

// Sorted returns a sorted copy of s in ascending order, leaving s untouched,
// like Python's sorted(s). A nil slice yields nil. It runs in O(n log n) time.
//
// To sort the values of an iterator, use slices.Sorted.
func Sorted[S ~[]E, E cmp.Ordered](s S) S {
	out := slices.Clone(s)
	slices.Sort(out)
	return out
}

// SortedBy returns a copy of s stably sorted by key(e) in ascending order,
// leaving s untouched, like Python's sorted(s, key=key). A nil slice yields
// nil. It runs in O(n log n) time.
func SortedBy[S ~[]E, E any, K cmp.Ordered](s S, key func(E) K) S {
	out := slices.Clone(s)
	SortBy(out, key)
	return out
}

// SortedFunc returns a copy of s stably sorted by the comparator cmp,
// leaving s untouched. A nil slice yields nil. It runs in O(n log n) time.
func SortedFunc[S ~[]E, E any](s S, cmp func(a, b E) int) S {
	out := slices.Clone(s)
	slices.SortStableFunc(out, cmp)
	return out
}

// ArgSort returns the indices that would sort s in ascending order, like
// numpy.argsort(kind="stable"): s[idx[0]] <= s[idx[1]] <= ... Equal elements
// keep their original relative order. s is not modified. It runs in
// O(n log n) time.
//
//	ArgSort([]int{30, 10, 20}) // [1 2 0]
func ArgSort[S ~[]E, E cmp.Ordered](s S) []int {
	idx := make([]int, len(s))
	for i := range idx {
		idx[i] = i
	}
	slices.SortStableFunc(idx, func(a, b int) int { return cmp.Compare(s[a], s[b]) })
	return idx
}
