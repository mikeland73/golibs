// Package itertools provides Python itertools-style combinatorics and
// sequence helpers built on Go iterators (iter.Seq / iter.Seq2).
//
//	for p := range itertools.Permutations([]int{1, 2, 3}, 2) {
//		fmt.Println(p) // [1 2] [1 3] [2 1] [2 3] [3 1] [3 2]
//	}
//	for c := range itertools.Combinations([]string{"a", "b", "c"}, 2) {
//		fmt.Println(c) // [a b] [a c] [b c]
//	}
//	p := itertools.PrefixSums([]int{3, 1, 4}) // [0 3 4 8]; sum(s[i:j]) == p[j]-p[i]
//
// Unless stated otherwise, every slice yielded by the combinatoric iterators
// (Permutations, Combinations, CombinationsWithReplacement, Product, PowerSet)
// is a fresh copy that the caller may retain or modify. Windows is the
// exception: it yields subslices of the input to avoid allocation.
//
// Elements are chosen by position, not by value, exactly like Python: if the
// input contains duplicates, the outputs will contain duplicates too.
//
// Related functionality elsewhere: for products of different slices use
// lo.CrossJoin2..9; for non-overlapping chunks use slices.Chunk; for zipping
// use lo.Zip2..9; for reductions to a single value use lo.Reduce.
package itertools

import (
	"cmp"
	"iter"
	"slices"
)

// Number is the set of integer and floating-point types.
type Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64
}

func checkR(fn string, r int) {
	if r < 0 {
		panic("itertools: " + fn + " called with negative r")
	}
}

// pick returns a fresh slice {s[idx[0]], s[idx[1]], ...}.
func pick[T any](s []T, idx []int) []T {
	out := make([]T, len(idx))
	for i, j := range idx {
		out[i] = s[j]
	}
	return out
}

// Permutations yields all r-length ordered arrangements of elements of s, in
// the same (lexicographic by position) order as Python's
// itertools.permutations. If r > len(s) nothing is yielded; if r == 0 a single
// empty slice is yielded. It panics if r < 0.
//
// Each yielded slice is a fresh copy. There are n!/(n-r)! results, each
// produced in O(r) amortized time.
func Permutations[T any](s []T, r int) iter.Seq[[]T] {
	checkR("Permutations", r)
	return func(yield func([]T) bool) {
		n := len(s)
		if r > n {
			return
		}
		indices := make([]int, n)
		for i := range indices {
			indices[i] = i
		}
		cycles := make([]int, r)
		for i := range cycles {
			cycles[i] = n - i
		}
		if !yield(pick(s, indices[:r])) {
			return
		}
		for n > 0 {
			advanced := false
			for i := r - 1; i >= 0; i-- {
				cycles[i]--
				if cycles[i] == 0 {
					// Rotate indices[i:] left by one.
					first := indices[i]
					copy(indices[i:], indices[i+1:])
					indices[n-1] = first
					cycles[i] = n - i
					continue
				}
				j := cycles[i]
				indices[i], indices[n-j] = indices[n-j], indices[i]
				if !yield(pick(s, indices[:r])) {
					return
				}
				advanced = true
				break
			}
			if !advanced {
				return
			}
		}
	}
}

// Combinations yields all r-length subsequences of s (elements in their
// original order, without repetition), in the same order as Python's
// itertools.combinations. If r > len(s) nothing is yielded; if r == 0 a single
// empty slice is yielded. It panics if r < 0.
//
// Each yielded slice is a fresh copy. There are C(n, r) results.
func Combinations[T any](s []T, r int) iter.Seq[[]T] {
	checkR("Combinations", r)
	return func(yield func([]T) bool) {
		n := len(s)
		if r > n {
			return
		}
		indices := make([]int, r)
		for i := range indices {
			indices[i] = i
		}
		if !yield(pick(s, indices)) {
			return
		}
		for {
			i := r - 1
			for i >= 0 && indices[i] == i+n-r {
				i--
			}
			if i < 0 {
				return
			}
			indices[i]++
			for j := i + 1; j < r; j++ {
				indices[j] = indices[j-1] + 1
			}
			if !yield(pick(s, indices)) {
				return
			}
		}
	}
}

// CombinationsWithReplacement yields all r-length combinations of elements of
// s where each element may be chosen more than once, in the same order as
// Python's itertools.combinations_with_replacement. If s is empty and r > 0
// nothing is yielded; if r == 0 a single empty slice is yielded. It panics if
// r < 0.
//
// Each yielded slice is a fresh copy. There are C(n+r-1, r) results.
func CombinationsWithReplacement[T any](s []T, r int) iter.Seq[[]T] {
	checkR("CombinationsWithReplacement", r)
	return func(yield func([]T) bool) {
		n := len(s)
		if n == 0 && r > 0 {
			return
		}
		indices := make([]int, r)
		if !yield(pick(s, indices)) {
			return
		}
		for {
			i := r - 1
			for i >= 0 && indices[i] == n-1 {
				i--
			}
			if i < 0 {
				return
			}
			v := indices[i] + 1
			for j := i; j < r; j++ {
				indices[j] = v
			}
			if !yield(pick(s, indices)) {
				return
			}
		}
	}
}

