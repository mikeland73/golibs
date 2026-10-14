// Package unionfind provides a generic disjoint-set union (union-find) data
// structure with path compression and union by size.
//
//	uf := unionfind.New("a", "b", "c", "d")
//	uf.Union("a", "b")       // true: merged
//	uf.Union("b", "a")       // false: already connected
//	uf.Connected("a", "b")   // true
//	uf.Size("a")             // 2
//	uf.Count()               // 3 components: {a b} {c} {d}
//	uf.Groups()              // [[a b] [c] [d]]
//
// Elements can be any comparable type (ints, strings, grid points, ...).
// Add and Union register unknown elements automatically, so explicit Add calls
// are only needed for elements that may never be unioned but should still count
// as components. Query methods (Find, Connected, Size) never add elements: an
// unknown element behaves like a singleton set.
//
// With both optimizations every operation runs in amortized O(α(n)) time,
// where α is the inverse Ackermann function (at most 4 for any practical n),
// i.e. effectively constant.
//
// A UnionFind is not safe for concurrent use; note that even Find, Connected
// and Size mutate internal state (path compression).
package unionfind

import (
	"fmt"
	"strings"
)

// UnionFind is a disjoint-set forest over elements of type T.
//
// The zero value is an empty UnionFind ready to use.
type UnionFind[T comparable] struct {
	ids    map[T]int // element -> dense id
	elems  []T       // id -> element, in insertion order
	parent []int     // id -> parent id
	size   []int     // root id -> component size
	count  int       // number of components
}

// New returns a UnionFind in which each of vals is in its own set.
// O(k) for k values.
func New[T comparable](vals ...T) *UnionFind[T] {
	u := &UnionFind[T]{}
	u.Add(vals...)
	return u
}

// Add registers each value that is not yet known as a new singleton set.
// Values already present are left unchanged. Amortized O(1) per value.
func (u *UnionFind[T]) Add(vals ...T) {
	for _, v := range vals {
		u.id(v)
	}
}

// id returns the dense id of x, adding x as a singleton if it is unknown.
func (u *UnionFind[T]) id(x T) int {
	if i, ok := u.ids[x]; ok {
		return i
	}
	if u.ids == nil {
		u.ids = make(map[T]int)
	}
	i := len(u.elems)
	u.ids[x] = i
	u.elems = append(u.elems, x)
	u.parent = append(u.parent, i)
	u.size = append(u.size, 1)
	u.count++
	return i
}

// root returns the root id of the set containing id i, compressing the path.
func (u *UnionFind[T]) root(i int) int {
	r := i
	for u.parent[r] != r {
		r = u.parent[r]
	}
	for u.parent[i] != r {
		u.parent[i], i = r, u.parent[i]
	}
	return r
}

// Contains reports whether x has been added (directly or via Union). O(1).
func (u *UnionFind[T]) Contains(x T) bool {
	_, ok := u.ids[x]
	return ok
}

// Find returns the representative element of the set containing x. Two
// elements are in the same set exactly when they have the same
// representative. If x is unknown it is returned unchanged and is not added.
// Amortized O(α(n)).
func (u *UnionFind[T]) Find(x T) T {
	i, ok := u.ids[x]
	if !ok {
		return x
	}
	return u.elems[u.root(i)]
}

// Union merges the sets containing a and b, adding either element if it is
// unknown. It reports whether a merge happened, i.e. false if a and b were
// already in the same set. Amortized O(α(n)).
func (u *UnionFind[T]) Union(a, b T) bool {
	ra, rb := u.root(u.id(a)), u.root(u.id(b))
	if ra == rb {
		return false
	}
	if u.size[ra] < u.size[rb] {
		ra, rb = rb, ra
	}
	u.parent[rb] = ra
	u.size[ra] += u.size[rb]
	u.count--
	return true
}

// Connected reports whether a and b are in the same set. An unknown element
// is only connected to itself. Amortized O(α(n)).
func (u *UnionFind[T]) Connected(a, b T) bool {
	ia, okA := u.ids[a]
	ib, okB := u.ids[b]
	if !okA || !okB {
		return a == b
	}
	return u.root(ia) == u.root(ib)
}

// Size returns the number of elements in the set containing x, or 1 if x is
// unknown. Amortized O(α(n)).
func (u *UnionFind[T]) Size(x T) int {
	i, ok := u.ids[x]
	if !ok {
		return 1
	}
	return u.size[u.root(i)]
}

// Count returns the number of disjoint sets among the known elements. O(1).
func (u *UnionFind[T]) Count() int {
	return u.count
}

// Len returns the number of known elements. O(1).
func (u *UnionFind[T]) Len() int {
	return len(u.elems)
}

// Groups returns the elements partitioned into their sets. The result is
// deterministic: groups are ordered by the insertion order of their earliest
// added element, and elements within a group are in insertion order.
// O(n α(n)).
func (u *UnionFind[T]) Groups() [][]T {
	groups := make([][]T, 0, u.count)
	slot := make(map[int]int, u.count) // root id -> index into groups
	for i, x := range u.elems {
		r := u.root(i)
		g, ok := slot[r]
		if !ok {
			g = len(groups)
			slot[r] = g
			groups = append(groups, make([]T, 0, u.size[r]))
		}
		groups[g] = append(groups[g], x)
	}
	return groups
}

// String returns the sets formatted like "UnionFind[[a b] [c]]", using the
// order of Groups. O(n α(n)).
func (u *UnionFind[T]) String() string {
	var sb strings.Builder
	sb.WriteString("UnionFind[")
	for i, g := range u.Groups() {
		if i > 0 {
			sb.WriteByte(' ')
		}
		fmt.Fprint(&sb, g)
	}
	sb.WriteByte(']')
	return sb.String()
}
