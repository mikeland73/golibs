// Package deque provides a generic double-ended queue backed by a growable
// ring buffer, similar to Python's collections.deque.
//
// The zero value is an empty deque ready to use:
//
//	var d deque.Deque[int]
//	d.PushBack(2, 3)
//	d.PushFront(1)
//	d.Front()    // 1
//	d.PopBack()  // 3
//	d.At(1)      // 2
//
// Pushing and popping at either end is amortized O(1) and random access via At
// and Set is O(1). The buffer shrinks when it becomes mostly empty.
//
// A deque is the natural structure for a monotonic queue, e.g. the sliding
// window maximum problem: keep indices whose values are decreasing, PopBack
// while the new value is larger, and PopFront when the front index leaves the
// window (see the package examples).
//
// A Deque is not safe for concurrent use.
package deque

import (
	"fmt"
	"iter"
	"strings"
)

const minCap = 8

// Deque is a double-ended queue. The zero value is an empty deque ready to use.
type Deque[T any] struct {
	buf  []T // ring buffer; len(buf) is the capacity (0 or a power of two)
	head int // index of the front element
	n    int // number of elements
}

// New returns a deque containing vals in order (vals[0] is at the front).
func New[T any](vals ...T) *Deque[T] {
	d := &Deque[T]{}
	d.PushBack(vals...)
	return d
}

// PushBack appends vals to the back in order, so the last value ends up at the
// back. Amortized O(1) per value.
func (d *Deque[T]) PushBack(vals ...T) {
	d.grow(len(vals))
	for _, v := range vals {
		d.buf[d.idx(d.n)] = v
		d.n++
	}
}

// PushFront pushes each of vals onto the front in turn, so the LAST value ends
// up at the front: PushFront(1, 2, 3) on an empty deque yields [3 2 1]. This
// matches Python's deque.extendleft. Amortized O(1) per value.
func (d *Deque[T]) PushFront(vals ...T) {
	d.grow(len(vals))
	mask := len(d.buf) - 1
	for _, v := range vals {
		d.head = (d.head - 1) & mask
		d.buf[d.head] = v
		d.n++
	}
}

// PopBack removes and returns the back element. It panics if the deque is
// empty. Amortized O(1).
func (d *Deque[T]) PopBack() T {
	v, ok := d.TryPopBack()
	if !ok {
		panic("deque: PopBack called on empty Deque")
	}
	return v
}

// PopFront removes and returns the front element. It panics if the deque is
// empty. Amortized O(1).
func (d *Deque[T]) PopFront() T {
	v, ok := d.TryPopFront()
	if !ok {
		panic("deque: PopFront called on empty Deque")
	}
	return v
}

// TryPopBack removes and returns the back element and true, or the zero value
// and false if the deque is empty. Amortized O(1).
func (d *Deque[T]) TryPopBack() (T, bool) {
	var zero T
	if d.n == 0 {
		return zero, false
	}
	i := d.idx(d.n - 1)
	v := d.buf[i]
	d.buf[i] = zero // allow GC of popped element
	d.n--
	d.shrink()
	return v, true
}

// TryPopFront removes and returns the front element and true, or the zero
// value and false if the deque is empty. Amortized O(1).
func (d *Deque[T]) TryPopFront() (T, bool) {
	var zero T
	if d.n == 0 {
		return zero, false
	}
	v := d.buf[d.head]
	d.buf[d.head] = zero // allow GC of popped element
	d.head = d.idx(1)
	d.n--
	d.shrink()
	return v, true
}

// Front returns the front element without removing it. It panics if the
// deque is empty. O(1).
func (d *Deque[T]) Front() T {
	if d.n == 0 {
		panic("deque: Front called on empty Deque")
	}
	return d.buf[d.head]
}

// Back returns the back element without removing it. It panics if the deque
// is empty. O(1).
func (d *Deque[T]) Back() T {
	if d.n == 0 {
		panic("deque: Back called on empty Deque")
	}
	return d.buf[d.idx(d.n-1)]
}

// TryFront returns the front element and true, or the zero value and false if
// the deque is empty. O(1).
func (d *Deque[T]) TryFront() (T, bool) {
	if d.n == 0 {
		var zero T
		return zero, false
	}
	return d.buf[d.head], true
}

// TryBack returns the back element and true, or the zero value and false if
// the deque is empty. O(1).
func (d *Deque[T]) TryBack() (T, bool) {
	if d.n == 0 {
		var zero T
		return zero, false
	}
	return d.buf[d.idx(d.n-1)], true
}

