package trie

import (
	"maps"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
)

func nodeCount(n *Node) int {
	c := 1
	for _, k := range n.kids {
		c += nodeCount(k)
	}
	return c
}

func TestZeroValue(t *testing.T) {
	var tr Trie
	if tr.Len() != 0 || !tr.IsEmpty() || tr.Contains("") || tr.HasPrefix("") {
		t.Fatal("zero value not empty")
	}
	if got := tr.WithPrefix(""); got != nil {
		t.Fatalf("WithPrefix = %v", got)
	}
	if tr.String() != "Trie[]" {
		t.Fatalf("String = %q", tr.String())
	}
	tr.Insert("x")
	if !tr.Contains("x") {
		t.Fatal("Insert on zero value failed")
	}
}

func TestBasic(t *testing.T) {
	tr := New("app", "apple", "apply", "bat", "app")
	tests := []struct {
		word     string
		contains bool
		count    int
		hasPfx   bool
		countPfx int
		withPfx  []string
	}{
		{"", false, 0, true, 5, []string{"app", "apple", "apply", "bat"}},
		{"a", false, 0, true, 4, []string{"app", "apple", "apply"}},
		{"app", true, 2, true, 4, []string{"app", "apple", "apply"}},
		{"appl", false, 0, true, 2, []string{"apple", "apply"}},
		{"apple", true, 1, true, 1, []string{"apple"}},
		{"apples", false, 0, false, 0, nil},
		{"b", false, 0, true, 1, []string{"bat"}},
		{"c", false, 0, false, 0, nil},
	}
	for _, tt := range tests {
		if got := tr.Contains(tt.word); got != tt.contains {
			t.Errorf("Contains(%q) = %v", tt.word, got)
		}
		if got := tr.Count(tt.word); got != tt.count {
			t.Errorf("Count(%q) = %d", tt.word, got)
		}
		if got := tr.HasPrefix(tt.word); got != tt.hasPfx {
			t.Errorf("HasPrefix(%q) = %v", tt.word, got)
		}
		if got := tr.CountPrefix(tt.word); got != tt.countPfx {
			t.Errorf("CountPrefix(%q) = %d", tt.word, got)
		}
		if got := tr.WithPrefix(tt.word); !slices.Equal(got, tt.withPfx) {
			t.Errorf("WithPrefix(%q) = %v", tt.word, got)
		}
	}
	if tr.Len() != 5 {
		t.Fatalf("Len = %d", tr.Len())
	}
	if got := tr.String(); got != "Trie[app apple apply bat]" {
		t.Fatalf("String = %q", got)
	}
}

func TestDeleteAndPrune(t *testing.T) {
	tr := New("app", "app", "apple", "bat")
	before := nodeCount(tr.Root())
	if tr.Delete("ap") || tr.Delete("apples") || tr.Delete("zzz") {
		t.Fatal("Delete of absent word returned true")
	}
	if !tr.Delete("app") || tr.Count("app") != 1 || tr.Len() != 3 {
		t.Fatal("Delete one occurrence failed")
	}
	if nodeCount(tr.Root()) != before {
		t.Fatal("nodes pruned while still in use")
	}
	if !tr.Delete("apple") {
		t.Fatal("Delete(apple) failed")
	}
	// "le" nodes are pruned: root, a, p, p, b, a, t.
	if got := nodeCount(tr.Root()); got != 7 {
		t.Fatalf("nodeCount = %d, want 7", got)
	}
	if !tr.Contains("app") || tr.HasPrefix("appl") {
		t.Fatal("wrong state after deleting apple")
	}
	tr.Delete("app")
	tr.Delete("bat")
	if !tr.IsEmpty() || nodeCount(tr.Root()) != 1 || tr.HasPrefix("") {
		t.Fatal("trie not empty after deleting everything")
	}
}

func TestEmptyWord(t *testing.T) {
	tr := New("", "", "a")
	if !tr.Contains("") || tr.Count("") != 2 || tr.Len() != 3 {
		t.Fatal("empty word not stored")
	}
	if got := tr.WithPrefix(""); !slices.Equal(got, []string{"", "a"}) {
		t.Fatalf("WithPrefix = %q", got)
	}
	if !tr.Delete("") || tr.Count("") != 1 {
		t.Fatal("Delete empty word failed")
	}
	if !tr.Root().IsWord() {
		t.Fatal("root should be a word")
	}
}

