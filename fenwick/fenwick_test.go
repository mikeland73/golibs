package fenwick

import (
	"fmt"
	"math"
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

func TestZeroAndEmpty(t *testing.T) {
	var z Tree[int]
	if z.Len() != 0 || z.PrefixSum(0) != 0 || z.RangeSum(0, 0) != 0 || z.LowerBound(5) != 0 {
		t.Fatal("zero value wrong")
	}
	if z.String() != "Tree[]" {
		t.Fatalf("String = %q", z.String())
	}
	e := From([]int(nil))
	if e.Len() != 0 || e.ToSlice() == nil || len(e.ToSlice()) != 0 {
		t.Fatal("From(nil) wrong")
	}
	if New[float64](0).Len() != 0 {
		t.Fatal("New(0) wrong")
	}
}

func TestBasic(t *testing.T) {
	tr := From([]int{5, 2, 9, 1, -3})
	if tr.Len() != 5 {
		t.Fatal("Len wrong")
	}
	wantPrefix := []int{0, 5, 7, 16, 17, 14}
	for i, w := range wantPrefix {
		if got := tr.PrefixSum(i); got != w {
			t.Errorf("PrefixSum(%d) = %d, want %d", i, got, w)
		}
	}
	if got := tr.RangeSum(1, 4); got != 12 {
		t.Errorf("RangeSum(1, 4) = %d", got)
	}
	if got := tr.RangeSum(2, 2); got != 0 {
		t.Errorf("RangeSum(2, 2) = %d", got)
	}
	tr.Add(1, 10)
	tr.Set(4, 3)
	if got := tr.ToSlice(); !slices.Equal(got, []int{5, 12, 9, 1, 3}) {
		t.Fatalf("ToSlice = %v", got)
	}
	if tr.Get(1) != 12 || tr.RangeSum(0, 5) != 30 {
		t.Fatal("after update wrong")
	}
	if got := tr.String(); got != "Tree[5 12 9 1 3]" {
		t.Fatalf("String = %q", got)
	}
	// ToSlice is a copy.
	s := tr.ToSlice()
	s[0] = 100
	if tr.Get(0) != 5 {
		t.Fatal("ToSlice aliased")
	}
}

func TestFromDoesNotAlias(t *testing.T) {
	vals := []int{1, 2, 3}
	tr := From(vals)
	tr.Add(0, 5)
	if vals[0] != 1 {
		t.Fatal("From modified input")
	}
	vals[1] = 99
	if tr.Get(1) != 2 {
		t.Fatal("From retained input")
	}
}

func TestLowerBound(t *testing.T) {
	tr := From([]int{0, 3, 0, 2, 5, 0})
	// prefix sums (i+1): 0 3 3 5 10 10
	tests := []struct{ target, want int }{
		{-1, 0}, {0, 0}, {1, 1}, {3, 1}, {4, 3}, {5, 3}, {6, 4}, {10, 4}, {11, 6},
	}
	for _, tt := range tests {
		if got := tr.LowerBound(tt.target); got != tt.want {
			t.Errorf("LowerBound(%d) = %d, want %d", tt.target, got, tt.want)
		}
	}
}

func TestPanics(t *testing.T) {
	tr := New[int](3)
	expectPanic(t, "fenwick: Add index 3 out of range [0, 3)", func() { tr.Add(3, 1) })
	expectPanic(t, "fenwick: Set index -1 out of range [0, 3)", func() { tr.Set(-1, 1) })
	expectPanic(t, "fenwick: Get index 5 out of range [0, 3)", func() { tr.Get(5) })
	expectPanic(t, "fenwick: PrefixSum index 4 out of range [0, 3]", func() { tr.PrefixSum(4) })
	expectPanic(t, "fenwick: RangeSum range [2, 1) invalid for length 3", func() { tr.RangeSum(2, 1) })
	expectPanic(t, "fenwick: RangeSum range [0, 4) invalid", func() { tr.RangeSum(0, 4) })
	expectPanic(t, "fenwick: RangeSum range [-1, 2) invalid", func() { tr.RangeSum(-1, 2) })
	expectPanic(t, "negative length", func() { New[int](-1) })
}

func TestTypes(t *testing.T) {
	u := From([]uint8{1, 2, 3})
	if u.RangeSum(1, 3) != 5 {
		t.Fatal("uint8 wrong")
	}
	f := From([]float64{0.5, 0.25, 0.125})
	if math.Abs(f.PrefixSum(3)-0.875) > 1e-12 {
		t.Fatal("float64 wrong")
	}
	f.Set(1, 1)
	if f.Get(1) != 1 || math.Abs(f.RangeSum(1, 3)-1.125) > 1e-12 {
		t.Fatal("float64 Set wrong")
	}
	type score int32
	s := New[score](2)
	s.Add(1, 7)
	if s.PrefixSum(2) != 7 {
		t.Fatal("named type wrong")
	}
}

func TestRandomizedVsNaive(t *testing.T) {
	rng := rand.New(rand.NewPCG(5, 6))
	for _, n := range []int{1, 2, 3, 7, 8, 9, 16, 33, 100} {
		ref := make([]int64, n)
		for i := range ref {
			ref[i] = rng.Int64N(50)
		}
		tr := From(ref)
		if n%2 == 0 { // also exercise New + Set
			tr = New[int64](n)
			for i, v := range ref {
				tr.Set(i, v)
			}
		}
		for op := range 2000 {
			i := rng.IntN(n)
			switch rng.IntN(5) {
			case 0:
				d := rng.Int64N(20) // keep values non-negative for LowerBound
				tr.Add(i, d)
				ref[i] += d
			case 1:
				v := rng.Int64N(50)
				tr.Set(i, v)
				ref[i] = v
			case 2:
				l := rng.IntN(n + 1)
				r := l + rng.IntN(n+1-l)
				var want int64
				for _, v := range ref[l:r] {
					want += v
				}
				if got := tr.RangeSum(l, r); got != want {
					t.Fatalf("n=%d op %d: RangeSum(%d, %d) = %d, want %d", n, op, l, r, got, want)
				}
			case 3:
				var total int64
				for _, v := range ref {
					total += v
				}
				target := rng.Int64N(total + 10)
				want := n
				var sum int64
				for j, v := range ref {
					sum += v
					if sum >= target {
						want = j
						break
					}
				}
				if got := tr.LowerBound(target); got != want {
					t.Fatalf("n=%d op %d: LowerBound(%d) = %d, want %d (ref %v)", n, op, target, got, want, ref)
				}
			case 4:
				if tr.Get(i) != ref[i] {
					t.Fatalf("n=%d op %d: Get(%d) = %d, want %d", n, op, i, tr.Get(i), ref[i])
				}
			}
		}
		if !slices.Equal(tr.ToSlice(), ref) {
			t.Fatalf("n=%d: ToSlice mismatch", n)
		}
		for i := range n + 1 {
			var want int64
			for _, v := range ref[:i] {
				want += v
			}
			if got := tr.PrefixSum(i); got != want {
				t.Fatalf("n=%d: PrefixSum(%d) = %d, want %d", n, i, got, want)
			}
		}
	}
}
