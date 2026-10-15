package segtree

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
)

func expectPanic(t *testing.T, want string, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		if r == nil {
			t.Fatalf("expected panic containing %q, got none", want)
		}
		if msg := fmt.Sprint(r); !strings.Contains(msg, want) {
			t.Fatalf("panic = %q, want it to contain %q", msg, want)
		}
	}()
	f()
}

func TestBasic(t *testing.T) {
	vals := []int{5, 2, 8, 1, 9, -3}
	sum, mn, mx := NewSum(vals), NewMin(vals), NewMax(vals)
	tests := []struct{ l, r, sum, min, max int }{
		{0, 6, 22, -3, 9},
		{0, 1, 5, 5, 5},
		{1, 4, 11, 1, 8},
		{4, 6, 6, -3, 9},
		{5, 6, -3, -3, -3},
	}
	for _, tt := range tests {
		if got := sum.Query(tt.l, tt.r); got != tt.sum {
			t.Errorf("sum.Query(%d, %d) = %d, want %d", tt.l, tt.r, got, tt.sum)
		}
		if got := mn.Query(tt.l, tt.r); got != tt.min {
			t.Errorf("min.Query(%d, %d) = %d, want %d", tt.l, tt.r, got, tt.min)
		}
		if got := mx.Query(tt.l, tt.r); got != tt.max {
			t.Errorf("max.Query(%d, %d) = %d, want %d", tt.l, tt.r, got, tt.max)
		}
	}
	if sum.Query(3, 3) != 0 {
		t.Fatal("empty sum should be 0")
	}
	mn.Set(5, 100)
	if mn.Query(0, 6) != 1 || mn.Get(5) != 100 {
		t.Fatal("Set wrong")
	}
	if got := mn.ToSlice(); !slices.Equal(got, []int{5, 2, 8, 1, 9, 100}) {
		t.Fatalf("ToSlice = %v", got)
	}
	if got := mn.String(); got != "Tree[5 2 8 1 9 100]" {
		t.Fatalf("String = %q", got)
	}
	if vals[5] != -3 {
		t.Fatal("constructor retained input slice")
	}
	if sum.Len() != 6 {
		t.Fatal("Len wrong")
	}
}

func TestEmptyAndSingle(t *testing.T) {
	s := NewSum([]float64(nil))
	if s.Len() != 0 || s.Query(0, 0) != 0 || s.String() != "Tree[]" || len(s.ToSlice()) != 0 {
		t.Fatal("empty sum tree wrong")
	}
	one := NewMax([]string{"x"})
	if one.Query(0, 1) != "x" {
		t.Fatal("single wrong")
	}
	one.Set(0, "y")
	if one.Query(0, 1) != "y" {
		t.Fatal("single Set wrong")
	}
}

func TestPanics(t *testing.T) {
	tr := NewMin([]int{1, 2, 3})
	expectPanic(t, "segtree: Get index 3 out of range [0, 3)", func() { tr.Get(3) })
	expectPanic(t, "segtree: Set index -1 out of range [0, 3)", func() { tr.Set(-1, 0) })
	expectPanic(t, "segtree: Query range [2, 1) invalid for length 3", func() { tr.Query(2, 1) })
	expectPanic(t, "segtree: Query range [0, 4) invalid", func() { tr.Query(0, 4) })
	expectPanic(t, "segtree: Query range [-1, 1) invalid", func() { tr.Query(-1, 1) })
	expectPanic(t, "segtree: Query on empty range [1, 1) of a Tree without identity", func() { tr.Query(1, 1) })
	expectPanic(t, "without identity", func() { NewMax([]int{}).Query(0, 0) })
	expectPanic(t, "nil combine", func() { New([]int{1}, 0, nil) })
}

// affine is x -> a*x + b; composition is associative but not commutative.
type affine struct{ a, b int64 }

const mod = 1_000_000_007

// composeAffine returns "apply f, then g".
func composeAffine(f, g affine) affine {
	return affine{g.a * f.a % mod, (g.a*f.b + g.b) % mod}
}

func TestNonCommutativeRandomized(t *testing.T) {
	rng := rand.New(rand.NewPCG(9, 10))
	for n := 1; n <= 40; n++ {
		// String concatenation: the most direct check of combine order.
		strs := make([]string, n)
		for i := range strs {
			strs[i] = string(rune('a' + rng.IntN(26)))
		}
		cat := New(strs, "", func(a, b string) string { return a + b })
		funcs := make([]affine, n)
		for i := range funcs {
			funcs[i] = affine{rng.Int64N(mod), rng.Int64N(mod)}
		}
		comp := New(funcs, affine{1, 0}, composeAffine)
		for range 300 {
			if rng.IntN(3) == 0 {
				i := rng.IntN(n)
				strs[i] = string(rune('a' + rng.IntN(26)))
				cat.Set(i, strs[i])
				funcs[i] = affine{rng.Int64N(mod), rng.Int64N(mod)}
				comp.Set(i, funcs[i])
				continue
			}
			l := rng.IntN(n + 1)
			r := l + rng.IntN(n+1-l)
			if got, want := cat.Query(l, r), strings.Join(strs[l:r], ""); got != want {
				t.Fatalf("n=%d: concat Query(%d, %d) = %q, want %q", n, l, r, got, want)
			}
			want := affine{1, 0}
			for _, f := range funcs[l:r] {
				want = composeAffine(want, f)
			}
			if got := comp.Query(l, r); got != want {
				t.Fatalf("n=%d: affine Query(%d, %d) = %v, want %v", n, l, r, got, want)
			}
		}
	}
}

func TestMinMaxSumRandomized(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 1))
	for _, n := range []int{1, 2, 3, 5, 8, 13, 64, 100, 257} {
		ref := make([]int, n)
		for i := range ref {
			ref[i] = rng.IntN(2001) - 1000
		}
		sum, mn, mx := NewSum(ref), NewMin(ref), NewMax(ref)
		for range 2000 {
			if rng.IntN(2) == 0 {
				i, v := rng.IntN(n), rng.IntN(2001)-1000
				ref[i] = v
				sum.Set(i, v)
				mn.Set(i, v)
				mx.Set(i, v)
				continue
			}
			l := rng.IntN(n)
			r := l + 1 + rng.IntN(n-l)
			s := 0
			for _, v := range ref[l:r] {
				s += v
			}
			if got := sum.Query(l, r); got != s {
				t.Fatalf("n=%d: sum.Query(%d, %d) = %d, want %d", n, l, r, got, s)
			}
			if got, want := mn.Query(l, r), slices.Min(ref[l:r]); got != want {
				t.Fatalf("n=%d: min.Query(%d, %d) = %d, want %d", n, l, r, got, want)
			}
			if got, want := mx.Query(l, r), slices.Max(ref[l:r]); got != want {
				t.Fatalf("n=%d: max.Query(%d, %d) = %d, want %d", n, l, r, got, want)
			}
		}
		if !slices.Equal(sum.ToSlice(), ref) {
			t.Fatalf("n=%d: ToSlice mismatch", n)
		}
		for i := range n {
			if mn.Get(i) != ref[i] {
				t.Fatalf("n=%d: Get(%d) mismatch", n, i)
			}
		}
	}
}
