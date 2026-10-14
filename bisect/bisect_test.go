package bisect

import (
	"cmp"
	"math"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
)

func naiveLeft(s []int, x int) int {
	for i, v := range s {
		if v >= x {
			return i
		}
	}
	return len(s)
}

func naiveRight(s []int, x int) int {
	for i, v := range s {
		if v > x {
			return i
		}
	}
	return len(s)
}

func TestLeftRightTable(t *testing.T) {
	tests := []struct {
		s           []int
		x           int
		left, right int
	}{
		{nil, 5, 0, 0},
		{[]int{}, 5, 0, 0},
		{[]int{3}, 1, 0, 0},
		{[]int{3}, 3, 0, 1},
		{[]int{3}, 5, 1, 1},
		{[]int{1, 2, 2, 2, 5}, 2, 1, 4},
		{[]int{1, 2, 2, 2, 5}, 0, 0, 0},
		{[]int{1, 2, 2, 2, 5}, 3, 4, 4},
		{[]int{1, 2, 2, 2, 5}, 9, 5, 5},
		{[]int{-5, -5, -1, 0}, -5, 0, 2},
		{[]int{7, 7, 7}, 7, 0, 3},
	}
	for _, tt := range tests {
		if got := Left(tt.s, tt.x); got != tt.left {
			t.Errorf("Left(%v, %d) = %d, want %d", tt.s, tt.x, got, tt.left)
		}
		if got := Right(tt.s, tt.x); got != tt.right {
			t.Errorf("Right(%v, %d) = %d, want %d", tt.s, tt.x, got, tt.right)
		}
	}
}

func TestLeftRightRandom(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	for range 2000 {
		n := r.IntN(20)
		s := make([]int, n)
		for i := range s {
			s[i] = r.IntN(11) - 5
		}
		slices.Sort(s)
		x := r.IntN(15) - 7
		if got, want := Left(s, x), naiveLeft(s, x); got != want {
			t.Fatalf("Left(%v, %d) = %d, want %d", s, x, got, want)
		}
		if got, want := Right(s, x), naiveRight(s, x); got != want {
			t.Fatalf("Right(%v, %d) = %d, want %d", s, x, got, want)
		}
		if got, _ := slices.BinarySearch(s, x); got != Left(s, x) {
			t.Fatalf("Left disagrees with slices.BinarySearch")
		}
		lo, hi := r.IntN(15)-7, r.IntN(15)-7
		want := 0
		for _, v := range s {
			if lo <= v && v < hi {
				want++
			}
		}
		if got := CountRange(s, lo, hi); got != want {
			t.Fatalf("CountRange(%v, %d, %d) = %d, want %d", s, lo, hi, got, want)
		}
	}
}

func TestStringsAndNamedSlice(t *testing.T) {
	type names []string
	s := names{"ant", "bee", "bee", "cat"}
	if got := Left(s, "bee"); got != 1 {
		t.Errorf("Left = %d, want 1", got)
	}
	if got := Right(s, "bee"); got != 3 {
		t.Errorf("Right = %d, want 3", got)
	}
	s = Insort(s, "bat")
	if want := (names{"ant", "bat", "bee", "bee", "cat"}); !slices.Equal(s, want) {
		t.Errorf("Insort = %v, want %v", s, want)
	}
}

func TestFloatNaN(t *testing.T) {
	// NaN sorts before everything with cmp.Compare.
	s := []float64{math.NaN(), 1, 2}
	if got := Left(s, 1.0); got != 1 {
		t.Errorf("Left = %d, want 1", got)
	}
}

type person struct {
	name string
	age  int
}

