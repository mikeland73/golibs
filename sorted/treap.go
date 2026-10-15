package sorted

// This file contains the single balanced tree shared by Map, Set and List: a
// treap (randomized binary search tree) whose nodes are augmented with subtree
// sizes for O(log n) order statistics (Rank, At, Bisect*).
//
// Priorities come from a per-tree splitmix64 generator with a fixed seed, so
// the shape of a tree depends only on the sequence of operations performed on
// it. Behavior is fully deterministic and reproducible.
//
// All operations below run in expected O(log n) time; recursion depth is the
// tree height, which is O(log n) with high probability.

// node is a treap node. size is the number of nodes in the subtree rooted here.
type node[K, V any] struct {
	key         K
	val         V
	left, right *node[K, V]
	prio        uint64
	size        int
}

// sz returns the subtree size of n, or 0 for a nil node.
func (n *node[K, V]) sz() int {
	if n == nil {
		return 0
	}
	return n.size
}

func (n *node[K, V]) update() {
	n.size = 1 + n.left.sz() + n.right.sz()
}

// treap is an ordered collection of key/value nodes. If multi is false, keys
// are unique; otherwise equal keys may repeat and are kept in insertion order.
type treap[K, V any] struct {
	root  *node[K, V]
	cmp   func(a, b K) int
	rng   uint64
	multi bool
}

const treapSeed = 0x9E3779B97F4A7C15

func newTreap[K, V any](cmp func(a, b K) int, multi bool) treap[K, V] {
	return treap[K, V]{cmp: cmp, rng: treapSeed, multi: multi}
}

// nextPrio returns the next value of the tree's splitmix64 sequence.
func (t *treap[K, V]) nextPrio() uint64 {
	t.rng += 0x9E3779B97F4A7C15
	z := t.rng
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	return z ^ (z >> 31)
}

func (t *treap[K, V]) len() int { return t.root.sz() }

func (t *treap[K, V]) clear() {
	t.root = nil
	t.rng = treapSeed
}

// merge joins a and b, where every key in a orders before every key in b.
func merge[K, V any](a, b *node[K, V]) *node[K, V] {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	if a.prio > b.prio {
		a.right = merge(a.right, b)
		a.update()
		return a
	}
	b.left = merge(a, b.left)
	b.update()
	return b
}

// split splits n into keys < k and keys >= k, or, if inclusive is true, into
// keys <= k and keys > k.
func (t *treap[K, V]) split(n *node[K, V], k K, inclusive bool) (l, r *node[K, V]) {
	if n == nil {
		return nil, nil
	}
	c := t.cmp(n.key, k)
	if c < 0 || (inclusive && c == 0) {
		n.right, r = t.split(n.right, k, inclusive)
		n.update()
		return n, r
	}
	l, n.left = t.split(n.left, k, inclusive)
	n.update()
	return l, n
}

// find returns the node with key k (the first one if keys repeat), or nil.
func (t *treap[K, V]) find(k K) *node[K, V] {
	var found *node[K, V]
	for n := t.root; n != nil; {
		c := t.cmp(k, n.key)
		switch {
		case c < 0:
			n = n.left
		case c > 0:
			n = n.right
		default:
			if !t.multi {
				return n
			}
			found = n
			n = n.left
		}
	}
	return found
}

// put inserts k with value v. For unique trees an existing key has its value
// replaced and put reports false. For multi trees the new node is placed after
// all existing equal keys and put always reports true.
func (t *treap[K, V]) put(k K, v V) bool {
	if !t.multi {
		if n := t.find(k); n != nil {
			n.val = v
			return false
		}
	}
	nn := &node[K, V]{key: k, val: v, prio: t.nextPrio(), size: 1}
	l, r := t.split(t.root, k, true)
	t.root = merge(merge(l, nn), r)
	return true
}

// remove deletes one node with key k (the first one if keys repeat) and
// returns it, or nil if k is absent.
func (t *treap[K, V]) remove(k K) *node[K, V] {
	var removed *node[K, V]
	t.root, removed = t.removeFrom(t.root, k)
	return removed
}

func (t *treap[K, V]) removeFrom(n *node[K, V], k K) (*node[K, V], *node[K, V]) {
	if n == nil {
		return nil, nil
	}
	var removed *node[K, V]
	c := t.cmp(k, n.key)
	switch {
	case c < 0:
		n.left, removed = t.removeFrom(n.left, k)
	case c > 0:
		n.right, removed = t.removeFrom(n.right, k)
	default:
		if t.multi {
			// Prefer an earlier equal key in the left subtree.
			n.left, removed = t.removeFrom(n.left, k)
		}
		if removed == nil {
			return merge(n.left, n.right), n
		}
	}
	if removed != nil {
		n.size--
	}
	return n, removed
}

