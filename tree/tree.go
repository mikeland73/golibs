// Package tree provides a binary tree node type and helpers for
// LeetCode-style problems: building trees from level-order notation,
// formatting them back, iterative traversals and a readable debug printer.
//
//	root := tree.Parse("[1,2,3,null,4]") // or tree.FromLevelOrder[int](1, 2, 3, nil, 4)
//	fmt.Println(tree.Format(root))        // [1,2,3,null,4]
//	fmt.Println(slices.Collect(tree.InOrder(root)))
//	fmt.Println(root) // draws the tree
//
// Traversals are iterative, so very deep (degenerate) trees cannot overflow
// the call stack, and they stop early when the loop breaks. All functions
// treat a nil *Node as the empty tree.
package tree

import (
	"fmt"
	"iter"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Node is a binary tree node.
type Node[T any] struct {
	Val         T
	Left, Right *Node[T]
}

// FromLevelOrder builds a tree from LeetCode level-order notation, where nil
// marks a missing child:
//
//	tree.FromLevelOrder[int](1, nil, 2, 3) // LeetCode [1,null,2,3]
//
// Every non-nil value must be a T, with one convenience: numeric values are
// converted to a numeric T if the conversion is lossless, so
// FromLevelOrder[float64](1, 2.5) and FromLevelOrder[int64](1, 2) work.
// It panics with a descriptive message on any other value, or if a non-nil
// value has no parent (for example, values after a level of only nils).
// It returns nil for no values or a nil first value. O(n).
func FromLevelOrder[T any](vals ...any) *Node[T] {
	if len(vals) == 0 || vals[0] == nil {
		for i, v := range vals {
			if v != nil {
				panic(fmt.Sprintf("tree: FromLevelOrder value %v at index %d has no parent", v, i))
			}
		}
		return nil
	}
	root := &Node[T]{Val: convert[T](vals[0], 0)}
	queue := []*Node[T]{root}
	i := 1
	for head := 0; head < len(queue) && i < len(vals); head++ {
		n := queue[head]
		if v := vals[i]; v != nil {
			n.Left = &Node[T]{Val: convert[T](v, i)}
			queue = append(queue, n.Left)
		}
		i++
		if i < len(vals) {
			if v := vals[i]; v != nil {
				n.Right = &Node[T]{Val: convert[T](v, i)}
				queue = append(queue, n.Right)
			}
			i++
		}
	}
	for ; i < len(vals); i++ {
		if vals[i] != nil {
			panic(fmt.Sprintf("tree: FromLevelOrder value %v at index %d has no parent", vals[i], i))
		}
	}
	return root
}

// convert returns v as a T, converting numbers losslessly, or panics.
func convert[T any](v any, i int) T {
	if t, ok := v.(T); ok {
		return t
	}
	tt := reflect.TypeFor[T]()
	rv := reflect.ValueOf(v)
	if isNumeric(rv.Kind()) && isNumeric(tt.Kind()) {
		cv := rv.Convert(tt)
		if isNegative(cv) == isNegative(rv) && cv.Convert(rv.Type()).Equal(rv) {
			return cv.Interface().(T)
		}
		panic(fmt.Sprintf("tree: FromLevelOrder[%v] value %v at index %d does not fit in %v", tt, v, i, tt))
	}
	panic(fmt.Sprintf("tree: FromLevelOrder[%v] value %v at index %d has type %T, want %v", tt, v, i, v, tt))
}

func isNumeric(k reflect.Kind) bool {
	return reflect.Int <= k && k <= reflect.Float64 // ints, uints, uintptr, floats
}

func isNegative(v reflect.Value) bool {
	switch {
	case v.CanInt():
		return v.Int() < 0
	case v.CanFloat():
		return v.Float() < 0
	}
	return false
}

// Parse builds an int tree from LeetCode notation such as "[1,2,null,3]".
// Whitespace around brackets, commas and values is ignored, and "[]" yields
// nil. It panics with a descriptive message on malformed input, so it is
// intended for literals in tests and examples. O(len(s)).
func Parse(s string) *Node[int] {
	t := strings.TrimSpace(s)
	if len(t) < 2 || t[0] != '[' || t[len(t)-1] != ']' {
		panic(fmt.Sprintf("tree: Parse(%q): input must be enclosed in [ ]", s))
	}
	inner := strings.TrimSpace(t[1 : len(t)-1])
	if inner == "" {
		return nil
	}
	parts := strings.Split(inner, ",")
	vals := make([]any, len(parts))
	for i, p := range parts {
		p = strings.TrimSpace(p)
		if p == "null" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			panic(fmt.Sprintf("tree: Parse(%q): invalid value %q at index %d", s, p, i))
		}
		vals[i] = n
	}
	return FromLevelOrder[int](vals...)
}

// Format returns the LeetCode level-order notation of the tree, e.g.
// "[1,2,null,3]", with trailing nulls trimmed. Values are formatted with
// fmt.Sprint. A nil tree formats as "[]". O(n).
func Format[T any](root *Node[T]) string {
	var parts []string
	last := 0 // number of parts up to and including the last non-null
	queue := []*Node[T]{root}
	for head := 0; head < len(queue); head++ {
		n := queue[head]
		if n == nil {
			parts = append(parts, "null")
			continue
		}
		parts = append(parts, fmt.Sprint(n.Val))
		last = len(parts)
		queue = append(queue, n.Left, n.Right)
	}
	return "[" + strings.Join(parts[:last], ",") + "]"
}

