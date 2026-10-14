package itertools

import (
	"fmt"
	"slices"
	"testing"
)

func collect[T any](seq func(func([]T) bool)) [][]T {
	var out [][]T
	for v := range seq {
		out = append(out, v)
	}
	return out
}

// refPerms generates r-permutations of indices in lexicographic order (the
// order Python uses).
func refPerms(n, r int) [][]int {
	var out [][]int
	used := make([]bool, n)
	cur := []int{}
	var rec func()
	rec = func() {
		if len(cur) == r {
			out = append(out, slices.Clone(cur))
			return
		}
		for i := range n {
			if !used[i] {
				used[i] = true
				cur = append(cur, i)
				rec()
				cur = cur[:len(cur)-1]
				used[i] = false
			}
		}
	}
	rec()
	return out
}

func rangeSlice(n int) []int {
	s := make([]int, n)
	for i := range s {
		s[i] = i
	}
	return s
}

func TestPermutations(t *testing.T) {
	got := collect(Permutations([]string{"a", "b", "c"}, 2))
	want := [][]string{{"a", "b"}, {"a", "c"}, {"b", "a"}, {"b", "c"}, {"c", "a"}, {"c", "b"}}
	if !slices.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("got %v want %v", got, want)
	}
	for n := range 6 {
		for r := range n + 2 {
			got := collect(Permutations(rangeSlice(n), r))
			want := refPerms(n, r)
			if r > n {
				want = nil
			}
			if len(got) != len(want) || !slices.EqualFunc(got, want, slices.Equal) {
				t.Errorf("n=%d r=%d: got %v want %v", n, r, got, want)
			}
		}
	}
}

func TestPermutationsEdge(t *testing.T) {
	if got := collect(Permutations([]int{}, 0)); len(got) != 1 || len(got[0]) != 0 {
		t.Errorf("empty r=0: %v", got)
	}
	if got := collect(Permutations([]int{1, 2}, 0)); len(got) != 1 || len(got[0]) != 0 {
		t.Errorf("r=0: %v", got)
	}
	if got := collect(Permutations([]int{1, 2}, 3)); len(got) != 0 {
		t.Errorf("r>n: %v", got)
	}
	// Duplicates are treated positionally.
	if got := collect(Permutations([]int{1, 1}, 2)); len(got) != 2 {
		t.Errorf("dups: %v", got)
	}
}

func TestCombinations(t *testing.T) {
	got := collect(Combinations([]int{0, 1, 2, 3}, 3))
	want := [][]int{{0, 1, 2}, {0, 1, 3}, {0, 2, 3}, {1, 2, 3}}
	if !slices.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("got %v want %v", got, want)
	}
	binom := func(n, k int) int {
		r := 1
		for i := 1; i <= k; i++ {
			r = r * (n - k + i) / i
		}
		return r
	}
	for n := range 7 {
		for r := range n + 2 {
			got := collect(Combinations(rangeSlice(n), r))
			wantLen := 0
			if r <= n {
				wantLen = binom(n, r)
			}
			if len(got) != wantLen {
				t.Errorf("n=%d r=%d: len %d want %d", n, r, len(got), wantLen)
			}
			// Each result strictly increasing, results strictly lexicographically increasing.
			for i, c := range got {
				if !slices.IsSorted(c) || len(slices.Compact(slices.Clone(c))) != len(c) {
					t.Errorf("not increasing: %v", c)
				}
				if i > 0 && slices.Compare(got[i-1], c) >= 0 {
					t.Errorf("not ordered: %v then %v", got[i-1], c)
				}
			}
		}
	}
	if got := collect(Combinations([]int{}, 0)); len(got) != 1 {
		t.Errorf("empty r=0: %v", got)
	}
}

func TestCombinationsWithReplacement(t *testing.T) {
	got := collect(CombinationsWithReplacement([]string{"a", "b", "c"}, 2))
	want := [][]string{{"a", "a"}, {"a", "b"}, {"a", "c"}, {"b", "b"}, {"b", "c"}, {"c", "c"}}
	if !slices.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("got %v want %v", got, want)
	}
	if got := collect(CombinationsWithReplacement([]int{}, 2)); len(got) != 0 {
		t.Errorf("empty r=2: %v", got)
	}
	if got := collect(CombinationsWithReplacement([]int{}, 0)); len(got) != 1 {
		t.Errorf("empty r=0: %v", got)
	}
	if got := collect(CombinationsWithReplacement([]int{7}, 3)); len(got) != 1 || !slices.Equal(got[0], []int{7, 7, 7}) {
		t.Errorf("single: %v", got)
	}
	// C(n+r-1, r): n=4, r=3 -> C(6,3) = 20
	if got := collect(CombinationsWithReplacement(rangeSlice(4), 3)); len(got) != 20 {
		t.Errorf("count: %d", len(got))
	}
}

