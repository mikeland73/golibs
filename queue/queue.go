// Package queue provides a generic FIFO queue backed by a growable ring buffer.
//
// The zero value is an empty queue ready to use:
//
//	var q queue.Queue[string]
//	q.Push("a", "b", "c")
//	q.Peek() // "a"
//	q.Pop()  // "a"
//	q.Len()  // 2
//
// Push and Pop are amortized O(1). The buffer shrinks when it becomes mostly
// empty, so long-running push/pop workloads do not retain memory unboundedly.
// For a double-ended queue see the deque package. A Queue is not safe for
// concurrent use.
package queue

import (
	"fmt"
	"iter"
	"strings"
)

const minCap = 8

// Queue is a first-in, first-out collection. The zero value is an empty queue
// ready to use.
type Queue[T any] struct {
	buf  []T // ring buffer; len(buf) is the capacity (0 or a power of two)
	head int // index of the front element
	n    int // number of elements
}

// New returns a queue containing vals, in order (vals[0] is at the front).
func New[T any](vals ...T) *Queue[T] {
	q := &Queue[T]{}
	q.Push(vals...)
	return q
}

// Push appends vals to the back of the queue in order. Amortized O(1) per
// value.
func (q *Queue[T]) Push(vals ...T) {
	if need := q.n + len(vals); need > len(q.buf) {
		c := max(len(q.buf), minCap)
		for c < need {
			c *= 2
		}
		q.resize(c)
	}
	mask := len(q.buf) - 1
	for _, v := range vals {
		q.buf[(q.head+q.n)&mask] = v
		q.n++
	}
}

// Pop removes and returns the front element. It panics if the queue is empty.
// Amortized O(1).
func (q *Queue[T]) Pop() T {
	v, ok := q.TryPop()
	if !ok {
		panic("queue: Pop called on empty Queue")
	}
	return v
}

// TryPop removes and returns the front element and true, or the zero value and
// false if the queue is empty. Amortized O(1).
func (q *Queue[T]) TryPop() (T, bool) {
	var zero T
	if q.n == 0 {
		return zero, false
	}
	v := q.buf[q.head]
	q.buf[q.head] = zero // allow GC of popped element
	q.head = (q.head + 1) & (len(q.buf) - 1)
	q.n--
	if c := len(q.buf); c > minCap && q.n <= c/4 {
		q.resize(c / 2)
	}
	return v, true
}

// Peek returns the front element without removing it. It panics if the queue
// is empty. O(1).
func (q *Queue[T]) Peek() T {
	if q.n == 0 {
		panic("queue: Peek called on empty Queue")
	}
	return q.buf[q.head]
}

// TryPeek returns the front element and true, or the zero value and false if
// the queue is empty. O(1).
func (q *Queue[T]) TryPeek() (T, bool) {
	if q.n == 0 {
		var zero T
		return zero, false
	}
	return q.buf[q.head], true
}

// Len returns the number of elements. O(1).
func (q *Queue[T]) Len() int { return q.n }

// IsEmpty reports whether the queue has no elements. O(1).
func (q *Queue[T]) IsEmpty() bool { return q.n == 0 }

// Clear removes all elements and releases the underlying buffer. O(1).
func (q *Queue[T]) Clear() {
	*q = Queue[T]{}
}

// All returns an iterator over the elements from front to back.
// The queue must not be modified during iteration.
func (q *Queue[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		mask := len(q.buf) - 1
		for i := range q.n {
			if !yield(q.buf[(q.head+i)&mask]) {
				return
			}
		}
	}
}

// ToSlice returns a copy of the elements from front to back. O(n).
func (q *Queue[T]) ToSlice() []T {
	out := make([]T, q.n)
	q.copyTo(out)
	return out
}

// String returns the elements from front to back, e.g. "Queue[1 2 3]" where 1
// is the front. O(n).
func (q *Queue[T]) String() string {
	var b strings.Builder
	b.WriteString("Queue[")
	i := 0
	for v := range q.All() {
		if i > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprint(&b, v)
		i++
	}
	b.WriteByte(']')
	return b.String()
}

// copyTo copies the elements in order into dst, which must have len >= q.n.
func (q *Queue[T]) copyTo(dst []T) {
	if q.n == 0 {
		return
	}
	end := q.head + q.n
	if end <= len(q.buf) {
		copy(dst, q.buf[q.head:end])
		return
	}
	k := copy(dst, q.buf[q.head:])
	copy(dst[k:], q.buf[:end-len(q.buf)])
}

// resize moves the elements into a new buffer of capacity c (a power of
// two >= q.n).
func (q *Queue[T]) resize(c int) {
	buf := make([]T, c)
	q.copyTo(buf)
	q.buf = buf
	q.head = 0
}
