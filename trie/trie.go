// Package trie provides a prefix tree (trie) over strings, keyed by runes so
// that it is safe for any Unicode text.
//
//	t := trie.New("app", "apple", "apply", "bat")
//	t.Contains("app")         // true
//	t.HasPrefix("ap")         // true
//	t.CountPrefix("app")      // 3
//	t.WithPrefix("appl")      // [apple apply]
//	t.Delete("apple")         // true
//
// The trie is a multiset: inserting a word twice stores it with count 2, and
// Delete removes one occurrence. Len, Count and CountPrefix count with
// multiplicity, while All and WithPrefix list each distinct word once.
//
// For custom traversals (e.g. "Word Search II") walk the nodes directly with
// Root, Walk, Child, Children and IsWord.
//
// Complexity, with L the length of the word or prefix in runes and σ the
// number of distinct children of a node: Insert O(L·σ) worst case (children
// are kept in a sorted slice), Contains/Count/HasPrefix/CountPrefix/Walk
// O(L·log σ), Delete O(L·σ). Listing words costs O(total size of the visited
// subtree).
//
// Words should be valid UTF-8; invalid bytes are treated as
// utf8.RuneError like in a range loop over a string. A Trie is not safe for
// concurrent use.
package trie

import (
	"iter"
	"slices"
	"strings"
	"unicode/utf8"
)

// Node is a node of a Trie. Each node corresponds to the prefix spelled by the
// runes on the path from the root to it.
type Node struct {
	runes  []rune  // sorted child edge labels
	kids   []*Node // kids[i] is the child for runes[i]
	count  int     // number of words ending exactly here
	prefix int     // number of words (with multiplicity) in this subtree
}

// Child returns the child of n reached by rune r, or nil if there is none.
// Child may be called on a nil *Node and then returns nil, so lookups can be
// chained. O(log σ).
func (n *Node) Child(r rune) *Node {
	if n == nil {
		return nil
	}
	if i, ok := slices.BinarySearch(n.runes, r); ok {
		return n.kids[i]
	}
	return nil
}

// Children returns an iterator over the children of n and their edge runes in
// ascending rune order. A nil *Node has no children.
func (n *Node) Children() iter.Seq2[rune, *Node] {
	return func(yield func(rune, *Node) bool) {
		if n == nil {
			return
		}
		for i, r := range n.runes {
			if !yield(r, n.kids[i]) {
				return
			}
		}
	}
}

// IsWord reports whether at least one inserted word ends at n. It returns
// false for a nil *Node. O(1).
func (n *Node) IsWord() bool {
	return n != nil && n.count > 0
}

// Count returns how many times the word ending at n was inserted (0 if no
// word ends here or n is nil). O(1).
func (n *Node) Count() int {
	if n == nil {
		return 0
	}
	return n.count
}

// Walk follows the runes of s starting at n and returns the node reached, or
// nil if the path does not exist. Walk may be called on a nil *Node.
// O(len(s)·log σ).
func (n *Node) Walk(s string) *Node {
	for _, r := range s {
		if n = n.Child(r); n == nil {
			return nil
		}
	}
	return n
}

// childOrNew returns the child for r, creating it if needed.
func (n *Node) childOrNew(r rune) *Node {
	i, ok := slices.BinarySearch(n.runes, r)
	if ok {
		return n.kids[i]
	}
	c := &Node{}
	n.runes = slices.Insert(n.runes, i, r)
	n.kids = slices.Insert(n.kids, i, c)
	return c
}

// removeChild deletes the child edge labeled r.
func (n *Node) removeChild(r rune) {
	if i, ok := slices.BinarySearch(n.runes, r); ok {
		n.runes = slices.Delete(n.runes, i, i+1)
		n.kids = slices.Delete(n.kids, i, i+1)
	}
}

// Trie is a prefix tree storing a multiset of strings.
//
// The zero value is an empty Trie ready to use.
type Trie struct {
	root Node
}

// New returns a Trie containing words.
func New(words ...string) *Trie {
	t := &Trie{}
	for _, w := range words {
		t.Insert(w)
	}
	return t
}