// Product yields the Cartesian product of s with itself repeat times, like
// Python's itertools.product(s, repeat=repeat), in lexicographic (odometer)
// order. If repeat == 0 a single empty slice is yielded; if s is empty and
// repeat > 0 nothing is yielded. It panics if repeat < 0.
//
// Each yielded slice is a fresh copy. There are len(s)^repeat results.
//
// For the product of several different slices use lo.CrossJoin2..9.
func Product[T any](s []T, repeat int) iter.Seq[[]T] {
	checkR("Product", repeat)
	return func(yield func([]T) bool) {
		n := len(s)
		if n == 0 && repeat > 0 {
			return
		}
		indices := make([]int, repeat)
		for {
			if !yield(pick(s, indices)) {
				return
			}
			i := repeat - 1
			for i >= 0 && indices[i] == n-1 {
				indices[i] = 0
				i--
			}
			if i < 0 {
				return
			}
			indices[i]++
		}
	}
}

// PowerSet yields all 2^len(s) subsets of s, ordered by size and then by
// position, like the Python itertools "powerset" recipe:
//
//	PowerSet([]int{1, 2, 3}) // [] [1] [2] [3] [1 2] [1 3] [2 3] [1 2 3]
//
// Each yielded slice is a fresh copy.
func PowerSet[T any](s []T) iter.Seq[[]T] {
	return func(yield func([]T) bool) {
		for r := range len(s) + 1 {
			for c := range Combinations(s, r) {
				if !yield(c) {
					return
				}
			}
		}
	}
}

// NextPermutation rearranges s in place into the next lexicographically
// greater permutation and reports true, like C++ std::next_permutation. If s is
// already the last (descending) permutation, it is reset to the first (sorted
// ascending) permutation and NextPermutation reports false.
//
// Unlike Permutations, it handles duplicates by value, so each distinct
// arrangement is produced exactly once:
//
//	s := []int{1, 1, 2}
//	slices.Sort(s)
//	for ok := true; ok; ok = itertools.NextPermutation(s) {
//		fmt.Println(s) // [1 1 2] [1 2 1] [2 1 1]
//	}
//
// It runs in O(n) time.
func NextPermutation[S ~[]E, E cmp.Ordered](s S) bool {
	n := len(s)
	i := n - 2
	for i >= 0 && cmp.Compare(s[i], s[i+1]) >= 0 {
		i--
	}
	if i < 0 {
		slices.Reverse(s)
		return false
	}
	j := n - 1
	for cmp.Compare(s[j], s[i]) <= 0 {
		j--
	}
	s[i], s[j] = s[j], s[i]
	slices.Reverse(s[i+1:])
	return true
}

// Accumulate returns the running reduction of s: out[0] = s[0] and
// out[i] = f(out[i-1], s[i]). The result has the same length as s. It is like
// Python's itertools.accumulate(s, f) without an initial value.
//
//	Accumulate([]int{1, 2, 3, 4}, func(a, b int) int { return a * b }) // [1 2 6 24]
//	Accumulate([]int{3, 1, 4}, max) // running max: [3 3 4]
func Accumulate[T any](s []T, f func(acc, x T) T) []T {
	out := make([]T, len(s))
	for i, x := range s {
		if i == 0 {
			out[i] = x
		} else {
			out[i] = f(out[i-1], x)
		}
	}
	return out
}

// PrefixSums returns the prefix sums of s with a leading zero, so the result
// has length len(s)+1 and p[i] is the sum of s[:i]. Range sums are then O(1):
//
//	p := PrefixSums(s)
//	sum(s[i:j]) == p[j] - p[i]
func PrefixSums[T Number](s []T) []T {
	p := make([]T, len(s)+1)
	for i, x := range s {
		p[i+1] = p[i] + x
	}
	return p
}

// Pairwise yields successive overlapping pairs (s[i], s[i+1]), like Python's
// itertools.pairwise. Slices with fewer than two elements yield nothing.
//
//	for a, b := range Pairwise([]int{1, 2, 4}) {
//		fmt.Println(b - a) // 1 2
//	}
func Pairwise[T any](s []T) iter.Seq2[T, T] {
	return func(yield func(T, T) bool) {
		for i := 0; i+1 < len(s); i++ {
			if !yield(s[i], s[i+1]) {
				return
			}
		}
	}
}

// Windows yields every overlapping window s[i:i+k] of length k, from left to
// right. If k > len(s) nothing is yielded. It panics if k < 1.
//
// The yielded slices are subslices of s, NOT copies: they share memory with s
// (and with each other), so modifications are visible through s. Their capacity
// is capped at k, so appending to a window never overwrites s. Use
// slices.Clone to retain a window independently.
//
// For non-overlapping chunks use slices.Chunk.
func Windows[T any](s []T, k int) iter.Seq[[]T] {
	if k < 1 {
		panic("itertools: Windows called with k < 1")
	}
	return func(yield func([]T) bool) {
		for i := 0; i+k <= len(s); i++ {
			if !yield(s[i : i+k : i+k]) {
				return
			}
		}
	}
}

// RunLength yields (value, count) for each run of consecutive equal elements
// of s — run-length encoding, like Python's
// ((k, len(list(g))) for k, g in itertools.groupby(s)).
//
//	for v, n := range RunLength([]byte("aaabcc")) {
//		fmt.Printf("%c%d", v, n) // a3b1c2
//	}
func RunLength[T comparable](s []T) iter.Seq2[T, int] {
	return func(yield func(T, int) bool) {
		for i := 0; i < len(s); {
			j := i + 1
			for j < len(s) && s[j] == s[i] {
				j++
			}
			if !yield(s[i], j-i) {
				return
			}
			i = j
		}
	}
}
