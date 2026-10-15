// Package set provides a generic hash set with Python-style set operations.
//
// Set is a map type, so len(s), for x := range s and s == nil work as usual:
//
//	s := set.New(1, 2, 3)
//	s.Add(4)
//	s.Contains(2)                    // true
//	s.Intersection(set.New(2, 4, 6)) // Set{2, 4}
//	for x := range s { ... }         // unordered
//	set.Sorted(s)                    // [1 2 3 4]
//
// Like any map, a nil Set can be read from but not written to; use New or
// make(set.Set[T]) to create one. Methods that return a new Set never modify
// their receiver or arguments. A Set is not safe for concurrent use.
//
// Add, Remove and Contains are O(1) on average.
package set

import (
	"cmp"
	"fmt"
	"iter"
	"maps"
	"slices"
	"strings"
)

// Set is an unordered collection of unique comparable values.
type Set[T comparable] map[T]struct{}

// New returns a set containing vals (duplicates are ignored). O(len(vals)).
func New[T comparable](vals ...T) Set[T] {
	s := make(Set[T], len(vals))
	s.Add(vals...)
	return s
}

// Collect returns a set of the values yielded by seq. O(n).
func Collect[T comparable](seq iter.Seq[T]) Set[T] {
	s := make(Set[T])
	for v := range seq {
		s[v] = struct{}{}
	}
	return s
}

// Add inserts vals into the set. O(1) average per value.
func (s Set[T]) Add(vals ...T) {
	for _, v := range vals {
		s[v] = struct{}{}
	}
}

// Remove deletes vals from the set. Values not in the set are ignored.
// O(1) average per value.
func (s Set[T]) Remove(vals ...T) {
	for _, v := range vals {
		delete(s, v)
	}
}

// Contains reports whether v is in the set. O(1) average.
func (s Set[T]) Contains(v T) bool {
	_, ok := s[v]
	return ok
}

// ContainsAll reports whether every one of vals is in the set. It returns true
// if vals is empty. O(len(vals)).
func (s Set[T]) ContainsAll(vals ...T) bool {
	for _, v := range vals {
		if !s.Contains(v) {
			return false
		}
	}
	return true
}

// ContainsAny reports whether at least one of vals is in the set. It returns
// false if vals is empty. O(len(vals)).
func (s Set[T]) ContainsAny(vals ...T) bool {
	for _, v := range vals {
		if s.Contains(v) {
			return true
		}
	}
	return false
}

// Len returns the number of elements; same as len(s). O(1).
func (s Set[T]) Len() int { return len(s) }

// IsEmpty reports whether the set has no elements. O(1).
func (s Set[T]) IsEmpty() bool { return len(s) == 0 }

// Clear removes all elements. O(n).
func (s Set[T]) Clear() { clear(s) }

// Clone returns a shallow copy of the set. Cloning a nil set returns an empty,
// non-nil set. O(n).
func (s Set[T]) Clone() Set[T] {
	if s == nil {
		return make(Set[T])
	}
	return maps.Clone(s)
}

// Union returns a new set with the elements of s and all others
// (s | others...). O(len(s) + sum of len(others)).
func (s Set[T]) Union(others ...Set[T]) Set[T] {
	res := s.Clone()
	for _, o := range others {
		for v := range o {
			res[v] = struct{}{}
		}
	}
	return res
}

// Intersection returns a new set with the elements of s that are in every one
// of others (s & others...). With no others it returns a copy of s.
// O(len(s) * len(others)).
func (s Set[T]) Intersection(others ...Set[T]) Set[T] {
	res := make(Set[T])
outer:
	for v := range s {
		for _, o := range others {
			if !o.Contains(v) {
				continue outer
			}
		}
		res[v] = struct{}{}
	}
	return res
}

// Difference returns a new set with the elements of s that are in none of
// others (s - others...). O(len(s) * len(others)).
func (s Set[T]) Difference(others ...Set[T]) Set[T] {
	res := make(Set[T])
outer:
	for v := range s {
		for _, o := range others {
			if o.Contains(v) {
				continue outer
			}
		}
		res[v] = struct{}{}
	}
	return res
}

// SymmetricDifference returns a new set with the elements in exactly one of s
// and other (s ^ other). O(len(s) + len(other)).
func (s Set[T]) SymmetricDifference(other Set[T]) Set[T] {
	res := make(Set[T])
	for v := range s {
		if !other.Contains(v) {
			res[v] = struct{}{}
		}
	}
	for v := range other {
		if !s.Contains(v) {
			res[v] = struct{}{}
		}
	}
	return res
}

// IsSubset reports whether every element of s is in other (s <= other).
// O(len(s)).
func (s Set[T]) IsSubset(other Set[T]) bool {
	if len(s) > len(other) {
		return false
	}
	for v := range s {
		if !other.Contains(v) {
			return false
		}
	}
	return true
}

// IsSuperset reports whether every element of other is in s (s >= other).
// O(len(other)).
func (s Set[T]) IsSuperset(other Set[T]) bool {
	return other.IsSubset(s)
}

// IsDisjoint reports whether s and other have no elements in common.
// O(min(len(s), len(other))).
func (s Set[T]) IsDisjoint(other Set[T]) bool {
	small, big := s, other
	if len(small) > len(big) {
		small, big = big, small
	}
	for v := range small {
		if big.Contains(v) {
			return false
		}
	}
	return true
}

// Equal reports whether s and other contain exactly the same elements.
// A nil set equals an empty set. O(len(s)).
func (s Set[T]) Equal(other Set[T]) bool {
	return len(s) == len(other) && s.IsSubset(other)
}

// Pop removes and returns an arbitrary element. It panics if the set is empty.
// O(1) average.
func (s Set[T]) Pop() T {
	v, ok := s.TryPop()
	if !ok {
		panic("set: Pop called on empty Set")
	}
	return v
}

// TryPop removes and returns an arbitrary element and true, or the zero value
// and false if the set is empty. O(1) average.
func (s Set[T]) TryPop() (T, bool) {
	for v := range s {
		delete(s, v)
		return v, true
	}
	var zero T
	return zero, false
}

// All returns an iterator over the elements in unspecified order.
// It is equivalent to maps.Keys(s).
func (s Set[T]) All() iter.Seq[T] {
	return maps.Keys(s)
}

// ToSlice returns the elements in unspecified order. Use Sorted for a
// deterministic order. O(n).
func (s Set[T]) ToSlice() []T {
	out := make([]T, 0, len(s))
	for v := range s {
		out = append(out, v)
	}
	return out
}

// String returns a deterministic representation such as "Set{1, 2, 3}", with
// elements ordered by their fmt.Sprint representation (so 10 sorts before 2).
// O(n log n).
func (s Set[T]) String() string {
	strs := make([]string, 0, len(s))
	for v := range s {
		strs = append(strs, fmt.Sprint(v))
	}
	slices.Sort(strs)
	return "Set{" + strings.Join(strs, ", ") + "}"
}

// Sorted returns the elements of s in ascending order. O(n log n).
func Sorted[T cmp.Ordered](s Set[T]) []T {
	out := s.ToSlice()
	slices.Sort(out)
	return out
}
