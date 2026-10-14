// Package fenwick provides a Fenwick tree (binary indexed tree) for prefix
// sums over a mutable array of numbers. The public API is 0-indexed and uses
// half-open ranges like Go slices.
//
//	t := fenwick.From([]int{5, 2, 9, 1})
//	t.PrefixSum(2)    // 7  (5 + 2)
//	t.Add(1, 10)      // values are now [5 12 9 1]
//	t.RangeSum(1, 3)  // 21 (12 + 9)
//	t.LowerBound(17)  // 1  (first i with PrefixSum(i+1) >= 17)
//
// Complexity: New and From O(n); Add, Set, PrefixSum, RangeSum and LowerBound
// O(log n); Get and Len O(1).
//
// For range-minimum or other associative queries use a segment tree instead.
// A Tree is not safe for concurrent use.
package fenwick

import (
	"fmt"
	"math/bits"
	"slices"
)

// Number is the set of integer and floating-point types a Tree can hold.
type Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64
}

// Tree is a Fenwick tree over a fixed-length array of numbers.
//
// The zero value is an empty tree of length 0; use New or From to create a
// tree with elements.
type Tree[T Number] struct {
	vals []T // current values
	bit  []T // bit[j-1] is the sum of vals over (j - lowbit(j), j] (1-indexed)
}

// New returns a tree of length n with all values zero. It panics if n < 0.
// O(n).
func New[T Number](n int) *Tree[T] {
	if n < 0 {
		panic(fmt.Sprintf("fenwick: New called with negative length %d", n))
	}
	return &Tree[T]{vals: make([]T, n), bit: make([]T, n)}
}

// From returns a tree holding a copy of vals. It builds the tree in O(n),
// faster than n calls to Add. vals is not modified or retained.
func From[T Number](vals []T) *Tree[T] {
	n := len(vals)
	t := &Tree[T]{vals: slices.Clone(vals), bit: slices.Clone(vals)}
	if t.vals == nil {
		t.vals, t.bit = []T{}, []T{}
	}
	for j := 1; j <= n; j++ {
		if p := j + j&-j; p <= n {
			t.bit[p-1] += t.bit[j-1]
		}
	}
	return t
}

func (t *Tree[T]) checkIndex(i int, method string) {
	if i < 0 || i >= len(t.vals) {
		panic(fmt.Sprintf("fenwick: %s index %d out of range [0, %d)", method, i, len(t.vals)))
	}
}

// Len returns the number of elements. O(1).
func (t *Tree[T]) Len() int {
	return len(t.vals)
}

// Add adds delta to the element at index i. It panics if i is out of range.
// O(log n).
func (t *Tree[T]) Add(i int, delta T) {
	t.checkIndex(i, "Add")
	t.vals[i] += delta
	t.add(i, delta)
}

func (t *Tree[T]) add(i int, delta T) {
	n := len(t.bit)
	for j := i + 1; j <= n; j += j & -j {
		t.bit[j-1] += delta
	}
}

// Set sets the element at index i to v. It panics if i is out of range.
// O(log n).
func (t *Tree[T]) Set(i int, v T) {
	t.checkIndex(i, "Set")
	delta := v - t.vals[i]
	t.vals[i] = v
	t.add(i, delta)
}

// Get returns the element at index i. It panics if i is out of range. O(1).
func (t *Tree[T]) Get(i int) T {
	t.checkIndex(i, "Get")
	return t.vals[i]
}

// PrefixSum returns the sum of the first i elements, i.e. of the range
// [0, i). It panics unless 0 <= i <= Len(). O(log n).
func (t *Tree[T]) PrefixSum(i int) T {
	if i < 0 || i > len(t.vals) {
		panic(fmt.Sprintf("fenwick: PrefixSum index %d out of range [0, %d]", i, len(t.vals)))
	}
	return t.prefix(i)
}

func (t *Tree[T]) prefix(i int) T {
	var sum T
	for j := i; j > 0; j -= j & -j {
		sum += t.bit[j-1]
	}
	return sum
}

// RangeSum returns the sum of the elements in [l, r). It panics unless
// 0 <= l <= r <= Len(). O(log n).
func (t *Tree[T]) RangeSum(l, r int) T {
	if l < 0 || r < l || r > len(t.vals) {
		panic(fmt.Sprintf("fenwick: RangeSum range [%d, %d) invalid for length %d", l, r, len(t.vals)))
	}
	return t.prefix(r) - t.prefix(l)
}

// LowerBound returns the smallest index i such that PrefixSum(i+1) >= target,
// or Len() if there is none. It requires all elements to be non-negative
// (so that prefix sums are non-decreasing); otherwise the result is
// unspecified. A target <= 0 yields 0 when the tree is non-empty.
// Typical uses are finding the k-th smallest element of a frequency table or
// weighted random sampling. O(log n).
func (t *Tree[T]) LowerBound(target T) int {
	n := len(t.bit)
	if n == 0 {
		return 0
	}
	pos := 0
	var sum T
	for step := 1 << (bits.Len(uint(n)) - 1); step > 0; step >>= 1 {
		if next := pos + step; next <= n && sum+t.bit[next-1] < target {
			pos = next
			sum += t.bit[next-1]
		}
	}
	return pos
}

// ToSlice returns a copy of the current values. O(n).
func (t *Tree[T]) ToSlice() []T {
	return slices.Clone(t.vals)
}

// String returns the values formatted like "Tree[1 2 3]". O(n).
func (t *Tree[T]) String() string {
	return "Tree" + fmt.Sprint(t.vals)
}
