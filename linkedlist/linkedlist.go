// Package linkedlist provides a singly linked list node type and helpers for
// LeetCode-style problems: building lists (with or without a cycle),
// converting them back to slices and printing them safely.
//
//	head := linkedlist.New(1, 2, 3)
//	fmt.Println(head)                     // 1 -> 2 -> 3
//	fmt.Println(linkedlist.ToSlice(head)) // [1 2 3]
//
//	cyc := linkedlist.WithCycle(1, 3, 2, 0, -4) // tail links back to index 1
//	fmt.Println(cyc)                            // 3 -> 2 -> 0 -> -4 -> ... (cycle to index 1)
//
// A nil *Node is the empty list. For a doubly linked list container use
// container/list.
package linkedlist

import (
	"fmt"
	"iter"
	"strings"
)

// Node is a singly linked list node.
type Node[T any] struct {
	Val  T
	Next *Node[T]
}

// New returns the head of a new list holding vals in order, or nil if vals
// is empty. O(n).
func New[T any](vals ...T) *Node[T] {
	return WithCycle(-1, vals...)
}

// WithCycle is like New but links the tail back to the node at index pos,
// creating a cycle, as in LeetCode's "linked list cycle" problems. pos = -1
// means no cycle. It panics if pos is not in [-1, len(vals)). O(n).
func WithCycle[T any](pos int, vals ...T) *Node[T] {
	if pos < -1 || pos >= len(vals) {
		panic(fmt.Sprintf("linkedlist: WithCycle pos %d out of range [-1, %d)", pos, len(vals)))
	}
	if len(vals) == 0 {
		return nil
	}
	nodes := make([]Node[T], len(vals)) // one allocation for the whole list
	for i, v := range vals {
		nodes[i].Val = v
		if i > 0 {
			nodes[i-1].Next = &nodes[i]
		}
	}
	if pos >= 0 {
		nodes[len(nodes)-1].Next = &nodes[pos]
	}
	return &nodes[0]
}

// ToSlice returns the values of the list in order. O(n).
// It panics if the list contains a cycle instead of looping forever.
func ToSlice[T any](head *Node[T]) []T {
	var out []T
	for n := range nodes(head, "ToSlice") {
		out = append(out, n.Val)
	}
	return out
}

// Len returns the number of nodes in the list. O(n).
// It panics if the list contains a cycle instead of looping forever.
func Len[T any](head *Node[T]) int {
	n := 0
	for range nodes(head, "Len") {
		n++
	}
	return n
}

// All returns an iterator over the values of the list in order. On a list
// with a cycle it never ends unless the loop breaks.
func All[T any](head *Node[T]) iter.Seq[T] {
	return func(yield func(T) bool) {
		for n := head; n != nil; n = n.Next {
			if !yield(n.Val) {
				return
			}
		}
	}
}

// NodeAt returns the node at index i (0-based), or nil if i is negative or
// at least the length of the list. It follows cycles, so it always
// terminates. O(i).
func NodeAt[T any](head *Node[T], i int) *Node[T] {
	if i < 0 {
		return nil
	}
	n := head
	for ; n != nil && i > 0; i-- {
		n = n.Next
	}
	return n
}

// String formats the list starting at n as "1 -> 2 -> 3". It is safe on
// lists with a cycle: when the walk reaches an already printed node it ends
// with "-> ... (cycle to index k)", where k is that node's index.
// A nil *Node prints as "nil". O(n) time and space.
func (n *Node[T]) String() string {
	if n == nil {
		return "nil"
	}
	var sb strings.Builder
	index := make(map[*Node[T]]int)
	for i, cur := 0, n; cur != nil; i, cur = i+1, cur.Next {
		if k, ok := index[cur]; ok {
			fmt.Fprintf(&sb, " -> ... (cycle to index %d)", k)
			break
		}
		index[cur] = i
		if i > 0 {
			sb.WriteString(" -> ")
		}
		fmt.Fprint(&sb, cur.Val)
	}
	return sb.String()
}

// nodes iterates over the nodes of the list and panics if it detects a
// cycle, using Floyd's tortoise and hare: O(1) extra space.
func nodes[T any](head *Node[T], fn string) iter.Seq[*Node[T]] {
	return func(yield func(*Node[T]) bool) {
		slow := head
		for fast, i := head, 0; fast != nil; fast, i = fast.Next, i+1 {
			if !yield(fast) {
				return
			}
			if i%2 == 1 {
				slow = slow.Next
				if slow == fast.Next {
					panic("linkedlist: " + fn + " called on a list with a cycle")
				}
			}
		}
	}
}
