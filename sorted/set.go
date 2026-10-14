package sorted

import (
	"cmp"
	"iter"
)

// Set is an ordered set of unique values, like Java's TreeSet or C++'s
// std::set. Values are kept sorted by the set's comparator.
//
// The zero value is not usable; create sets with NewSet or NewSetFunc.
type Set[T any] struct {
	t treap[T, struct{}]
}

// NewSet returns a Set ordered by cmp.Compare containing vals.
// O(k log k) for k values.
func NewSet[T cmp.Ordered](vals ...T) *Set[T] {
	return NewSetFunc(cmp.Compare[T], vals...)
}

// NewSetFunc returns a Set ordered by cmp containing vals. cmp must return a
// negative number when a < b, zero when a == b and a positive number when
// a > b (the same shape as cmp.Compare). Values for which cmp returns 0 are
// considered duplicates. O(k log k) for k values.
func NewSetFunc[T any](cmp func(a, b T) int, vals ...T) *Set[T] {
	checkCmp(cmp, "NewSetFunc")
	s := &Set[T]{t: newTreap[T, struct{}](cmp, false)}
	s.Add(vals...)
	return s
}

// Add inserts each value that is not already present. O(log n) per value.
func (s *Set[T]) Add(vals ...T) {
	for _, v := range vals {
		s.t.put(v, struct{}{})
	}
}

// Remove deletes v and reports whether it was present. O(log n).
func (s *Set[T]) Remove(v T) bool {
	return s.t.remove(v) != nil
}

// Contains reports whether v is in the set. O(log n).
func (s *Set[T]) Contains(v T) bool {
	return s.t.find(v) != nil
}

// Len returns the number of values. O(1).
func (s *Set[T]) Len() int {
	return s.t.len()
}

// IsEmpty reports whether the set has no values. O(1).
func (s *Set[T]) IsEmpty() bool {
	return s.t.len() == 0
}

// Clear removes all values. O(1).
func (s *Set[T]) Clear() {
	s.t.clear()
}

// Min returns the smallest value, or ok == false if the set is empty.
// O(log n).
func (s *Set[T]) Min() (v T, ok bool) {
	return keyOf(s.t.min())
}

// Max returns the largest value, or ok == false if the set is empty.
// O(log n).
func (s *Set[T]) Max() (v T, ok bool) {
	return keyOf(s.t.max())
}

// PopMin removes and returns the smallest value, or ok == false if the set
// is empty. O(log n).
func (s *Set[T]) PopMin() (v T, ok bool) {
	if s.t.len() == 0 {
		return keyOf[T, struct{}](nil)
	}
	return keyOf(s.t.removeAt(0))
}

// PopMax removes and returns the largest value, or ok == false if the set is
// empty. O(log n).
func (s *Set[T]) PopMax() (v T, ok bool) {
	if s.t.len() == 0 {
		return keyOf[T, struct{}](nil)
	}
	return keyOf(s.t.removeAt(s.t.len() - 1))
}

// Floor returns the greatest value <= v, or ok == false if there is none.
// O(log n).
func (s *Set[T]) Floor(v T) (T, bool) {
	return keyOf(s.t.below(v, true))
}

// Ceiling returns the least value >= v, or ok == false if there is none.
// O(log n).
func (s *Set[T]) Ceiling(v T) (T, bool) {
	return keyOf(s.t.above(v, true))
}

// Lower returns the greatest value < v, or ok == false if there is none.
// O(log n).
func (s *Set[T]) Lower(v T) (T, bool) {
	return keyOf(s.t.below(v, false))
}

// Higher returns the least value > v, or ok == false if there is none.
// O(log n).
func (s *Set[T]) Higher(v T) (T, bool) {
	return keyOf(s.t.above(v, false))
}

// Rank returns the number of values strictly less than v. v need not be in
// the set. O(log n).
func (s *Set[T]) Rank(v T) int {
	return s.t.rank(v, false)
}

// At returns the i-th smallest value (0-based). It panics if i is not in
// [0, Len()). O(log n).
func (s *Set[T]) At(i int) T {
	checkIndex(i, s.t.len(), "Set.At")
	return s.t.at(i).key
}

// All returns an iterator over the values in ascending order. Each step
// costs amortized O(1).
func (s *Set[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		s.t.each(func(n *node[T, struct{}]) bool { return yield(n.key) })
	}
}

// Backward returns an iterator over the values in descending order. Each
// step costs amortized O(1).
func (s *Set[T]) Backward() iter.Seq[T] {
	return func(yield func(T) bool) {
		s.t.descend(func(n *node[T, struct{}]) bool { return yield(n.key) })
	}
}

// Range returns an iterator over the values v with lo <= v < hi in ascending
// order. Locating lo costs O(log n), then each step costs amortized O(1).
func (s *Set[T]) Range(lo, hi T) iter.Seq[T] {
	return func(yield func(T) bool) {
		s.t.ascend(lo, true, hi, true, func(n *node[T, struct{}]) bool { return yield(n.key) })
	}
}

// ToSlice returns the values in ascending order as a new slice. O(n).
func (s *Set[T]) ToSlice() []T {
	return s.t.keys()
}

// String returns the set formatted as "Set{a, b, c}" in ascending order.
// O(n).
func (s *Set[T]) String() string {
	return formatSeq("Set", &s.t)
}
