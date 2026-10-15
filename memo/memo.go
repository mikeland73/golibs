// Package memo memoizes pure functions, like Python's @functools.cache, which
// is handy for top-down dynamic programming.
//
// Recursive lets a function call the memoized version of itself:
//
//	fib := memo.Recursive(func(fib func(int) int, n int) int {
//		if n < 2 {
//			return n
//		}
//		return fib(n-1) + fib(n-2)
//	})
//	fib(90) // 2880067194370816120, in O(n) calls
//
// Func2/Recursive2 and Func3/Recursive3 handle two and three arguments (for
// more, pack the arguments into a comparable struct and use the
// single-argument versions).
//
// The cache is unbounded and lives as long as the returned function. The
// returned functions are NOT safe for concurrent use; create one per goroutine
// or guard calls with a mutex. Results for a given key are computed at most
// once per memoized function (unless the function panics).
//
// For a function with no arguments, use sync.OnceValue.
package memo

// Func returns a memoized version of f: the first call with a given k
// computes f(k) and caches it; later calls with the same k return the cached
// value in O(1) expected time.
//
// Func is not recursive-aware: recursive calls inside f go to f itself and
// are not cached. Use Recursive for recursive functions.
func Func[K comparable, V any](f func(K) V) func(K) V {
	cache := make(map[K]V)
	return func(k K) V {
		if v, ok := cache[k]; ok {
			return v
		}
		v := f(k)
		cache[k] = v
		return v
	}
}

// Recursive returns a memoized version of a recursive function. f receives
// self, the memoized function itself, and must use it for recursive calls so
// that every subproblem is cached:
//
//	climb := memo.Recursive(func(climb func(int) int, n int) int {
//		if n <= 1 {
//			return 1
//		}
//		return climb(n-1) + climb(n-2)
//	})
//
// Recursion depth is limited only by Go's (growable) stack.
func Recursive[K comparable, V any](f func(self func(K) V, k K) V) func(K) V {
	cache := make(map[K]V)
	var self func(K) V
	self = func(k K) V {
		if v, ok := cache[k]; ok {
			return v
		}
		v := f(self, k)
		cache[k] = v
		return v
	}
	return self
}

type key2[A, B comparable] struct {
	a A
	b B
}

type key3[A, B, C comparable] struct {
	a A
	b B
	c C
}

// Func2 is Func for functions of two arguments. Results are cached by the
// pair (a, b).
func Func2[A, B comparable, V any](f func(A, B) V) func(A, B) V {
	cache := make(map[key2[A, B]]V)
	return func(a A, b B) V {
		k := key2[A, B]{a, b}
		if v, ok := cache[k]; ok {
			return v
		}
		v := f(a, b)
		cache[k] = v
		return v
	}
}

// Recursive2 is Recursive for functions of two arguments, e.g. grid or
// two-string DP:
//
//	lcs := memo.Recursive2(func(lcs func(int, int) int, i, j int) int {
//		if i == len(a) || j == len(b) {
//			return 0
//		}
//		if a[i] == b[j] {
//			return 1 + lcs(i+1, j+1)
//		}
//		return max(lcs(i+1, j), lcs(i, j+1))
//	})
func Recursive2[A, B comparable, V any](f func(self func(A, B) V, a A, b B) V) func(A, B) V {
	cache := make(map[key2[A, B]]V)
	var self func(A, B) V
	self = func(a A, b B) V {
		k := key2[A, B]{a, b}
		if v, ok := cache[k]; ok {
			return v
		}
		v := f(self, a, b)
		cache[k] = v
		return v
	}
	return self
}

// Func3 is Func for functions of three arguments. Results are cached by the
// triple (a, b, c).
func Func3[A, B, C comparable, V any](f func(A, B, C) V) func(A, B, C) V {
	cache := make(map[key3[A, B, C]]V)
	return func(a A, b B, c C) V {
		k := key3[A, B, C]{a, b, c}
		if v, ok := cache[k]; ok {
			return v
		}
		v := f(a, b, c)
		cache[k] = v
		return v
	}
}

// Recursive3 is Recursive for functions of three arguments, e.g. knapsack or
// DP with an extra state such as "remaining moves".
func Recursive3[A, B, C comparable, V any](f func(self func(A, B, C) V, a A, b B, c C) V) func(A, B, C) V {
	cache := make(map[key3[A, B, C]]V)
	var self func(A, B, C) V
	self = func(a A, b B, c C) V {
		k := key3[A, B, C]{a, b, c}
		if v, ok := cache[k]; ok {
			return v
		}
		v := f(self, a, b, c)
		cache[k] = v
		return v
	}
	return self
}