// removeAll deletes every node with key k and returns how many were removed.
func (t *treap[K, V]) removeAll(k K) int {
	l, rest := t.split(t.root, k, false)
	mid, r := t.split(rest, k, true)
	t.root = merge(l, r)
	return mid.sz()
}

// removeAt deletes and returns the i-th smallest node. i must be in range.
func (t *treap[K, V]) removeAt(i int) *node[K, V] {
	var removed *node[K, V]
	t.root, removed = removeAtFrom(t.root, i)
	return removed
}

func removeAtFrom[K, V any](n *node[K, V], i int) (*node[K, V], *node[K, V]) {
	var removed *node[K, V]
	ls := n.left.sz()
	switch {
	case i < ls:
		n.left, removed = removeAtFrom(n.left, i)
	case i > ls:
		n.right, removed = removeAtFrom(n.right, i-ls-1)
	default:
		return merge(n.left, n.right), n
	}
	n.size--
	return n, removed
}

// at returns the i-th smallest node (0-based). i must be in range.
func (t *treap[K, V]) at(i int) *node[K, V] {
	n := t.root
	for {
		ls := n.left.sz()
		switch {
		case i < ls:
			n = n.left
		case i > ls:
			i -= ls + 1
			n = n.right
		default:
			return n
		}
	}
}

// rank returns the number of keys < k, or, if inclusive, the number <= k.
func (t *treap[K, V]) rank(k K, inclusive bool) int {
	r := 0
	for n := t.root; n != nil; {
		c := t.cmp(n.key, k)
		if c < 0 || (inclusive && c == 0) {
			r += n.left.sz() + 1
			n = n.right
		} else {
			n = n.left
		}
	}
	return r
}

func (t *treap[K, V]) min() *node[K, V] {
	n := t.root
	if n == nil {
		return nil
	}
	for n.left != nil {
		n = n.left
	}
	return n
}

func (t *treap[K, V]) max() *node[K, V] {
	n := t.root
	if n == nil {
		return nil
	}
	for n.right != nil {
		n = n.right
	}
	return n
}

// below returns the greatest node with key < k, or <= k if inclusive.
// Among equal keys it returns the last one.
func (t *treap[K, V]) below(k K, inclusive bool) *node[K, V] {
	var best *node[K, V]
	for n := t.root; n != nil; {
		c := t.cmp(n.key, k)
		if c < 0 || (inclusive && c == 0) {
			best = n
			n = n.right
		} else {
			n = n.left
		}
	}
	return best
}

// above returns the least node with key > k, or >= k if inclusive.
// Among equal keys it returns the first one.
func (t *treap[K, V]) above(k K, inclusive bool) *node[K, V] {
	var best *node[K, V]
	for n := t.root; n != nil; {
		c := t.cmp(n.key, k)
		if c > 0 || (inclusive && c == 0) {
			best = n
			n = n.left
		} else {
			n = n.right
		}
	}
	return best
}

// ascend calls yield for each node in ascending order, starting at the first
// node with key >= lo (if hasLo) and stopping before the first node with
// key >= hi (if hasHi), or when yield returns false.
func (t *treap[K, V]) ascend(lo K, hasLo bool, hi K, hasHi bool, yield func(*node[K, V]) bool) {
	stack := make([]*node[K, V], 0, 32)
	for n := t.root; n != nil; {
		if !hasLo || t.cmp(n.key, lo) >= 0 {
			stack = append(stack, n)
			n = n.left
		} else {
			n = n.right
		}
	}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if hasHi && t.cmp(n.key, hi) >= 0 {
			return
		}
		if !yield(n) {
			return
		}
		for c := n.right; c != nil; c = c.left {
			stack = append(stack, c)
		}
	}
}

// descend calls yield for each node in descending order until yield returns
// false.
func (t *treap[K, V]) descend(yield func(*node[K, V]) bool) {
	stack := make([]*node[K, V], 0, 32)
	for n := t.root; n != nil; n = n.right {
		stack = append(stack, n)
	}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if !yield(n) {
			return
		}
		for c := n.left; c != nil; c = c.right {
			stack = append(stack, c)
		}
	}
}

// each calls yield for every node in ascending order until yield returns
// false.
func (t *treap[K, V]) each(yield func(*node[K, V]) bool) {
	var zero K
	t.ascend(zero, false, zero, false, yield)
}

// keys returns all keys in ascending order.
func (t *treap[K, V]) keys() []K {
	keys := make([]K, 0, t.len())
	t.each(func(n *node[K, V]) bool {
		keys = append(keys, n.key)
		return true
	})
	return keys
}