func TestFuncVariants(t *testing.T) {
	people := []person{{"a", 20}, {"b", 30}, {"c", 30}, {"d", 40}}
	byAge := func(p person, age int) int { return cmp.Compare(p.age, age) }
	tests := []struct {
		age         int
		left, right int
	}{
		{10, 0, 0}, {20, 0, 1}, {25, 1, 1}, {30, 1, 3}, {40, 3, 4}, {50, 4, 4},
	}
	for _, tt := range tests {
		if got := LeftFunc(people, tt.age, byAge); got != tt.left {
			t.Errorf("LeftFunc(%d) = %d, want %d", tt.age, got, tt.left)
		}
		if got := RightFunc(people, tt.age, byAge); got != tt.right {
			t.Errorf("RightFunc(%d) = %d, want %d", tt.age, got, tt.right)
		}
	}

	cmpPerson := func(a, b person) int { return cmp.Compare(a.age, b.age) }
	people = InsortFunc(people, person{"e", 30}, cmpPerson)
	var got []string
	for _, p := range people {
		got = append(got, p.name)
	}
	if want := "abced"; strings.Join(got, "") != want {
		t.Errorf("InsortFunc order = %v, want %s (stable, after equals)", got, want)
	}
}

func TestInsort(t *testing.T) {
	var s []int
	r := rand.New(rand.NewPCG(3, 4))
	var all []int
	for range 200 {
		x := r.IntN(50) - 25
		s = Insort(s, x)
		all = append(all, x)
		if !slices.IsSorted(s) {
			t.Fatalf("Insort produced unsorted slice %v", s)
		}
	}
	slices.Sort(all)
	if !slices.Equal(s, all) {
		t.Fatalf("Insort lost elements")
	}
}

func TestCountRangeEmptyInterval(t *testing.T) {
	s := []int{1, 2, 3}
	if got := CountRange(s, 3, 1); got != 0 {
		t.Errorf("CountRange reversed = %d, want 0", got)
	}
	if got := CountRange(s, 2, 2); got != 0 {
		t.Errorf("CountRange empty = %d, want 0", got)
	}
	if got := CountRange(s, 1, 4); got != 3 {
		t.Errorf("CountRange all = %d, want 3", got)
	}
}

func TestFirstTrueLastTrueRandom(t *testing.T) {
	r := rand.New(rand.NewPCG(5, 6))
	for range 3000 {
		lo := r.IntN(41) - 20
		hi := lo + r.IntN(30) - 3 // sometimes hi <= lo
		threshold := r.IntN(60) - 30

		calls := 0
		got := FirstTrue(lo, hi, func(x int) bool {
			calls++
			if x < lo || x >= hi {
				t.Fatalf("pred called out of range: %d not in [%d, %d)", x, lo, hi)
			}
			return x >= threshold
		})
		want := hi
		for x := lo; x < hi; x++ {
			if x >= threshold {
				want = x
				break
			}
		}
		if hi <= lo {
			want = hi
		}
		if got != want {
			t.Fatalf("FirstTrue(%d, %d, x>=%d) = %d, want %d", lo, hi, threshold, got, want)
		}

		got = LastTrue(lo, hi, func(x int) bool {
			if x < lo || x >= hi {
				t.Fatalf("pred called out of range: %d not in [%d, %d)", x, lo, hi)
			}
			return x <= threshold
		})
		want = lo - 1
		for x := lo; x < hi; x++ {
			if x <= threshold {
				want = x
			}
		}
		if got != want {
			t.Fatalf("LastTrue(%d, %d, x<=%d) = %d, want %d", lo, hi, threshold, got, want)
		}
	}
}

