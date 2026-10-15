// Package pq provides a generic binary-heap priority queue with Peek, plus
// heapq-style helpers for selecting the k largest or smallest elements.
//
// Unlike container/heap, there is no interface to implement and no boxing:
//
//	var q pq.PriorityQueue[int] // zero value: min-queue for ordered types
//	q.Push(5, 1, 3)
//	q.Peek() // 1
//	q.Pop()  // 1
//	q.Pop()  // 3
//
// For a custom order pass a comparator with the same shape as
// slices.SortFunc / cmp.Compare. The element that compares smallest is popped
// first:
//
//	type task struct{ name string; prio int }
//	q := pq.New(func(a, b task) int { return cmp.Compare(b.prio, a.prio) }) // highest prio first
//
// NewMax gives a max-queue for ordered types.
//
// A PriorityQueue is not safe for concurrent use.
package pq

import (
	"cmp"
	"fmt"
	"iter"
	"reflect"
	"slices"
	"strings"
)

// PriorityQueue is a binary min-heap ordered by a comparator. The element
// that compares smallest according to the comparator is at the top.
//
// The zero value is an empty min-queue when T is an ordered type (an integer,
// float or string type, including named types such as `type dist int`),
// ordered like cmp.Compare. For other element types, or for a different
// order, create queues with New, NewMax or From; pushing onto a zero-value
// queue of a non-ordered type panics.
type PriorityQueue[T any] struct {
	data []T
	cmp  func(a, b T) int
}

// New returns an empty priority queue ordered by cmp. The element for which
// cmp reports the smallest value is popped first. cmp must not be nil.
func New[T any](cmp func(a, b T) int) *PriorityQueue[T] {
	if cmp == nil {
		panic("pq: New called with nil comparator")
	}
	return &PriorityQueue[T]{cmp: cmp}
}

// NewMin returns an empty min-priority queue: Pop returns the smallest element.
func NewMin[T cmp.Ordered]() *PriorityQueue[T] {
	return New(cmp.Compare[T])
}

// NewMax returns an empty max-priority queue: Pop returns the largest element.
func NewMax[T cmp.Ordered]() *PriorityQueue[T] {
	return New(func(a, b T) int { return cmp.Compare(b, a) })
}

// From returns a priority queue ordered by cmp containing a copy of items.
// It heapifies in O(n), which is faster than pushing items one at a time.
// The items slice is not modified or retained.
func From[T any](items []T, cmp func(a, b T) int) *PriorityQueue[T] {
	if cmp == nil {
		panic("pq: From called with nil comparator")
	}
	q := New(cmp)
	q.data = slices.Clone(items)
	for i := len(q.data)/2 - 1; i >= 0; i-- {
		q.down(i)
	}
	return q
}

// Push adds vals to the queue. O(log n) per value.
func (q *PriorityQueue[T]) Push(vals ...T) {
	q.order() // a zero-value queue of a non-ordered type panics here
	for _, v := range vals {
		q.data = append(q.data, v)
		q.up(len(q.data) - 1)
	}
}

// Pop removes and returns the top (smallest by cmp) element.
// It panics if the queue is empty. O(log n).
func (q *PriorityQueue[T]) Pop() T {
	v, ok := q.TryPop()
	if !ok {
		panic("pq: Pop called on empty PriorityQueue")
	}
	return v
}

// TryPop removes and returns the top element and true, or the zero value and
// false if the queue is empty. O(log n).
func (q *PriorityQueue[T]) TryPop() (T, bool) {
	n := len(q.data)
	if n == 0 {
		var zero T
		return zero, false
	}
	top := q.data[0]
	q.data[0] = q.data[n-1]
	var zero T
	q.data[n-1] = zero // allow GC of popped element
	q.data = q.data[:n-1]
	if n > 1 {
		q.down(0)
	}
	return top, true
}

// Peek returns the top (smallest by cmp) element without removing it.
// It panics if the queue is empty. O(1).
func (q *PriorityQueue[T]) Peek() T {
	if len(q.data) == 0 {
		panic("pq: Peek called on empty PriorityQueue")
	}
	return q.data[0]
}

