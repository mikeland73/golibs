package memo

import (
	"strings"
	"testing"
)

func TestFunc(t *testing.T) {
	calls := 0
	sq := Func(func(x int) int { calls++; return x * x })
	for range 3 {
		if sq(4) != 16 || sq(-3) != 9 {
			t.Fatal("wrong value")
		}
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
	// Separate memoized functions have separate caches.
	sq2 := Func(func(x int) int { calls++; return x * x })
	sq2(4)
	if calls != 3 {
		t.Errorf("caches should be independent, calls = %d", calls)
	}
	// Zero values are cached too.
	zcalls := 0
	z := Func(func(s string) int { zcalls++; return 0 })
	z("")
	z("")
	if zcalls != 1 {
		t.Errorf("zero value not cached: %d", zcalls)
	}
}

func TestFuncStructKey(t *testing.T) {
	type pt struct{ x, y int }
	calls := 0
	f := Func(func(p pt) int { calls++; return p.x + p.y })
	f(pt{1, 2})
	f(pt{1, 2})
	f(pt{2, 1})
	if calls != 2 {
		t.Errorf("calls = %d", calls)
	}
}

func TestRecursive(t *testing.T) {
	calls := 0
	fib := Recursive(func(fib func(int) int, n int) int {
		calls++
		if n < 2 {
			return n
		}
		return fib(n-1) + fib(n-2)
	})
	if got := fib(90); got != 2880067194370816120 {
		t.Errorf("fib(90) = %d", got)
	}
	if calls != 91 {
		t.Errorf("calls = %d, want 91", calls)
	}
	fib(50)
	if calls != 91 {
		t.Error("cached values recomputed")
	}
}

func TestRecursiveDeep(t *testing.T) {
	sum := Recursive(func(sum func(int) int, n int) int {
		if n == 0 {
			return 0
		}
		return n + sum(n-1)
	})
	if got := sum(100000); got != 100000*100001/2 {
		t.Errorf("got %d", got)
	}
}

func TestFunc2(t *testing.T) {
	calls := 0
	add := Func2(func(a int, b string) string { calls++; return strings.Repeat(b, a) })
	add(2, "x")
	add(2, "x")
	add(3, "x")
	add(2, "y")
	if calls != 3 || add(3, "x") != "xxx" {
		t.Errorf("calls = %d", calls)
	}
}

func TestRecursive2(t *testing.T) {
	a, b := "AGGTAB", "GXTXAYB"
	calls := 0
	lcs := Recursive2(func(lcs func(int, int) int, i, j int) int {
		calls++
		if i == len(a) || j == len(b) {
			return 0
		}
		if a[i] == b[j] {
			return 1 + lcs(i+1, j+1)
		}
		return max(lcs(i+1, j), lcs(i, j+1))
	})
	if got := lcs(0, 0); got != 4 {
		t.Errorf("lcs = %d", got)
	}
	if calls > (len(a)+1)*(len(b)+1) {
		t.Errorf("too many calls: %d", calls)
	}
}

func TestFunc3(t *testing.T) {
	calls := 0
	f := Func3(func(a, b, c int) int { calls++; return a*100 + b*10 + c })
	if f(1, 2, 3) != 123 || f(3, 2, 1) != 321 || f(1, 2, 3) != 123 {
		t.Error("wrong value")
	}
	if calls != 2 {
		t.Errorf("calls = %d", calls)
	}
}

func TestRecursive3(t *testing.T) {
	// 0/1 knapsack: best value using items[i:], capacity w, at most k items.
	weights := []int{1, 3, 4, 5}
	values := []int{1, 4, 5, 7}
	calls := 0
	best := Recursive3(func(best func(int, int, int) int, i, w, k int) int {
		calls++
		if i == len(weights) || k == 0 {
			return 0
		}
		res := best(i+1, w, k)
		if weights[i] <= w {
			res = max(res, values[i]+best(i+1, w-weights[i], k-1))
		}
		return res
	})
	if got := best(0, 7, 4); got != 9 {
		t.Errorf("knapsack = %d, want 9", got)
	}
	if got := best(0, 7, 1); got != 7 {
		t.Errorf("knapsack k=1 = %d, want 7", got)
	}
	before := calls
	best(0, 7, 4)
	if calls != before {
		t.Error("cached call recomputed")
	}
}

func TestPanicNotCached(t *testing.T) {
	calls := 0
	f := Func(func(x int) int {
		calls++
		if calls == 1 {
			panic("boom")
		}
		return x
	})
	func() {
		defer func() { _ = recover() }()
		f(1)
	}()
	if f(1) != 1 || calls != 2 {
		t.Errorf("panic result should not be cached; calls = %d", calls)
	}
}