func TestFirstTrueExtremeRanges(t *testing.T) {
	for _, th := range []int64{math.MinInt64, -1, 0, 1, math.MaxInt64 - 1, 1 << 62} {
		got := FirstTrue(int64(math.MinInt64), int64(math.MaxInt64), func(x int64) bool { return x >= th })
		if got != th {
			t.Errorf("FirstTrue int64 threshold %d = %d", th, got)
		}
	}
	got := FirstTrue(int64(math.MinInt64), int64(math.MaxInt64), func(int64) bool { return false })
	if got != math.MaxInt64 {
		t.Errorf("FirstTrue none = %d, want MaxInt64", got)
	}

	for _, th := range []uint64{0, 1, 1 << 63, math.MaxUint64 - 1} {
		got := FirstTrue(uint64(0), uint64(math.MaxUint64), func(x uint64) bool { return x >= th })
		if got != th {
			t.Errorf("FirstTrue uint64 threshold %d = %d", th, got)
		}
	}

	// Exhaustive over int8 to make sure the midpoint never overflows.
	for lo := math.MinInt8; lo <= math.MaxInt8; lo++ {
		for _, hi := range []int{lo, min(lo+1, math.MaxInt8), math.MaxInt8} {
			for _, th := range []int{lo, (lo + hi) / 2, hi} {
				got := FirstTrue(int8(lo), int8(hi), func(x int8) bool {
					if int(x) < lo || int(x) >= hi {
						t.Fatalf("pred out of range %d [%d,%d)", x, lo, hi)
					}
					return int(x) >= th
				})
				want := max(th, lo)
				if want > hi {
					want = hi
				}
				if int(got) != want {
					t.Fatalf("FirstTrue int8(%d,%d,>=%d) = %d, want %d", lo, hi, th, got, want)
				}
			}
		}
	}
}

func TestLastTrueLargeRange(t *testing.T) {
	got := LastTrue(int64(math.MinInt64)+1, int64(math.MaxInt64), func(x int64) bool { return x <= 12345 })
	if got != 12345 {
		t.Errorf("LastTrue = %d, want 12345", got)
	}
	// Max sqrt: largest x with x*x <= 1e18.
	got = LastTrue(int64(0), int64(2e9), func(x int64) bool { return x*x <= 1e18 })
	if got != 1e9 {
		t.Errorf("LastTrue sqrt = %d, want 1e9", got)
	}
}

func TestFirstTrueCallCount(t *testing.T) {
	calls := 0
	FirstTrue(0, 1<<20, func(x int) bool { calls++; return x >= 777 })
	if calls > 21 {
		t.Errorf("pred called %d times, want <= 21", calls)
	}
}

func TestFloat(t *testing.T) {
	got := Float(0, 2, func(x float64) bool { return x*x >= 2 }, 1e-12)
	if math.Abs(got-math.Sqrt2) > 1e-11 {
		t.Errorf("Float sqrt2 = %v", got)
	}
	if got*got < 2 {
		t.Errorf("Float returned a value where pred is false: %v", got)
	}

	// Negative range.
	got = Float(-10, 10, func(x float64) bool { return x >= -3.25 }, 1e-9)
	if math.Abs(got+3.25) > 1e-8 {
		t.Errorf("Float negative = %v, want -3.25", got)
	}

	// eps = 0 must terminate.
	calls := 0
	got = Float(0, 1, func(x float64) bool { calls++; return x >= 0.3 }, 0)
	if calls > 200 {
		t.Errorf("Float made %d calls, want <= 200", calls)
	}
	if got < 0.3 || got-0.3 > 1e-15 {
		t.Errorf("Float eps=0 = %v", got)
	}

	// Never true -> original hi.
	if got := Float(0, 5, func(float64) bool { return false }, 1e-6); got != 5 {
		t.Errorf("Float none = %v, want 5", got)
	}
	// Always true -> converges to lo.
	if got := Float(1, 5, func(float64) bool { return true }, 1e-6); got-1 > 1e-6 {
		t.Errorf("Float all = %v, want ~1", got)
	}
	// Empty interval.
	if got := Float(3, 3, func(float64) bool { t.Fatal("pred called"); return true }, 1e-6); got != 3 {
		t.Errorf("Float empty = %v", got)
	}
	// Huge range with tiny eps terminates once precision is exhausted.
	calls = 0
	got = Float(-1e300, 1e300, func(x float64) bool { calls++; return x >= 1 }, 1e-300)
	if calls > 2100 {
		t.Errorf("Float made %d calls, want <= 2100", calls)
	}
	if got != 1 {
		t.Errorf("Float huge range = %v, want 1", got)
	}
}

func TestFloatWideRange(t *testing.T) {
	got := Float(-math.MaxFloat64, math.MaxFloat64, func(x float64) bool { return x >= 3 }, 1e-9)
	if math.Abs(got-3) > 1e-6 {
		t.Fatalf("Float over the full float64 range = %v, want ~3", got)
	}
}
