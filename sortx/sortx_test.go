package sortx

import (
	"cmp"
	"math"
	"slices"
	"strings"
	"testing"
)

type person struct {
	name string
	age  int
}

var people = []person{
	{"carol", 30}, {"alice", 25}, {"bob", 30}, {"dave", 25}, {"eve", 35},
}

func names(ps []person) string {
	var b strings.Builder
	for i, p := range ps {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(p.name)
	}
	return b.String()
}

func age(p person) int     { return p.age }
func name(p person) string { return p.name }

func TestBy(t *testing.T) {
	c := By(age)
	if c(people[0], people[1]) != 1 || c(people[1], people[0]) != -1 || c(people[0], people[2]) != 0 {
		t.Error("By compare")
	}
	s := slices.Clone(people)
	slices.SortStableFunc(s, By(age))
	if got := names(s); got != "alice dave carol bob eve" {
		t.Errorf("got %s", got)
	}
	// Works with BinarySearchFunc.
	i, found := slices.BinarySearchFunc(s, 30, func(p person, a int) int { return cmp.Compare(p.age, a) })
	if !found || i != 2 {
		t.Errorf("search %d %v", i, found)
	}
}

func TestDesc(t *testing.T) {
	s := slices.Clone(people)
	slices.SortStableFunc(s, Desc(By(age)))
	if got := names(s); got != "eve carol bob alice dave" {
		t.Errorf("got %s", got)
	}
	ints := []int{3, 1, 2}
	slices.SortFunc(ints, Desc(cmp.Compare[int]))
	if !slices.Equal(ints, []int{3, 2, 1}) {
		t.Errorf("got %v", ints)
	}
}

func TestChain(t *testing.T) {
	s := slices.Clone(people)
	slices.SortFunc(s, Chain(Desc(By(age)), By(name)))
	if got := names(s); got != "eve bob carol alice dave" {
		t.Errorf("got %s", got)
	}
	if Chain[int]()(1, 2) != 0 {
		t.Error("empty chain should be 0")
	}
	calls := 0
	counting := func(a, b int) int { calls++; return 0 }
	Chain(cmp.Compare[int], counting)(1, 2)
	if calls != 0 {
		t.Error("later comparators should only run on ties")
	}
}

func TestSortBy(t *testing.T) {
	s := slices.Clone(people)
	SortBy(s, age)
	if got := names(s); got != "alice dave carol bob eve" {
		t.Errorf("SortBy: %s", got)
	}
	s = slices.Clone(people)
	SortByDesc(s, age)
	if got := names(s); got != "eve carol bob alice dave" {
		t.Errorf("SortByDesc (stable): %s", got)
	}
	// key called exactly once per element.
	calls := 0
	words := []string{"ccc", "a", "bb", "dddd"}
	SortBy(words, func(w string) int { calls++; return len(w) })
	if calls != 4 || !slices.Equal(words, []string{"a", "bb", "ccc", "dddd"}) {
		t.Errorf("calls=%d words=%v", calls, words)
	}
	// Named slice type and empty input.
	type ints []int
	x := ints{3, -1, 2}
	SortBy(x, func(v int) int { return v * v })
	if !slices.Equal(x, ints{-1, 2, 3}) {
		t.Errorf("abs sort: %v", x)
	}
	SortBy([]int(nil), func(v int) int { return v })
	SortByDesc([]int{}, func(v int) int { return v })
	// NaN keys sort first, like cmp.Compare.
	f := []float64{2, math.NaN(), 1}
	SortBy(f, func(v float64) float64 { return v })
	if !math.IsNaN(f[0]) || f[1] != 1 || f[2] != 2 {
		t.Errorf("NaN: %v", f)
	}
}

func TestSortedCopies(t *testing.T) {
	orig := []int{3, 1, 2, 1}
	in := slices.Clone(orig)
	if got := Sorted(in); !slices.Equal(got, []int{1, 1, 2, 3}) {
		t.Errorf("Sorted: %v", got)
	}
	if !slices.Equal(in, orig) {
		t.Error("Sorted modified input")
	}
	if Sorted([]int(nil)) != nil {
		t.Error("Sorted(nil) should be nil")
	}
	if got := Sorted([]string{"b", "a"}); !slices.Equal(got, []string{"a", "b"}) {
		t.Errorf("strings: %v", got)
	}

	p := slices.Clone(people)
	if got := names(SortedBy(p, name)); got != "alice bob carol dave eve" {
		t.Errorf("SortedBy: %s", got)
	}
	if !slices.Equal(p, people) {
		t.Error("SortedBy modified input")
	}
	if got := names(SortedFunc(p, Desc(By(age)))); got != "eve carol bob alice dave" {
		t.Errorf("SortedFunc (stable): %s", got)
	}
	if !slices.Equal(p, people) {
		t.Error("SortedFunc modified input")
	}
	type named []int
	n := SortedBy(named{2, 1}, func(v int) int { return v })
	if !slices.Equal(n, named{1, 2}) {
		t.Errorf("named: %v", n)
	}
}

func TestArgSort(t *testing.T) {
	tests := []struct {
		in   []int
		want []int
	}{
		{nil, []int{}},
		{[]int{5}, []int{0}},
		{[]int{30, 10, 20}, []int{1, 2, 0}},
		{[]int{2, 1, 2, 1}, []int{1, 3, 0, 2}}, // stable
		{[]int{-1, -5, 0}, []int{1, 0, 2}},
	}
	for _, tt := range tests {
		in := slices.Clone(tt.in)
		if got := ArgSort(in); !slices.Equal(got, tt.want) {
			t.Errorf("ArgSort(%v) = %v want %v", tt.in, got, tt.want)
		}
		if !slices.Equal(in, tt.in) {
			t.Error("ArgSort modified input")
		}
	}
}