// Insert adds one occurrence of word. The empty string is a valid word.
// O(L·σ).
func (t *Trie) Insert(word string) {
	n := &t.root
	n.prefix++
	for _, r := range word {
		n = n.childOrNew(r)
		n.prefix++
	}
	n.count++
}

// Contains reports whether word has been inserted (and not fully deleted).
// O(L·log σ).
func (t *Trie) Contains(word string) bool {
	return t.root.Walk(word).IsWord()
}

// Count returns how many times word is stored. O(L·log σ).
func (t *Trie) Count(word string) int {
	return t.root.Walk(word).Count()
}

// HasPrefix reports whether any stored word starts with prefix. Every word
// starts with the empty prefix, so HasPrefix("") reports whether the trie is
// non-empty. O(L·log σ).
func (t *Trie) HasPrefix(prefix string) bool {
	n := t.root.Walk(prefix)
	return n != nil && n.prefix > 0
}

// CountPrefix returns the number of stored words, with multiplicity, that
// start with prefix. O(L·log σ).
func (t *Trie) CountPrefix(prefix string) int {
	n := t.root.Walk(prefix)
	if n == nil {
		return 0
	}
	return n.prefix
}

// WithPrefix returns the distinct stored words that start with prefix, in
// ascending order. It returns nil if there are none.
func (t *Trie) WithPrefix(prefix string) []string {
	n := t.root.Walk(prefix)
	if n == nil {
		return nil
	}
	var out []string
	// Re-encode so invalid UTF-8 in prefix becomes U+FFFD, matching the
	// stored words.
	buf := []byte(string([]rune(prefix)))
	collect(n, &buf, func(w string) bool {
		out = append(out, w)
		return true
	})
	return out
}

// collect calls yield for each distinct word in n's subtree in ascending
// order, where *buf holds the UTF-8 encoding of n's prefix. It returns false
// if yield asked to stop.
func collect(n *Node, buf *[]byte, yield func(string) bool) bool {
	if n.count > 0 && !yield(string(*buf)) {
		return false
	}
	for i, r := range n.runes {
		l := len(*buf)
		*buf = utf8.AppendRune(*buf, r)
		ok := collect(n.kids[i], buf, yield)
		*buf = (*buf)[:l]
		if !ok {
			return false
		}
	}
	return true
}

// Delete removes one occurrence of word and reports whether it was present.
// Nodes that no longer lead to any word are pruned. O(L·σ).
func (t *Trie) Delete(word string) bool {
	if !t.Contains(word) {
		return false
	}
	n := &t.root
	n.prefix--
	for _, r := range word {
		c := n.Child(r)
		c.prefix--
		if c.prefix == 0 {
			// c's whole subtree holds no more words.
			n.removeChild(r)
			return true
		}
		n = c
	}
	n.count--
	return true
}

// Len returns the number of stored words, with multiplicity. O(1).
func (t *Trie) Len() int {
	return t.root.prefix
}

// IsEmpty reports whether the trie stores no words. O(1).
func (t *Trie) IsEmpty() bool {
	return t.root.prefix == 0
}

// Clear removes all words. O(1).
func (t *Trie) Clear() {
	t.root = Node{}
}

// All returns an iterator over the distinct stored words in ascending order.
func (t *Trie) All() iter.Seq[string] {
	return func(yield func(string) bool) {
		var buf []byte
		collect(&t.root, &buf, yield)
	}
}

// Root returns the root node, which corresponds to the empty prefix. It is
// never nil. Modifying the trie may invalidate nodes obtained earlier.
func (t *Trie) Root() *Node {
	return &t.root
}

// Walk returns the node for prefix, or nil if no stored word starts with
// prefix. Walk("") returns the root. O(L·log σ).
func (t *Trie) Walk(prefix string) *Node {
	return t.root.Walk(prefix)
}

// String returns the distinct words formatted like "Trie[app apple bat]".
func (t *Trie) String() string {
	var sb strings.Builder
	sb.WriteString("Trie[")
	first := true
	for w := range t.All() {
		if !first {
			sb.WriteByte(' ')
		}
		first = false
		sb.WriteString(w)
	}
	sb.WriteByte(']')
	return sb.String()
}