func TestUnicode(t *testing.T) {
	tr := New("héllo", "hé", "日本語", "日本", "🙂ok", "z")
	if !tr.Contains("日本") || !tr.HasPrefix("日") || tr.Contains("日") {
		t.Fatal("CJK lookup failed")
	}
	if got := tr.WithPrefix("日"); !slices.Equal(got, []string{"日本", "日本語"}) {
		t.Fatalf("WithPrefix = %v", got)
	}
	if tr.CountPrefix("h") != 2 || tr.CountPrefix("hé") != 2 {
		t.Fatal("CountPrefix with accents failed")
	}
	// Rune order matches byte-wise string order for valid UTF-8.
	all := slices.Collect(tr.All())
	if !slices.IsSorted(all) || len(all) != 6 {
		t.Fatalf("All = %v not sorted", all)
	}
	if n := tr.Walk("🙂"); n == nil || n.Child('o') == nil {
		t.Fatal("emoji path missing")
	}
	// The root has one child per distinct first rune, not per byte.
	n := 0
	for range tr.Root().Children() {
		n++
	}
	if n != 4 {
		t.Fatalf("root children = %d, want 4", n)
	}
}

func TestNodeAPI(t *testing.T) {
	tr := New("cat", "car", "cart", "dog")
	root := tr.Root()
	var runes []rune
	for r, c := range root.Children() {
		runes = append(runes, r)
		if c == nil {
			t.Fatal("nil child")
		}
	}
	if string(runes) != "cd" {
		t.Fatalf("root children = %q", string(runes))
	}
	ca := root.Child('c').Child('a')
	if ca == nil || ca.IsWord() {
		t.Fatal("ca wrong")
	}
	if !ca.Child('r').IsWord() || ca.Child('r').Count() != 1 {
		t.Fatal("car should be a word")
	}
	if ca.Walk("rt") != tr.Walk("cart") || !tr.Walk("cart").IsWord() {
		t.Fatal("Walk mismatch")
	}
	// nil-safe chaining
	var nilNode *Node
	if root.Child('x').Child('y') != nil || nilNode.IsWord() || nilNode.Count() != 0 || nilNode.Walk("a") != nil {
		t.Fatal("nil node handling wrong")
	}
	for range nilNode.Children() {
		t.Fatal("nil node has children")
	}
	if tr.Walk("") != root || tr.Walk("cx") != nil {
		t.Fatal("Trie.Walk wrong")
	}
	// Early break in Children and All.
	for range root.Children() {
		break
	}
	for range tr.All() {
		break
	}
}

func TestClear(t *testing.T) {
	tr := New("a", "b")
	tr.Clear()
	if !tr.IsEmpty() || tr.Contains("a") || len(slices.Collect(tr.All())) != 0 {
		t.Fatal("Clear failed")
	}
}

func TestRandomizedVsMap(t *testing.T) {
	rng := rand.New(rand.NewPCG(11, 12))
	alphabet := []rune("abcé日")
	randWord := func() string {
		var sb strings.Builder
		for range rng.IntN(5) {
			sb.WriteRune(alphabet[rng.IntN(len(alphabet))])
		}
		return sb.String()
	}
	tr := New()
	ref := map[string]int{}
	for op := range 5000 {
		w := randWord()
		switch rng.IntN(5) {
		case 0, 1:
			tr.Insert(w)
			ref[w]++
		case 2:
			want := ref[w] > 0
			if want {
				ref[w]--
				if ref[w] == 0 {
					delete(ref, w)
				}
			}
			if got := tr.Delete(w); got != want {
				t.Fatalf("op %d: Delete(%q) = %v, want %v", op, w, got, want)
			}
		case 3:
			if tr.Count(w) != ref[w] || tr.Contains(w) != (ref[w] > 0) {
				t.Fatalf("op %d: Count(%q) = %d, want %d", op, w, tr.Count(w), ref[w])
			}
		case 4:
			var want []string
			total := 0
			for k, c := range ref {
				if strings.HasPrefix(k, w) {
					want = append(want, k)
					total += c
				}
			}
			slices.Sort(want)
			if got := tr.WithPrefix(w); !slices.Equal(got, want) {
				t.Fatalf("op %d: WithPrefix(%q) = %q, want %q", op, w, got, want)
			}
			if got := tr.CountPrefix(w); got != total {
				t.Fatalf("op %d: CountPrefix(%q) = %d, want %d", op, w, got, total)
			}
			if got := tr.HasPrefix(w); got != (total > 0) {
				t.Fatalf("op %d: HasPrefix(%q) = %v", op, w, got)
			}
		}
		total := 0
		for _, c := range ref {
			total += c
		}
		if tr.Len() != total {
			t.Fatalf("op %d: Len = %d, want %d", op, tr.Len(), total)
		}
	}
	if got, want := slices.Collect(tr.All()), slices.Sorted(maps.Keys(ref)); !slices.Equal(got, want) {
		t.Fatalf("All = %q, want %q", got, want)
	}
	// Every non-root node must lead to at least one word (pruning invariant).
	var check func(n *Node)
	check = func(n *Node) {
		for _, c := range n.kids {
			if c.prefix <= 0 {
				t.Fatal("unpruned empty node")
			}
			check(c)
		}
	}
	check(tr.Root())
}
