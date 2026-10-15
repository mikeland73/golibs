// Package segtree provides a generic segment tree supporting point updates and
// range queries with any associative combine function (sum, min, max, gcd,
// matrix product, ...). Indices are 0-based and ranges are half-open [l, r),
// like Go slices.
//
//	t := segtree.NewMin([]int{5, 2, 8, 1, 9})
//	t.Query(0, 3)  // 2
//	t.Set(1, 7)    // values are now [5 7 8 1 9]
//	t.Query(0, 3)  // 5
//
//	gcd := segtree.New([]int{12, 18, 24}, 0, func(a, b int) int {
//		for b != 0 { a, b = b, a%b }
//		return a
//	})
//	gcd.Query(0, 3) // 6
//
// The implementation is the iterative bottom-up tree with 2n nodes. combine
// does not need to be commutative: Query(l, r) always equals
// combine(...combine(combine(v[l], v[l+1]), v[l+2])..., v[r-1]).
//
// Empty ranges: a tree created with New or NewSum returns its identity
// element for Query(i, i). Min and max have no identity element for a general
// ordered type (there is no "largest string"), so trees created with NewMin
// or NewMax panic when queried on an empty range instead of relying on a
// sentinel value.
//
// Complexity: construction O(n); Query and Set O(log n) calls to combine;
// Get and Len O(1). A Tree is not safe for concurrent use.
package segtree

import (
	"cmp"
	"fmt"

	"github.com/mikeland73/golibs/mathx"
)

// Tree is a segment tree over a fixed-length array of values of type T.
//
// The zero value is not usable; create trees with New, NewSum, NewMin or
// NewMax.
type Tree[T any] struct {
	n           int
	nodes       []T // leaves at nodes[n:2n]; nodes[i] = combine(nodes[2i], nodes[2i+1]) for 1 <= i < n
	combine     func(a, b T) T
	identity    T
	hasIdentity bool
}

// New returns a tree over a copy of vals. combine must be associative and
// identity must satisfy combine(identity, x) == combine(x, identity) == x;
// identity is returned by Query on an empty range. vals is not modified or
// retained. O(n).
func New[T any](vals []T, identity T, combine func(a, b T) T) *Tree[T] {
	t := build(vals, combine)
	t.identity = identity
	t.hasIdentity = true
	return t
}

// NewSum returns a tree whose Query returns the sum of the range (0 for an
// empty range). O(n).
func NewSum[T mathx.Number](vals []T) *Tree[T] {
	return New(vals, 0, func(a, b T) T { return a + b })
}

// NewMin returns a tree whose Query returns the minimum of the range. Query
// panics on an empty range. O(n).
func NewMin[T cmp.Ordered](vals []T) *Tree[T] {
	return build(vals, func(a, b T) T { return min(a, b) })
}

// NewMax returns a tree whose Query returns the maximum of the range. Query
// panics on an empty range. O(n).
func NewMax[T cmp.Ordered](vals []T) *Tree[T] {
	return build(vals, func(a, b T) T { return max(a, b) })
}

func build[T any](vals []T, combine func(a, b T) T) *Tree[T] {
	if combine == nil {
		panic("segtree: New called with nil combine function")
	}
	n := len(vals)
	t := &Tree[T]{n: n, nodes: make([]T, 2*n), combine: combine}
	copy(t.nodes[n:], vals)
	for i := n - 1; i >= 1; i-- {
		t.nodes[i] = combine(t.nodes[2*i], t.nodes[2*i+1])
	}
	return t
}

func (t *Tree[T]) checkIndex(i int, method string) {
	if i < 0 || i >= t.n {
		panic(fmt.Sprintf("segtree: %s index %d out of range [0, %d)", method, i, t.n))
	}
}

// Len returns the number of elements. O(1).
func (t *Tree[T]) Len() int {
	return t.n
}

// Get returns the element at index i. It panics if i is out of range. O(1).
func (t *Tree[T]) Get(i int) T {
	t.checkIndex(i, "Get")
	return t.nodes[t.n+i]
}

// Set replaces the element at index i with v. It panics if i is out of
// range. O(log n).
func (t *Tree[T]) Set(i int, v T) {
	t.checkIndex(i, "Set")
	i += t.n
	t.nodes[i] = v
	for i >>= 1; i >= 1; i >>= 1 {
		t.nodes[i] = t.combine(t.nodes[2*i], t.nodes[2*i+1])
	}
}

// Query returns the combination of the elements in [l, r), in index order.
// It panics unless 0 <= l <= r <= Len(). For an empty range (l == r) it
// returns the identity, or panics if the tree has none (NewMin, NewMax).
// O(log n).
func (t *Tree[T]) Query(l, r int) T {
	if l < 0 || r < l || r > t.n {
		panic(fmt.Sprintf("segtree: Query range [%d, %d) invalid for length %d", l, r, t.n))
	}
	if l == r {
		if !t.hasIdentity {
			panic(fmt.Sprintf("segtree: Query on empty range [%d, %d) of a Tree without identity (NewMin/NewMax)", l, r))
		}
		return t.identity
	}
	// Accumulate the left and right parts separately so that combine is
	// applied in index order, and track emptiness instead of starting from
	// an identity element.
	var left, right T
	hasLeft, hasRight := false, false
	for l, r = l+t.n, r+t.n; l < r; l, r = l>>1, r>>1 {
		if l&1 == 1 {
			if hasLeft {
				left = t.combine(left, t.nodes[l])
			} else {
				left, hasLeft = t.nodes[l], true
			}
			l++
		}
		if r&1 == 1 {
			r--
			if hasRight {
				right = t.combine(t.nodes[r], right)
			} else {
				right, hasRight = t.nodes[r], true
			}
		}
	}
	switch {
	case !hasLeft:
		return right
	case !hasRight:
		return left
	default:
		return t.combine(left, right)
	}
}

// ToSlice returns a copy of the current elements. O(n).
func (t *Tree[T]) ToSlice() []T {
	out := make([]T, t.n)
	copy(out, t.nodes[t.n:])
	return out
}

// String returns the elements formatted like "Tree[1 2 3]". O(n).
func (t *Tree[T]) String() string {
	return "Tree" + fmt.Sprint(t.nodes[t.n:])
}