// TryPeek returns the top element and true, or the zero value and false if
// the queue is empty. O(1).
func (q *PriorityQueue[T]) TryPeek() (T, bool) {
	if len(q.data) == 0 {
		var zero T
		return zero, false
	}
	return q.data[0], true
}

// PushPop pushes x and then pops and returns the top element, like Python's
// heapq.heappushpop. It is more efficient than Push followed by Pop: if x
// would be the new top it is returned immediately without touching the heap.
// Works on an empty queue (returns x). O(log n).
func (q *PriorityQueue[T]) PushPop(x T) T {
	if len(q.data) == 0 || q.order()(x, q.data[0]) <= 0 {
		return x
	}
	top := q.data[0]
	q.data[0] = x
	q.down(0)
	return top
}

// Replace pops and returns the top element and then pushes x, like Python's
// heapq.heapreplace. The returned value may be "larger" than x. It is more
// efficient than Pop followed by Push. It panics if the queue is empty.
// O(log n).
func (q *PriorityQueue[T]) Replace(x T) T {
	if len(q.data) == 0 {
		panic("pq: Replace called on empty PriorityQueue")
	}
	top := q.data[0]
	q.data[0] = x
	q.down(0)
	return top
}

// Len returns the number of elements in the queue. O(1).
func (q *PriorityQueue[T]) Len() int { return len(q.data) }

// IsEmpty reports whether the queue has no elements. O(1).
func (q *PriorityQueue[T]) IsEmpty() bool { return len(q.data) == 0 }

// Clear removes all elements, keeping the allocated capacity. O(n).
func (q *PriorityQueue[T]) Clear() {
	clear(q.data)
	q.data = q.data[:0]
}

// All returns an iterator over the elements in internal heap order, which is
// NOT sorted (only the first element is guaranteed to be the top). To consume
// in priority order, Pop repeatedly. The queue must not be modified during
// iteration.
func (q *PriorityQueue[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, v := range q.data {
			if !yield(v) {
				return
			}
		}
	}
}

// ToSlice returns a copy of the elements in internal heap order (NOT sorted).
// O(n).
func (q *PriorityQueue[T]) ToSlice() []T {
	return slices.Clone(q.data)
}

// String returns the elements in sorted (priority) order, e.g. "PriorityQueue[1
// 2 3]". It does not modify the queue. O(n log n).
func (q *PriorityQueue[T]) String() string {
	s := slices.Clone(q.data)
	if len(s) > 1 {
		slices.SortStableFunc(s, q.order())
	}
	var b strings.Builder
	b.WriteString("PriorityQueue[")
	for i, v := range s {
		if i > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprint(&b, v)
	}
	b.WriteByte(']')
	return b.String()
}

// order returns the queue's comparator, defaulting a zero-value queue to
// ascending order for ordered element types.
func (q *PriorityQueue[T]) order() func(a, b T) int {
	if q.cmp == nil {
		q.cmp = defaultCompare[T]()
	}
	return q.cmp
}

func (q *PriorityQueue[T]) up(i int) {
	cmp := q.order()
	d := q.data
	x := d[i]
	for i > 0 {
		p := (i - 1) / 2
		if cmp(x, d[p]) >= 0 {
			break
		}
		d[i] = d[p]
		i = p
	}
	d[i] = x
}

func (q *PriorityQueue[T]) down(i int) {
	cmp := q.order()
	d := q.data
	n := len(d)
	x := d[i]
	for {
		c := 2*i + 1
		if c >= n {
			break
		}
		if r := c + 1; r < n && cmp(d[r], d[c]) < 0 {
			c = r
		}
		if cmp(x, d[c]) <= 0 {
			break
		}
		d[i] = d[c]
		i = c
	}
	d[i] = x
}

// NLargest returns the k largest elements of s in descending order, like
// Python's heapq.nlargest. If k <= 0 it returns an empty slice; if k >= len(s)
// it returns all elements sorted descending. s is not modified.
// O(n log k).
func NLargest[T cmp.Ordered](s []T, k int) []T {
	return NLargestFunc(s, k, cmp.Compare[T])
}

// NSmallest returns the k smallest elements of s in ascending order, like
// Python's heapq.nsmallest. If k <= 0 it returns an empty slice; if k >= len(s)
// it returns all elements sorted ascending. s is not modified.
// O(n log k).
func NSmallest[T cmp.Ordered](s []T, k int) []T {
	return NSmallestFunc(s, k, cmp.Compare[T])
}