// At returns the element at index i, where 0 is the front and Len()-1 is the
// back. It panics if i is out of range. O(1).
func (d *Deque[T]) At(i int) T {
	d.checkIndex("At", i)
	return d.buf[d.idx(i)]
}

// Set replaces the element at index i (0 is the front) with v. It panics if i
// is out of range. O(1).
func (d *Deque[T]) Set(i int, v T) {
	d.checkIndex("Set", i)
	d.buf[d.idx(i)] = v
}

// Rotate rotates the deque n steps to the right, like Python's deque.rotate:
// for positive n the back element moves to the front n times; for negative n
// the front element moves to the back -n times. Rotating an empty deque is a
// no-op. O(min(k, Len()-k)) where k = n mod Len().
func (d *Deque[T]) Rotate(n int) {
	if d.n <= 1 {
		return
	}
	k := n % d.n
	if k < 0 {
		k += d.n
	}
	if k == 0 {
		return
	}
	mask := len(d.buf) - 1
	if d.n == len(d.buf) {
		// Buffer is full: rotating is just moving the head.
		d.head = (d.head - k) & mask
		return
	}
	var zero T
	if k <= d.n-k {
		// Move k elements from back to front.
		for range k {
			back := d.idx(d.n - 1)
			d.head = (d.head - 1) & mask
			d.buf[d.head] = d.buf[back]
			d.buf[back] = zero
		}
		return
	}
	// Move n-k elements from front to back.
	for range d.n - k {
		dst := d.idx(d.n)
		d.buf[dst] = d.buf[d.head]
		d.buf[d.head] = zero
		d.head = (d.head + 1) & mask
	}
}

// Len returns the number of elements. O(1).
func (d *Deque[T]) Len() int { return d.n }

// IsEmpty reports whether the deque has no elements. O(1).
func (d *Deque[T]) IsEmpty() bool { return d.n == 0 }

// Clear removes all elements and releases the underlying buffer. O(1).
func (d *Deque[T]) Clear() {
	*d = Deque[T]{}
}

// All returns an iterator over the index and value of each element from front
// to back. The deque must not be modified during iteration.
func (d *Deque[T]) All() iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		for i := range d.n {
			if !yield(i, d.buf[d.idx(i)]) {
				return
			}
		}
	}
}

// Backward returns an iterator over the index and value of each element from
// back to front. The deque must not be modified during iteration.
func (d *Deque[T]) Backward() iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		for i := d.n - 1; i >= 0; i-- {
			if !yield(i, d.buf[d.idx(i)]) {
				return
			}
		}
	}
}

// ToSlice returns a copy of the elements from front to back. O(n).
func (d *Deque[T]) ToSlice() []T {
	out := make([]T, d.n)
	d.copyTo(out)
	return out
}

// String returns the elements from front to back, e.g. "Deque[1 2 3]". O(n).
func (d *Deque[T]) String() string {
	var b strings.Builder
	b.WriteString("Deque[")
	for i, v := range d.All() {
		if i > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprint(&b, v)
	}
	b.WriteByte(']')
	return b.String()
}

// idx maps logical index i (0 = front) to a buffer index.
func (d *Deque[T]) idx(i int) int {
	return (d.head + i) & (len(d.buf) - 1)
}

func (d *Deque[T]) checkIndex(method string, i int) {
	if i < 0 || i >= d.n {
		panic(fmt.Sprintf("deque: %s index %d out of range [0, %d)", method, i, d.n))
	}
}

// grow ensures there is room for extra more elements.
func (d *Deque[T]) grow(extra int) {
	need := d.n + extra
	if need <= len(d.buf) {
		return
	}
	c := max(len(d.buf), minCap)
	for c < need {
		c *= 2
	}
	d.resize(c)
}

// shrink halves the buffer when it is at most a quarter full.
func (d *Deque[T]) shrink() {
	if c := len(d.buf); c > minCap && d.n <= c/4 {
		d.resize(c / 2)
	}
}

func (d *Deque[T]) copyTo(dst []T) {
	if d.n == 0 {
		return
	}
	end := d.head + d.n
	if end <= len(d.buf) {
		copy(dst, d.buf[d.head:end])
		return
	}
	k := copy(dst, d.buf[d.head:])
	copy(dst[k:], d.buf[:end-len(d.buf)])
}

func (d *Deque[T]) resize(c int) {
	buf := make([]T, c)
	d.copyTo(buf)
	d.buf = buf
	d.head = 0
}
