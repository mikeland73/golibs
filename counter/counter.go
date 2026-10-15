// Package counter provides a multiset that counts hashable values, like
// Python's collections.Counter.
//
// Counter is a map type, so c[x]++, c[x] (0 when missing), len(c) and
// for k, n := range c all work as usual:
//
//	c := counter.New(strings.Split("abracadabra", "")...)
//	c["a"]            // 5
//	c["z"]            // 0
//	c.MostCommon(2)   // [{a 5} {b 2}]
//	counter.New([]rune("listen")...).Equal(counter.New([]rune("silent")...)) // true
//
// Like any map, a nil Counter can be read from but not written to; use New or
// make(counter.Counter[T]). Counts may be zero or negative (e.g. after
// Subtract); Equal treats a missing key and a zero count as the same.
// A Counter is not safe for concurrent use.
package counter

import (
	"cmp"
	"fmt"
	"iter"
	"maps"
	"slices"
	"strings"
)

// Counter maps each value to its count.
type Counter[T comparable] map[T]int

// Entry is a key and its count, as returned by MostCommon and LeastCommon.
type Entry[T comparable] struct {
	Key   T
	Count int
}

// New returns a counter with the occurrences of vals counted. O(len(vals)).
func New[T comparable](vals ...T) Counter[T] {
	c := make(Counter[T])
	c.Add(vals...)
	return c
}

// Collect returns a counter of the values yielded by seq. O(n).
func Collect[T comparable](seq iter.Seq[T]) Counter[T] {
	c := make(Counter[T])
	for v := range seq {
		c[v]++
	}
	return c
}

// Add increments the count of each of vals by one. O(len(vals)).
func (c Counter[T]) Add(vals ...T) {
	for _, v := range vals {
		c[v]++
	}
}

// Subtract decrements the count of each of vals by one. Counts may become
// zero or negative; entries are not deleted (see Positive). O(len(vals)).
func (c Counter[T]) Subtract(vals ...T) {
	for _, v := range vals {
		c[v]--
	}
}

// Update adds the counts from other into c. O(len(other)).
func (c Counter[T]) Update(other Counter[T]) {
	for k, n := range other {
		c[k] += n
	}
}

// Total returns the sum of all counts. O(n).
func (c Counter[T]) Total() int {
	t := 0
	for _, n := range c {
		t += n
	}
	return t
}

// MostCommon returns the n entries with the highest counts, sorted by count
// descending. Ties are broken by the fmt.Sprint representation of the key in
// ascending order, so the result is deterministic. If n <= 0 or n >= len(c),
// all entries are returned. O(m log m) for m entries.
func (c Counter[T]) MostCommon(n int) []Entry[T] {
	return c.sortedEntries(n, func(a, b int) int { return cmp.Compare(b, a) })
}

// LeastCommon returns the n entries with the lowest counts, sorted by count
// ascending. Ties are broken by the fmt.Sprint representation of the key in
// ascending order. If n <= 0 or n >= len(c), all entries are returned.
// O(m log m) for m entries.
func (c Counter[T]) LeastCommon(n int) []Entry[T] {
	return c.sortedEntries(n, cmp.Compare[int])
}

// Equal reports whether c and other have the same count for every key,
// treating missing keys as zero. So New("a") with c["b"] = 0 equals New("a").
// O(len(c) + len(other)).
func (c Counter[T]) Equal(other Counter[T]) bool {
	for k, n := range c {
		if other[k] != n {
			return false
		}
	}
	for k, n := range other {
		if c[k] != n {
			return false
		}
	}
	return true
}

// Clone returns a copy of the counter. Cloning a nil counter returns an empty,
// non-nil counter. O(n).
func (c Counter[T]) Clone() Counter[T] {
	if c == nil {
		return make(Counter[T])
	}
	return maps.Clone(c)
}

// Positive returns a copy of c containing only entries with a count > 0,
// like Python's +c. O(n).
func (c Counter[T]) Positive() Counter[T] {
	res := make(Counter[T])
	for k, n := range c {
		if n > 0 {
			res[k] = n
		}
	}
	return res
}

// Elements returns each key repeated as many times as its count, in
// MostCommon order. Keys with a count <= 0 are omitted. O(m log m + Total()).
func (c Counter[T]) Elements() []T {
	var out []T
	for _, e := range c.MostCommon(0) {
		for range e.Count {
			out = append(out, e.Key)
		}
	}
	return out
}

// String returns the entries in MostCommon order, e.g.
// "Counter{a: 5, b: 2, r: 2}". O(m log m).
func (c Counter[T]) String() string {
	var b strings.Builder
	b.WriteString("Counter{")
	for i, e := range c.MostCommon(0) {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%v: %d", e.Key, e.Count)
	}
	b.WriteByte('}')
	return b.String()
}

// sortedEntries returns the first n entries sorted by countCmp, then by the
// key's fmt representation ascending.
func (c Counter[T]) sortedEntries(n int, countCmp func(a, b int) int) []Entry[T] {
	type keyed struct {
		e   Entry[T]
		str string
	}
	all := make([]keyed, 0, len(c))
	for k, cnt := range c {
		all = append(all, keyed{Entry[T]{k, cnt}, fmt.Sprint(k)})
	}
	slices.SortFunc(all, func(a, b keyed) int {
		if r := countCmp(a.e.Count, b.e.Count); r != 0 {
			return r
		}
		if r := strings.Compare(a.str, b.str); r != 0 {
			return r
		}
		// Distinct keys with identical %v output (rare, e.g. T = any holding
		// 1 and "1"): fall back to the Go-syntax representation.
		return strings.Compare(fmt.Sprintf("%#v", a.e.Key), fmt.Sprintf("%#v", b.e.Key))
	})
	if n <= 0 || n > len(all) {
		n = len(all)
	}
	out := make([]Entry[T], n)
	for i := range out {
		out[i] = all[i].e
	}
	return out
}
