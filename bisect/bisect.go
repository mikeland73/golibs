// Package bisect provides binary search helpers in the spirit of Python's
// bisect module, plus "binary search on the answer" helpers over integer and
// floating-point ranges.
//
//	s := []int{1, 2, 2, 2, 5}
//	bisect.Left(s, 2)  // 1: first index with s[i] >= 2
//	bisect.Right(s, 2) // 4: first index with s[i] > 2
//	s = bisect.Insort(s, 3) // [1 2 2 2 3 5]
//
//	// Smallest capacity that ships all packages within 5 days:
//	capacity := bisect.FirstTrue(1, 1<<31, func(c int) bool { return canShip(c, 5) })
//
// The standard library already covers some of these cases:
// slices.BinarySearch and slices.BinarySearchFunc return the same index as
// Left and LeftFunc together with a "found" flag, and sort.Search / sort.Find
// search the int range [0, n). This package adds the "right" variants,
// insertion that keeps a slice sorted, and generic, overflow-safe searches
// over arbitrary (including negative) integer ranges and float ranges.
//
// All searches over a slice run in O(log n) comparisons. Insort and
// InsortFunc are O(n) because of the element shift. FirstTrue and LastTrue
// call pred O(log(hi-lo)) times.
package bisect

import (
	"cmp"
	"slices"

	"golang.org/x/exp/constraints"
)

// Left returns the first index i such that s[i] >= x, or len(s) if there is
// none. s must be sorted in ascending order. Inserting x at the returned index
// keeps s sorted and places x before any existing elements equal to x.
//
// It is equivalent to the index returned by slices.BinarySearch. O(log n).
func Left[S ~[]E, E cmp.Ordered](s S, x E) int {
	return LeftFunc(s, x, cmp.Compare[E])
}

// Right returns the first index i such that s[i] > x, or len(s) if there is
// none. s must be sorted in ascending order. Inserting x at the returned index
// keeps s sorted and places x after any existing elements equal to x.
// O(log n).
func Right[S ~[]E, E cmp.Ordered](s S, x E) int {
	return RightFunc(s, x, cmp.Compare[E])
}

// LeftFunc is like Left but uses a custom comparison. cmp(e, target) must
// return a negative number if e sorts before target, zero if they are equal
// and a positive number if e sorts after target. s must be sorted in
// increasing order according to cmp. It returns the first index i with
// cmp(s[i], target) >= 0.
//
// It is equivalent to the index returned by slices.BinarySearchFunc. O(log n).
func LeftFunc[S ~[]E, E, T any](s S, target T, cmp func(E, T) int) int {
	lo, hi := 0, len(s)
	for lo < hi {
		mid := int(uint(lo+hi) >> 1)
		if cmp(s[mid], target) < 0 {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}

// RightFunc is like Right but uses a custom comparison with the same contract
// as in LeftFunc. It returns the first index i with cmp(s[i], target) > 0.
// O(log n).
func RightFunc[S ~[]E, E, T any](s S, target T, cmp func(E, T) int) int {
	lo, hi := 0, len(s)
	for lo < hi {
		mid := int(uint(lo+hi) >> 1)
		if cmp(s[mid], target) <= 0 {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}

// Insort inserts x into the sorted slice s, after any elements equal to x,
// and returns the updated slice. Like append, the result may share or replace
// the underlying array of s, so always use the return value:
//
//	s = bisect.Insort(s, x)
//
// O(log n) comparisons plus O(n) to shift elements.
func Insort[S ~[]E, E cmp.Ordered](s S, x E) S {
	return slices.Insert(s, Right(s, x), x)
}

// InsortFunc is like Insort but orders elements with cmp, which has the same
// shape as the comparator of slices.SortFunc. s must be sorted according to
// cmp. x is inserted after any elements that compare equal to it.
func InsortFunc[S ~[]E, E any](s S, x E, cmp func(a, b E) int) S {
	return slices.Insert(s, RightFunc(s, x, cmp), x)
}

// CountRange returns the number of elements x of the sorted slice s with
// lo <= x < hi. It returns 0 if hi <= lo. O(log n).
func CountRange[S ~[]E, E cmp.Ordered](s S, lo, hi E) int {
	if cmp.Compare(lo, hi) >= 0 { // consistent with Left's ordering, even for NaN
		return 0
	}
	return Left(s, hi) - Left(s, lo)
}

// FirstTrue returns the smallest x in [lo, hi) for which pred(x) is true, or
// hi if there is none. pred must be monotone over the range: false for every
// x below some threshold and true from the threshold on
// (false, ..., false, true, ..., true). If hi <= lo it returns hi without
// calling pred.
//
// This is "binary search on the answer". Any range of any integer type works,
// including negative bounds and the full range of int64, without overflow.
// pred is called O(log(hi-lo)) times.
func FirstTrue[T constraints.Integer](lo, hi T, pred func(T) bool) T {
	for lo < hi {
		// floor((lo+hi)/2) without overflow; >> is arithmetic for signed T.
		mid := (lo & hi) + (lo^hi)>>1
		if pred(mid) {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	return hi
}

// LastTrue returns the largest x in [lo, hi) for which pred(x) is true, or lo-1
// if there is none. pred must be monotone over the range: true for every x up
// to some threshold and false after it (true, ..., true, false, ..., false). If
// hi <= lo it returns lo-1 without calling pred.
//
// Note that lo-1 wraps around if lo is the minimum value of T (for example 0
// for unsigned types); pick lo so that lo-1 is representable if you need to
// detect "none". pred is called O(log(hi-lo)) times.
func LastTrue[T constraints.Integer](lo, hi T, pred func(T) bool) T {
	if hi < lo {
		return lo - 1
	}
	return FirstTrue(lo, hi, func(x T) bool { return !pred(x) }) - 1
}

// Float returns (an approximation of) the smallest x in [lo, hi] for which
// pred(x) is true. pred must be monotone: false below some threshold and true
// from it on. The search stops once hi-lo <= eps or when float64 precision
// is exhausted (at most ~2100 halvings, even for the widest finite range).
// It returns the upper end of the final interval, i.e. a value for which
// pred is known to be true unless pred was false everywhere, in which case
// the original hi is returned. lo and hi must be finite; any finite range
// works, even one wider than MaxFloat64.
//
// Example: the square root of 2 to 1e-9 precision:
//
//	bisect.Float(0, 2, func(x float64) bool { return x*x >= 2 }, 1e-9)
func Float(lo, hi float64, pred func(float64) bool, eps float64) float64 {
	// 2100 halvings take any finite float64 interval down to adjacent
	// representable values, so the cap is only a safety net.
	for range 2100 {
		if hi-lo <= eps {
			break
		}
		mid := lo/2 + hi/2 // hi-lo could overflow to +Inf
		if mid <= lo || mid >= hi {
			break // no representable midpoint left
		}
		if pred(mid) {
			hi = mid
		} else {
			lo = mid
		}
	}
	return hi
}
