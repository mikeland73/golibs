// Package stack provides a generic LIFO stack backed by a slice.
//
// The zero value is an empty stack ready to use:
//
//	var s stack.Stack[int]
//	s.Push(1, 2, 3)
//	s.Peek() // 3
//	s.Pop()  // 3
//	s.Len()  // 2
//
// Push, Pop and Peek are amortized O(1). A Stack is not safe for concurrent use.
package stack

import (
	"fmt"
	"iter"
	"slices"
	"strings"
)

// Stack is a last-in, first-out collection. The zero value is an empty stack
// ready to use.
type Stack[T any] struct {
	data []T // bottom at index 0, top at the end
}

// New returns a stack containing vals, pushed in order (the last value is on top).
func New[T any](vals ...T) *Stack[T] {
	return &Stack[T]{data: slices.Clone(vals)}
}

// Push pushes vals onto the stack in order, so the last value ends up on top.
// Amortized O(1) per value.
func (s *Stack[T]) Push(vals ...T) {
	s.data = append(s.data, vals...)
}

// Pop removes and returns the top element. It panics if the stack is empty. O(1).
func (s *Stack[T]) Pop() T {
	v, ok := s.TryPop()
	if !ok {
		panic("stack: Pop called on empty Stack")
	}
	return v
}

// TryPop removes and returns the top element and true, or the zero value and
// false if the stack is empty. O(1).
func (s *Stack[T]) TryPop() (T, bool) {
	var zero T
	n := len(s.data)
	if n == 0 {
		return zero, false
	}
	v := s.data[n-1]
	s.data[n-1] = zero // allow GC of popped element
	s.data = s.data[:n-1]
	return v, true
}

// Peek returns the top element without removing it. It panics if the stack
// is empty. O(1).
func (s *Stack[T]) Peek() T {
	v, ok := s.TryPeek()
	if !ok {
		panic("stack: Peek called on empty Stack")
	}
	return v
}

// TryPeek returns the top element and true, or the zero value and false if
// the stack is empty. O(1).
func (s *Stack[T]) TryPeek() (T, bool) {
	if len(s.data) == 0 {
		var zero T
		return zero, false
	}
	return s.data[len(s.data)-1], true
}

// Len returns the number of elements. O(1).
func (s *Stack[T]) Len() int { return len(s.data) }

// IsEmpty reports whether the stack has no elements. O(1).
func (s *Stack[T]) IsEmpty() bool { return len(s.data) == 0 }

// Clear removes all elements, keeping the allocated capacity. O(n).
func (s *Stack[T]) Clear() {
	clear(s.data)
	s.data = s.data[:0]
}

// All returns an iterator over the elements from top to bottom (pop order).
// The stack must not be modified during iteration.
func (s *Stack[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		for i := len(s.data) - 1; i >= 0; i-- {
			if !yield(s.data[i]) {
				return
			}
		}
	}
}

// ToSlice returns a copy of the elements from bottom to top (push order), so
// the last element is the top. O(n).
func (s *Stack[T]) ToSlice() []T {
	return slices.Clone(s.data)
}

// String returns the elements from bottom to top, e.g. "Stack[1 2 3]" where 3
// is the top. O(n).
func (s *Stack[T]) String() string {
	var b strings.Builder
	b.WriteString("Stack[")
	for i, v := range s.data {
		if i > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprint(&b, v)
	}
	b.WriteByte(']')
	return b.String()
}
