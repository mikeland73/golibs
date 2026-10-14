package unionfind

import (
	"math/rand/v2"
	"slices"
	"testing"
)

func TestZeroValue(t *testing.T) {
	var u UnionFind[int]
	if u.Len() != 0 || u.Count() != 0 {
		t.Fatal("zero value not empty")
	}
	if got := u.Groups(); len(got) != 0 {
		t.Fatalf("Groups = %v", got)
	}
	if u.String() != "UnionFind[]" {
		t.Fatalf("String = %q", u.String())
	}
	if !u.Union(1, 2) {
		t.Fatal("Union on zero value failed")
	}
	if u.Len() != 2 || u.Count() != 1 || !u.Connected(1, 2) {
		t.Fatal("zero value Union wrong")
	}
}

func TestBasic(t *testing.T) {
	u := New("a", "b", "c", "d", "e")
	if u.Len() != 5 || u.Count() != 5 {
		t.Fatalf("Len/Count = %d/%d", u.Len(), u.Count())
	}
	if !u.Union("a", "b") || !u.Union("c", "d") || !u.Union("b", "d") {
		t.Fatal("Union should merge")
	}
	if u.Union("a", "c") {
		t.Fatal("Union of connected elements should return false")
	}
	if u.Count() != 2 {
		t.Fatalf("Count = %d, want 2", u.Count())
	}
	if !u.Connected("a", "d") || u.Connected("a", "e") {
		t.Fatal("Connected wrong")
	}
	if u.Size("c") != 4 || u.Size("e") != 1 {
		t.Fatal("Size wrong")
	}
	if u.Find("a") != u.Find("d") || u.Find("e") != "e" {
		t.Fatal("Find wrong")
	}
	want := [][]string{{"a", "b", "c", "d"}, {"e"}}
	if got := u.Groups(); !slices.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("Groups = %v, want %v", got, want)
	}
	if got := u.String(); got != "UnionFind[[a b c d] [e]]" {
		t.Fatalf("String = %q", got)
	}
}

func TestLazyAddAndUnknown(t *testing.T) {
	u := New[int]()
	// Queries on unknown elements do not add them.
	if u.Find(7) != 7 || u.Size(7) != 1 || u.Connected(7, 8) || !u.Connected(7, 7) {
		t.Fatal("unknown element should behave as singleton")
	}
	if u.Len() != 0 || u.Contains(7) {
		t.Fatal("queries must not add elements")
	}
	// Union adds lazily.
	u.Union(1, 2)
	u.Add(3, 1, 3)
	if u.Len() != 3 || u.Count() != 2 || !u.Contains(3) {
		t.Fatalf("Len/Count = %d/%d", u.Len(), u.Count())
	}
	if u.Connected(1, 9) {
		t.Fatal("known vs unknown should not be connected")
	}
	// Self-union adds but does not merge.
	if u.Union(5, 5) {
		t.Fatal("self Union should return false")
	}
	if u.Len() != 4 || u.Count() != 3 {
		t.Fatalf("after self-union Len/Count = %d/%d", u.Len(), u.Count())
	}
}

func TestStructElements(t *testing.T) {
	type point struct{ r, c int }
	grid := []string{
		"11000",
		"11000",
		"00100",
		"00011",
	}
	u := New[point]()
	for r, row := range grid {
		for c, ch := range row {
			if ch != '1' {
				continue
			}
			u.Add(point{r, c})
			if r > 0 && grid[r-1][c] == '1' {
				u.Union(point{r, c}, point{r - 1, c})
			}
			if c > 0 && row[c-1] == '1' {
				u.Union(point{r, c}, point{r, c - 1})
			}
		}
	}
	if u.Count() != 3 {
		t.Fatalf("islands = %d, want 3", u.Count())
	}
	if u.Size(point{0, 0}) != 4 {
		t.Fatalf("Size = %d, want 4", u.Size(point{0, 0}))
	}
}

func TestGroupsDeterministic(t *testing.T) {
	u := New(5, 4, 3, 2, 1)
	u.Union(1, 5)
	u.Union(2, 4)
	want := [][]int{{5, 1}, {4, 2}, {3}}
	for range 3 {
		if got := u.Groups(); !slices.EqualFunc(got, want, slices.Equal) {
			t.Fatalf("Groups = %v, want %v", got, want)
		}
	}
}

func TestRandomizedVsNaive(t *testing.T) {
	rng := rand.New(rand.NewPCG(7, 8))
	const n = 300
	u := New[int]()
	label := map[int]int{} // naive: element -> component label
	relabel := func(from, to int) {
		for k, v := range label {
			if v == from {
				label[k] = to
			}
		}
	}
	for op := range 5000 {
		a, b := rng.IntN(n), rng.IntN(n)
		switch rng.IntN(4) {
		case 0:
			u.Add(a)
			if _, ok := label[a]; !ok {
				label[a] = a
			}
		case 1:
			for _, x := range []int{a, b} {
				if _, ok := label[x]; !ok {
					label[x] = x
				}
			}
			want := label[a] != label[b]
			if want {
				relabel(label[b], label[a])
			}
			if got := u.Union(a, b); got != want {
				t.Fatalf("op %d: Union(%d, %d) = %v, want %v", op, a, b, got, want)
			}
		case 2:
			la, okA := label[a]
			lb, okB := label[b]
			want := a == b || (okA && okB && la == lb)
			if got := u.Connected(a, b); got != want {
				t.Fatalf("op %d: Connected(%d, %d) = %v, want %v", op, a, b, got, want)
			}
			if want != (u.Find(a) == u.Find(b)) {
				t.Fatalf("op %d: Find disagrees with Connected", op)
			}
		case 3:
			want := 1
			if la, ok := label[a]; ok {
				want = 0
				for _, v := range label {
					if v == la {
						want++
					}
				}
			}
			if got := u.Size(a); got != want {
				t.Fatalf("op %d: Size(%d) = %d, want %d", op, a, got, want)
			}
		}
		comps := map[int]bool{}
		for _, v := range label {
			comps[v] = true
		}
		if u.Count() != len(comps) || u.Len() != len(label) {
			t.Fatalf("op %d: Count/Len = %d/%d, want %d/%d", op, u.Count(), u.Len(), len(comps), len(label))
		}
	}
	// Groups must partition exactly the known elements consistently with labels.
	seen := map[int]bool{}
	for _, g := range u.Groups() {
		for _, x := range g {
			if seen[x] {
				t.Fatalf("element %d in multiple groups", x)
			}
			seen[x] = true
			if label[x] != label[g[0]] {
				t.Fatalf("group %v mixes components", g)
			}
		}
		if len(g) != u.Size(g[0]) {
			t.Fatalf("group size %d != Size %d", len(g), u.Size(g[0]))
		}
	}
	if len(seen) != len(label) {
		t.Fatalf("Groups cover %d elements, want %d", len(seen), len(label))
	}
}

func TestLongChain(t *testing.T) {
	u := New[int]()
	const n = 100000
	for i := range n - 1 {
		u.Union(i, i+1)
	}
	if u.Count() != 1 || u.Size(0) != n || !u.Connected(0, n-1) {
		t.Fatal("chain not fully connected")
	}
}