func TestProduct(t *testing.T) {
	got := collect(Product([]int{0, 1}, 3))
	want := [][]int{{0, 0, 0}, {0, 0, 1}, {0, 1, 0}, {0, 1, 1}, {1, 0, 0}, {1, 0, 1}, {1, 1, 0}, {1, 1, 1}}
	if !slices.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("got %v want %v", got, want)
	}
	if got := collect(Product([]int{1, 2}, 0)); len(got) != 1 || len(got[0]) != 0 {
		t.Errorf("repeat=0: %v", got)
	}
	if got := collect(Product([]int{}, 2)); len(got) != 0 {
		t.Errorf("empty: %v", got)
	}
	if got := collect(Product([]int{}, 0)); len(got) != 1 {
		t.Errorf("empty repeat=0: %v", got)
	}
	if got := collect(Product([]int{5}, 2)); len(got) != 1 || !slices.Equal(got[0], []int{5, 5}) {
		t.Errorf("single: %v", got)
	}
	if got := collect(Product(rangeSlice(3), 4)); len(got) != 81 {
		t.Errorf("count: %d", len(got))
	}
}

func TestPowerSet(t *testing.T) {
	got := collect(PowerSet([]int{1, 2, 3}))
	want := [][]int{{}, {1}, {2}, {3}, {1, 2}, {1, 3}, {2, 3}, {1, 2, 3}}
	if !slices.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("got %v want %v", got, want)
	}
	if got := collect(PowerSet([]int(nil))); len(got) != 1 || len(got[0]) != 0 {
		t.Errorf("empty: %v", got)
	}
	if got := collect(PowerSet(rangeSlice(10))); len(got) != 1024 {
		t.Errorf("count: %d", len(got))
	}
}

func TestFreshCopies(t *testing.T) {
	seqs := map[string]func(func([]int) bool){
		"Permutations":                Permutations([]int{1, 2, 3}, 2),
		"Combinations":                Combinations([]int{1, 2, 3}, 2),
		"CombinationsWithReplacement": CombinationsWithReplacement([]int{1, 2, 3}, 2),
		"Product":                     Product([]int{1, 2, 3}, 2),
		"PowerSet":                    PowerSet([]int{1, 2, 3}),
	}
	for name, seq := range seqs {
		var kept [][]int
		var snap []string
		for v := range seq {
			kept = append(kept, v)
			snap = append(snap, fmt.Sprint(v))
		}
		for i, v := range kept {
			if fmt.Sprint(v) != snap[i] {
				t.Errorf("%s: retained slice %d mutated: %v vs %s", name, i, v, snap[i])
			}
		}
		// Mutating yielded slices must not affect later results.
		var a, b []string
		for v := range seq {
			a = append(a, fmt.Sprint(v))
			for i := range v {
				v[i] = -1
			}
		}
		for v := range seq {
			b = append(b, fmt.Sprint(v))
		}
		if !slices.Equal(a, b) {
			t.Errorf("%s: mutation leaked: %v vs %v", name, a, b)
		}
	}
}

func TestEarlyBreak(t *testing.T) {
	seqs := map[string]func(func([]int) bool){
		"Permutations":                Permutations(rangeSlice(5), 3),
		"Combinations":                Combinations(rangeSlice(5), 3),
		"CombinationsWithReplacement": CombinationsWithReplacement(rangeSlice(5), 3),
		"Product":                     Product(rangeSlice(5), 3),
		"PowerSet":                    PowerSet(rangeSlice(5)),
		"Windows":                     Windows(rangeSlice(5), 2),
	}
	for name, seq := range seqs {
		for stop := range 4 {
			n := 0
			for range seq {
				if n == stop {
					break
				}
				n++
			}
			if n != stop {
				t.Errorf("%s: stop=%d n=%d", name, stop, n)
			}
		}
	}
	n := 0
	for range Pairwise(rangeSlice(5)) {
		n++
		break
	}
	for range RunLength([]int{1, 2, 3}) {
		n++
		break
	}
	if n != 2 {
		t.Errorf("Seq2 early break: %d", n)
	}
}

func TestNegativeRPanics(t *testing.T) {
	cases := map[string]func(){
		"Permutations":                func() { Permutations([]int{1}, -1) },
		"Combinations":                func() { Combinations([]int{1}, -1) },
		"CombinationsWithReplacement": func() { CombinationsWithReplacement([]int{1}, -1) },
		"Product":                     func() { Product([]int{1}, -1) },
		"Windows0":                    func() { Windows([]int{1}, 0) },
	}
	for name, f := range cases {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s: expected panic", name)
				}
			}()
			f()
		}()
	}
}

