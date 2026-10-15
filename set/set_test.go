package set

import (
	"slices"
	"strings"
	"testing"
)

func TestNewAndBasics(t *testing.T) {
	s := New(1, 2, 2, 3)
	if s.Len() != 3 || len(s) != 3 {
		t.Fatalf("Len = %d", s.Len())
	}
	if !s.Contains(2) || s.Contains(4) {
		t.Fatal("Contains wrong")
	}
	s.Add(4, 4, 5)
	s.Remove(1, 99)
	if got := Sorted(s); !slices.Equal(got, []int{2, 3, 4, 5}) {
		t.Fatalf("Sorted = %v", got)
	}
	if !s.ContainsAll(2, 3) || s.ContainsAll(2, 9) || !s.ContainsAll() {
		t.Fatal("ContainsAll wrong")
	}
	if !s.ContainsAny(9, 2) || s.ContainsAny(9, 10) || s.ContainsAny() {
		t.Fatal("ContainsAny wrong")
	}
	if s.IsEmpty() {
		t.Fatal("IsEmpty wrong")
	}
	s.Clear()
	if !s.IsEmpty() || len(s) != 0 {
		t.Fatal("Clear failed")
	}
	if e := New[string](); e == nil || !e.IsEmpty() {
		t.Fatal("New() should be empty non-nil")
	}
}

func TestNilSetReads(t *testing.T) {
	var s Set[int]
	if s.Contains(1) || s.Len() != 0 || !s.IsEmpty() || s.String() != "Set{}" {
		t.Fatal("nil set reads wrong")
	}
	if c := s.Clone(); c == nil || len(c) != 0 {
		t.Fatal("Clone of nil should be empty non-nil")
	}
	if u := s.Union(New(1)); !u.Equal(New(1)) {
		t.Fatal("nil Union wrong")
	}
	if !s.Equal(New[int]()) || !s.IsSubset(New(1)) {
		t.Fatal("nil Equal/IsSubset wrong")
	}
	s.Remove(1) // delete on nil map is a no-op
	if _, ok := s.TryPop(); ok {
		t.Fatal("TryPop on nil ok")
	}
}

func TestCollect(t *testing.T) {
	s := Collect(slices.Values([]string{"a", "b", "a"}))
	if !s.Equal(New("a", "b")) {
		t.Fatalf("Collect = %v", s)
	}
}

func TestSetOps(t *testing.T) {
	a := New(1, 2, 3, 4)
	b := New(3, 4, 5)
	c := New(4, 5, 6)
	aCopy, bCopy := a.Clone(), b.Clone()
	tests := []struct {
		name string
		got  Set[int]
		want []int
	}{
		{"union", a.Union(b), []int{1, 2, 3, 4, 5}},
		{"union3", a.Union(b, c), []int{1, 2, 3, 4, 5, 6}},
		{"union0", a.Union(), []int{1, 2, 3, 4}},
		{"inter", a.Intersection(b), []int{3, 4}},
		{"inter3", a.Intersection(b, c), []int{4}},
		{"inter0", a.Intersection(), []int{1, 2, 3, 4}},
		{"interEmpty", a.Intersection(New[int]()), []int{}},
		{"diff", a.Difference(b), []int{1, 2}},
		{"diff3", a.Difference(b, New(1)), []int{2}},
		{"diff0", a.Difference(), []int{1, 2, 3, 4}},
		{"symdiff", a.SymmetricDifference(b), []int{1, 2, 5}},
		{"symdiffSelf", a.SymmetricDifference(a), []int{}},
	}
	for _, tt := range tests {
		if got := Sorted(tt.got); !slices.Equal(got, tt.want) {
			t.Errorf("%s = %v want %v", tt.name, got, tt.want)
		}
	}
	if !a.Equal(aCopy) || !b.Equal(bCopy) {
		t.Fatal("operations mutated receivers")
	}
	// Results must be independent of receivers.
	u := a.Union()
	u.Add(100)
	if a.Contains(100) {
		t.Fatal("Union() aliases receiver")
	}
	i := a.Intersection()
	i.Add(100)
	if a.Contains(100) {
		t.Fatal("Intersection() aliases receiver")
	}
}

func TestRelations(t *testing.T) {
	a := New(1, 2)
	b := New(1, 2, 3)
	empty := New[int]()
	if !a.IsSubset(b) || b.IsSubset(a) || !a.IsSubset(a) || !empty.IsSubset(a) {
		t.Fatal("IsSubset wrong")
	}
	if !b.IsSuperset(a) || a.IsSuperset(b) || !a.IsSuperset(empty) {
		t.Fatal("IsSuperset wrong")
	}
	if a.IsDisjoint(b) || !a.IsDisjoint(New(3, 4)) || !empty.IsDisjoint(a) || !New(9).IsDisjoint(b) {
		t.Fatal("IsDisjoint wrong")
	}
	if !a.Equal(New(2, 1)) || a.Equal(b) || a.Equal(New(1, 3)) {
		t.Fatal("Equal wrong")
	}
}

func TestPop(t *testing.T) {
	s := New("x", "y")
	seen := New[string]()
	seen.Add(s.Pop())
	v, ok := s.TryPop()
	if !ok {
		t.Fatal("TryPop failed")
	}
	seen.Add(v)
	if !seen.Equal(New("x", "y")) || !s.IsEmpty() {
		t.Fatalf("popped %v, remaining %v", seen, s)
	}
	if _, ok := s.TryPop(); ok {
		t.Fatal("TryPop on empty ok")
	}
	defer func() {
		if r := recover(); r == nil || !strings.Contains(r.(string), "set: Pop called on empty Set") {
			t.Fatalf("panic = %v", r)
		}
	}()
	s.Pop()
}

func TestIterationAndString(t *testing.T) {
	s := New(10, 2, 1)
	got := slices.Sorted(s.All())
	if !slices.Equal(got, []int{1, 2, 10}) {
		t.Fatalf("All = %v", got)
	}
	ts := s.ToSlice()
	slices.Sort(ts)
	if !slices.Equal(ts, []int{1, 2, 10}) {
		t.Fatalf("ToSlice = %v", ts)
	}
	if str := s.String(); str != "Set{1, 10, 2}" {
		t.Fatalf("String = %q", str)
	}
	if str := New("b", "a").String(); str != "Set{a, b}" {
		t.Fatalf("String = %q", str)
	}
	// String is deterministic across calls.
	for range 20 {
		if New(5, 3, 9, 7, 1).String() != "Set{1, 3, 5, 7, 9}" {
			t.Fatal("String not deterministic")
		}
	}
	for x := range s {
		if !s.Contains(x) {
			t.Fatal("range over set broken")
		}
	}
}
