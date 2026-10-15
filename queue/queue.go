// Package queue provides a generic FIFO queue.
//
// The zero value is an empty queue ready to use:
//
//	var q queue.Queue[string]
//	q.Push("a", "b", "c")
//	q.Peek() // "a"
//	q.Pop()  // "a"
//	q.Len()  // 2
//
// A Queue is a FIFO view of a deque.Deque (a growable ring buffer that
// shrinks when mostly empty): Push and Pop are amortized O(1), and long-running
// push/pop workloads do not retain memory unboundedly. Use the deque package
// directly for access to both ends. A Queue is not safe for concurrent use.
package queue

import (
	"iter"
	"strings"

	"github.com/mikeland73/golibs/deque"
)

// Queue is a first-in, first-out collection. The zero value is an empty queue
// ready to use.
type Queue[T any] struct {
	d deque.Deque[T]
}

// New returns a queue containing vals, in order (vals[0] is at the front).
func New[T any](vals ...T) *Queue[T] {
	q := &Queue[T]{}
	q.Push(vals...)
	return q
}

// Push appends vals to the back of the queue in order. Amortized O(1) per
// value.
func (q *Queue[T]) Push(vals ...T) { q.d.PushBack(vals...) }

// Pop removes and returns the front element. It panics if the queue is empty.
// Amortized O(1).
func (q *Queue[T]) Pop() T {
	v, ok := q.d.TryPopFront()
	if !ok {
		panic("queue: Pop called on empty Queue")
	}
	return v
}

// TryPop removes and returns the front element and true, or the zero value
// and false if the queue is empty. Amortized O(1).
func (q *Queue[T]) TryPop() (T, bool) { return q.d.TryPopFront() }

// Peek returns the front element without removing it. It panics if the queue
// is empty. O(1).
func (q *Queue[T]) Peek() T {
	v, ok := q.d.TryFront()
	if !ok {
		panic("queue: Peek called on empty Queue")
	}
	return v
}

// TryPeek returns the front element and true, or the zero value and false if
// the queue is empty. O(1).
func (q *Queue[T]) TryPeek() (T, bool) { return q.d.TryFront() }

// Len returns the number of elements in the queue. O(1).
func (q *Queue[T]) Len() int { return q.d.Len() }

// IsEmpty reports whether the queue has no elements. O(1).
func (q *Queue[T]) IsEmpty() bool { return q.d.IsEmpty() }

// Clear removes all elements and releases the underlying buffer. O(1).
func (q *Queue[T]) Clear() { q.d.Clear() }

// All returns an iterator over the elements from front to back. The queue
// must not be modified during iteration.
func (q *Queue[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, v := range q.d.All() {
			if !yield(v) {
				return
			}
		}
	}
}

// ToSlice returns the elements from front to back in a new slice. O(n).
func (q *Queue[T]) ToSlice() []T { return q.d.ToSlice() }

// String formats the queue front to back, e.g. "Queue[a b c]".
func (q *Queue[T]) String() string {
	return "Queue" + strings.TrimPrefix(q.d.String(), "Deque")
}
