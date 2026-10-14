package sorted

import (
	"cmp"
	"iter"
)

// List is a sorted list that allows duplicates (a sorted multiset), like
// Python's sortedcontainers.SortedList or C++'s std::multiset. Elements are
// kept sorted by the list's comparator; elements that compare equal are kept
// in insertion order. It supports O(log n) indexing, which makes problems
// such as a sliding-window median straightforward.
//
// The zero value is not usable; create lists with NewList or NewListFunc.
type List[T any] struct {
	t treap[T, struct{}]
}

// NewList returns a List ordered by cmp.Compare containing vals.
// O(k log k) for k values.
func NewList[T cmp.Ordered](vals ...T) *List[T] {
	return NewListFunc(cmp.Compare[T], vals...)
}

// NewListFunc returns a List ordered by cmp containing vals. cmp must return
// a negative number when a < b, zero when a == b and a positive number when
// a > b (the same shape as cmp.Compare). O(k log k) for k values.
func NewListFunc[T any](cmp func(a, b T) int, vals ...T) *List[T] {
	checkCmp(cmp, "NewListFunc")
	l := &List[T]{t: newTreap[T, struct{}](cmp, true)}
	l.Add(vals...)
	return l
}

// Add inserts each value, keeping the list sorted. A value equal to existing
// elements is placed after them. O(log n) per value.
func (l *List[T]) Add(vals ...T) {
	for _, v := range vals {
		l.t.put(v, struct{}{})
	}
}

// Remove deletes one element equal to v (the earliest inserted one if there
// are several) and reports whether one was found. O(log n).
func (l *List[T]) Remove(v T) bool {
	return l.t.remove(v) != nil
}

// RemoveAll deletes every element equal to v and returns how many were
// removed. O(log n).
func (l *List[T]) RemoveAll(v T) int {
	return l.t.removeAll(v)
}

// Count returns the number of elements equal to v. O(log n).
func (l *List[T]) Count(v T) int {
	return l.t.rank(v, true) - l.t.rank(v, false)
}

// Contains reports whether an element equal to v is in the list. O(log n).
func (l *List[T]) Contains(v T) bool {
	return l.t.find(v) != nil
}

// Len returns the number of elements, counting duplicates. O(1).
func (l *List[T]) Len() int {
	return l.t.len()
}

// IsEmpty reports whether the list has no elements. O(1).
func (l *List[T]) IsEmpty() bool {
	return l.t.len() == 0
}

// Clear removes all elements. O(1).
func (l *List[T]) Clear() {
	l.t.clear()
}

// At returns the i-th smallest element (0-based, counting duplicates). It
// panics if i is not in [0, Len()). O(log n).
func (l *List[T]) At(i int) T {
	checkIndex(i, l.t.len(), "List.At")
	return l.t.at(i).key
}

// BisectLeft returns the number of elements strictly less than v, which is
// the index at which v would be inserted before any equal elements.
// O(log n).
func (l *List[T]) BisectLeft(v T) int {
	return l.t.rank(v, false)
}

// BisectRight returns the number of elements less than or equal to v, which
// is the index at which v would be inserted after any equal elements.
// O(log n).
func (l *List[T]) BisectRight(v T) int {
	return l.t.rank(v, true)
}

// Min returns the smallest element, or ok == false if the list is empty.
// O(log n).
func (l *List[T]) Min() (v T, ok bool) {
	return keyOf(l.t.min())
}

// Max returns the largest element, or ok == false if the list is empty.
// O(log n).
func (l *List[T]) Max() (v T, ok bool) {
	return keyOf(l.t.max())
}

// PopMin removes and returns the smallest element, or ok == false if the
// list is empty. O(log n).
func (l *List[T]) PopMin() (v T, ok bool) {
	if l.t.len() == 0 {
		return keyOf[T, struct{}](nil)
	}
	return keyOf(l.t.removeAt(0))
}

// PopMax removes and returns the largest element, or ok == false if the list
// is empty. O(log n).
func (l *List[T]) PopMax() (v T, ok bool) {
	if l.t.len() == 0 {
		return keyOf[T, struct{}](nil)
	}
	return keyOf(l.t.removeAt(l.t.len() - 1))
}

// Floor returns the greatest element <= v, or ok == false if there is none.
// O(log n).
func (l *List[T]) Floor(v T) (T, bool) {
	return keyOf(l.t.below(v, true))
}

// Ceiling returns the least element >= v, or ok == false if there is none.
// O(log n).
func (l *List[T]) Ceiling(v T) (T, bool) {
	return keyOf(l.t.above(v, true))
}

// Lower returns the greatest element < v, or ok == false if there is none.
// O(log n).
func (l *List[T]) Lower(v T) (T, bool) {
	return keyOf(l.t.below(v, false))
}

// Higher returns the least element > v, or ok == false if there is none.
// O(log n).
func (l *List[T]) Higher(v T) (T, bool) {
	return keyOf(l.t.above(v, false))
}

// All returns an iterator over the elements in ascending order, including
// duplicates. Each step costs amortized O(1).
func (l *List[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		l.t.each(func(n *node[T, struct{}]) bool { return yield(n.key) })
	}
}

// Backward returns an iterator over the elements in descending order,
// including duplicates. Each step costs amortized O(1).
func (l *List[T]) Backward() iter.Seq[T] {
	return func(yield func(T) bool) {
		l.t.descend(func(n *node[T, struct{}]) bool { return yield(n.key) })
	}
}

// ToSlice returns the elements in ascending order as a new slice. O(n).
func (l *List[T]) ToSlice() []T {
	return l.t.keys()
}

// String returns the list formatted as "List{a, b, b, c}" in ascending
// order. O(n).
func (l *List[T]) String() string {
	return formatSeq("List", &l.t)
}