// NLargestFunc is like NLargest but orders elements by cmp. The result is
// sorted from largest to smallest according to cmp; equal elements keep
// their input order, as with heapq.nlargest. O(n log k).
func NLargestFunc[T any](s []T, k int, cmp func(a, b T) int) []T {
	if cmp == nil {
		panic("pq: NLargestFunc called with nil comparator")
	}
	return best(s, k, func(a, b indexed[T]) int {
		if c := cmp(b.v, a.v); c != 0 {
			return c
		}
		return a.i - b.i
	})
}

// NSmallestFunc is like NSmallest but orders elements by cmp. The result is
// sorted from smallest to largest according to cmp; equal elements keep
// their input order, as with heapq.nsmallest. O(n log k).
func NSmallestFunc[T any](s []T, k int, cmp func(a, b T) int) []T {
	if cmp == nil {
		panic("pq: NSmallestFunc called with nil comparator")
	}
	return best(s, k, func(a, b indexed[T]) int {
		if c := cmp(a.v, b.v); c != 0 {
			return c
		}
		return a.i - b.i
	})
}

// indexed pairs a value with its input position so ties can be broken by
// position.
type indexed[T any] struct {
	v T
	i int
}

// best returns the k elements of s that come first under order, sorted by
// order. order must be a strict total order (it breaks ties by index). It
// keeps a heap of size k whose top is the worst element kept so far.
func best[T any](s []T, k int, order func(a, b indexed[T]) int) []T {
	if k <= 0 {
		return []T{}
	}
	k = min(k, len(s))
	worstFirst := func(a, b indexed[T]) int { return order(b, a) }
	init := make([]indexed[T], k)
	for i := range k {
		init[i] = indexed[T]{s[i], i}
	}
	h := From(init, worstFirst)
	for i := k; i < len(s); i++ {
		if x := (indexed[T]{s[i], i}); order(x, h.data[0]) < 0 {
			h.Replace(x)
		}
	}
	slices.SortFunc(h.data, order)
	res := make([]T, len(h.data))
	for i, x := range h.data {
		res[i] = x.v
	}
	return res
}

// defaultCompare returns a cmp.Compare-style comparator for T if T's
// underlying type is an integer, float or string type, and panics otherwise.
func defaultCompare[T any]() func(a, b T) int {
	// Fast path: T is one of the predeclared ordered types.
	var c any
	switch any(*new(T)).(type) {
	case int:
		c = cmp.Compare[int]
	case int8:
		c = cmp.Compare[int8]
	case int16:
		c = cmp.Compare[int16]
	case int32:
		c = cmp.Compare[int32]
	case int64:
		c = cmp.Compare[int64]
	case uint:
		c = cmp.Compare[uint]
	case uint8:
		c = cmp.Compare[uint8]
	case uint16:
		c = cmp.Compare[uint16]
	case uint32:
		c = cmp.Compare[uint32]
	case uint64:
		c = cmp.Compare[uint64]
	case uintptr:
		c = cmp.Compare[uintptr]
	case float32:
		c = cmp.Compare[float32]
	case float64:
		c = cmp.Compare[float64]
	case string:
		c = cmp.Compare[string]
	}
	if c != nil {
		return c.(func(a, b T) int)
	}
	// Named types (e.g. `type dist int`) go through reflection.
	t := reflect.TypeFor[T]()
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return func(a, b T) int {
			return cmp.Compare(reflect.ValueOf(a).Int(), reflect.ValueOf(b).Int())
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return func(a, b T) int {
			return cmp.Compare(reflect.ValueOf(a).Uint(), reflect.ValueOf(b).Uint())
		}
	case reflect.Float32, reflect.Float64:
		return func(a, b T) int {
			return cmp.Compare(reflect.ValueOf(a).Float(), reflect.ValueOf(b).Float())
		}
	case reflect.String:
		return func(a, b T) int {
			return cmp.Compare(reflect.ValueOf(a).String(), reflect.ValueOf(b).String())
		}
	}
	panic(fmt.Sprintf("pq: zero-value PriorityQueue[%v] needs an ordered element type; create it with pq.New and a comparator", t))
}