func TestNextPermutation(t *testing.T) {
	s := []int{1, 2, 3}
	var got []string
	for ok := true; ok; ok = NextPermutation(s) {
		got = append(got, fmt.Sprint(s))
	}
	want := []string{"[1 2 3]", "[1 3 2]", "[2 1 3]", "[2 3 1]", "[3 1 2]", "[3 2 1]"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
	if !slices.Equal(s, []int{1, 2, 3}) {
		t.Errorf("not reset to sorted: %v", s)
	}

	d := []int{1, 1, 2}
	got = nil
	for ok := true; ok; ok = NextPermutation(d) {
		got = append(got, fmt.Sprint(d))
	}
	want = []string{"[1 1 2]", "[1 2 1]", "[2 1 1]"}
	if !slices.Equal(got, want) {
		t.Errorf("dups: got %v want %v", got, want)
	}

	if NextPermutation([]int{}) || NextPermutation([]int{5}) {
		t.Error("empty/single should return false")
	}
	last := []string{"c", "b", "a"}
	if NextPermutation(last) || !slices.Equal(last, []string{"a", "b", "c"}) {
		t.Errorf("last: %v", last)
	}

	// Count matches 5! for distinct elements.
	p := rangeSlice(5)
	n := 1
	for NextPermutation(p) {
		n++
	}
	if n != 120 {
		t.Errorf("count %d", n)
	}
}

func TestAccumulate(t *testing.T) {
	mul := func(a, b int) int { return a * b }
	tests := []struct {
		in   []int
		f    func(a, b int) int
		want []int
	}{
		{nil, mul, []int{}},
		{[]int{5}, mul, []int{5}},
		{[]int{1, 2, 3, 4}, mul, []int{1, 2, 6, 24}},
		{[]int{3, 1, 4, 1, 5}, func(a, b int) int { return max(a, b) }, []int{3, 3, 4, 4, 5}},
		{[]int{-1, -2}, func(a, b int) int { return a + b }, []int{-1, -3}},
	}
	for _, tt := range tests {
		if got := Accumulate(tt.in, tt.f); !slices.Equal(got, tt.want) {
			t.Errorf("Accumulate(%v) = %v want %v", tt.in, got, tt.want)
		}
	}
	strs := Accumulate([]string{"a", "b", "c"}, func(a, b string) string { return a + b })
	if !slices.Equal(strs, []string{"a", "ab", "abc"}) {
		t.Errorf("strings: %v", strs)
	}
}

func TestPrefixSums(t *testing.T) {
	s := []int{3, -1, 4, 1, 5}
	p := PrefixSums(s)
	if !slices.Equal(p, []int{0, 3, 2, 6, 7, 12}) {
		t.Fatalf("got %v", p)
	}
	for i := range len(s) + 1 {
		for j := i; j <= len(s); j++ {
			sum := 0
			for _, x := range s[i:j] {
				sum += x
			}
			if p[j]-p[i] != sum {
				t.Errorf("range %d:%d", i, j)
			}
		}
	}
	if got := PrefixSums([]float64(nil)); !slices.Equal(got, []float64{0}) {
		t.Errorf("empty: %v", got)
	}
	if got := PrefixSums([]float64{0.5, 0.25}); !slices.Equal(got, []float64{0, 0.5, 0.75}) {
		t.Errorf("float: %v", got)
	}
}

func TestPairwise(t *testing.T) {
	var got [][2]int
	for a, b := range Pairwise([]int{1, 2, 4, 8}) {
		got = append(got, [2]int{a, b})
	}
	if !slices.Equal(got, [][2]int{{1, 2}, {2, 4}, {4, 8}}) {
		t.Errorf("got %v", got)
	}
	for _, in := range [][]int{nil, {1}} {
		for range Pairwise(in) {
			t.Errorf("expected nothing for %v", in)
		}
	}
}

func TestWindows(t *testing.T) {
	s := []int{1, 2, 3, 4}
	got := collect(Windows(s, 2))
	want := [][]int{{1, 2}, {2, 3}, {3, 4}}
	if !slices.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("got %v", got)
	}
	if got := collect(Windows(s, 4)); len(got) != 1 {
		t.Errorf("k=len: %v", got)
	}
	if got := collect(Windows(s, 5)); len(got) != 0 {
		t.Errorf("k>len: %v", got)
	}
	if got := collect(Windows([]int{}, 1)); len(got) != 0 {
		t.Errorf("empty: %v", got)
	}
	// Windows alias s.
	for w := range Windows(s, 2) {
		w[0] = 0
		break
	}
	if s[0] != 0 {
		t.Error("window should alias input")
	}
	// Capacity is capped so append doesn't clobber s.
	for w := range Windows(s, 2) {
		_ = append(w, 99)
		break
	}
	if s[2] != 3 {
		t.Errorf("append clobbered input: %v", s)
	}
}

func TestRunLength(t *testing.T) {
	type pair struct {
		v byte
		n int
	}
	var got []pair
	for v, n := range RunLength([]byte("aaabccddd")) {
		got = append(got, pair{v, n})
	}
	want := []pair{{'a', 3}, {'b', 1}, {'c', 2}, {'d', 3}}
	if !slices.Equal(got, want) {
		t.Errorf("got %v", got)
	}
	for range RunLength([]int(nil)) {
		t.Error("expected nothing")
	}
	n := 0
	for v, c := range RunLength([]int{7}) {
		if v != 7 || c != 1 {
			t.Errorf("single: %d %d", v, c)
		}
		n++
	}
	if n != 1 {
		t.Error("single: count")
	}
}