// PreOrder returns an iterator over the values in preorder (node, left,
// right). O(n) time, O(height) extra space.
func PreOrder[T any](root *Node[T]) iter.Seq[T] {
	return func(yield func(T) bool) {
		if root == nil {
			return
		}
		stack := []*Node[T]{root}
		for len(stack) > 0 {
			n := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if !yield(n.Val) {
				return
			}
			if n.Right != nil {
				stack = append(stack, n.Right)
			}
			if n.Left != nil {
				stack = append(stack, n.Left)
			}
		}
	}
}

// InOrder returns an iterator over the values in inorder (left, node,
// right); for a binary search tree this is sorted order. O(n) time,
// O(height) extra space.
func InOrder[T any](root *Node[T]) iter.Seq[T] {
	return func(yield func(T) bool) {
		var stack []*Node[T]
		cur := root
		for cur != nil || len(stack) > 0 {
			for cur != nil {
				stack = append(stack, cur)
				cur = cur.Left
			}
			n := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if !yield(n.Val) {
				return
			}
			cur = n.Right
		}
	}
}

// PostOrder returns an iterator over the values in postorder (left, right,
// node). O(n) time, O(height) extra space.
func PostOrder[T any](root *Node[T]) iter.Seq[T] {
	return func(yield func(T) bool) {
		var stack []*Node[T]
		var last *Node[T] // most recently yielded node
		cur := root
		for cur != nil || len(stack) > 0 {
			if cur != nil {
				stack = append(stack, cur)
				cur = cur.Left
				continue
			}
			top := stack[len(stack)-1]
			if top.Right != nil && top.Right != last {
				cur = top.Right
				continue
			}
			if !yield(top.Val) {
				return
			}
			last = top
			stack = stack[:len(stack)-1]
		}
	}
}

// LevelOrder returns an iterator over the levels of the tree from the root
// down, yielding the values of each level left to right as a new slice.
// O(n) time, O(width) extra space.
func LevelOrder[T any](root *Node[T]) iter.Seq[[]T] {
	return func(yield func([]T) bool) {
		if root == nil {
			return
		}
		level := []*Node[T]{root}
		for len(level) > 0 {
			vals := make([]T, len(level))
			var next []*Node[T]
			for i, n := range level {
				vals[i] = n.Val
				if n.Left != nil {
					next = append(next, n.Left)
				}
				if n.Right != nil {
					next = append(next, n.Right)
				}
			}
			if !yield(vals) {
				return
			}
			level = next
		}
	}
}

// Height returns the number of nodes on the longest root-to-leaf path
// (LeetCode's "maximum depth"): 0 for a nil tree and 1 for a single node.
// O(n).
func Height[T any](root *Node[T]) int {
	h := 0
	for range LevelOrder(root) {
		h++
	}
	return h
}

// Size returns the number of nodes in the tree. O(n).
func Size[T any](root *Node[T]) int {
	n := 0
	for range PreOrder(root) {
		n++
	}
	return n
}

// String draws the tree top-down using ASCII characters, for debugging:
//
//	 _1
//	/  \
//	2  3
//	 \
//	 4
//
// Values are formatted with fmt.Sprint. The drawing's width grows with the
// total width of all values, so it is best suited to small trees.
// A nil *Node prints as "<nil>".
func (n *Node[T]) String() string {
	if n == nil {
		return "<nil>"
	}
	lines, _, _ := render(n)
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	return strings.Join(lines, "\n")
}

// render draws the subtree rooted at n. All returned lines have the same
// width (in runes) and mid is the column of the middle of n's label.
func render[T any](n *Node[T]) (lines []string, width, mid int) {
	label := fmt.Sprint(n.Val)
	if label == "" {
		label = `""`
	}
	u := utf8.RuneCountInString(label)
	sp := func(k int) string { return strings.Repeat(" ", k) }
	us := func(k int) string { return strings.Repeat("_", k) }

	switch {
	case n.Left == nil && n.Right == nil:
		return []string{label}, u, u / 2

	case n.Right == nil:
		left, lw, lm := render(n.Left)
		out := []string{
			sp(lm+1) + us(lw-lm-1) + label,
			sp(lm) + "/" + sp(lw-lm-1+u),
		}
		for _, l := range left {
			out = append(out, l+sp(u))
		}
		return out, lw + u, lw + u/2

	case n.Left == nil:
		right, rw, rm := render(n.Right)
		out := []string{
			label + us(rm) + sp(rw-rm),
			sp(u+rm) + `\` + sp(rw-rm-1),
		}
		for _, l := range right {
			out = append(out, sp(u)+l)
		}
		return out, rw + u, u / 2

	default:
		left, lw, lm := render(n.Left)
		right, rw, rm := render(n.Right)
		out := []string{
			sp(lm+1) + us(lw-lm-1) + label + us(rm) + sp(rw-rm),
			sp(lm) + "/" + sp(lw-lm-1+u+rm) + `\` + sp(rw-rm-1),
		}
		for i := range max(len(left), len(right)) {
			l, r := sp(lw), sp(rw)
			if i < len(left) {
				l = left[i]
			}
			if i < len(right) {
				r = right[i]
			}
			out = append(out, l+sp(u)+r)
		}
		return out, lw + u + rw, lw + u/2
	}
}
