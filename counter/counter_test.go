package counter

import (
	"maps"
	"slices"
	"strings"
	"testing"
)

func TestNewAddSubtract(t *testing.T) {
	c := New(strings.Split("abracadabra", "")...)
	want := map[string]int{"a": 5, "b": 2, "r": 2, "c": 1, "d": 1}
	if !maps.Equal(c, want) {
		t.Fatalf("New = %v", c)
	}
	if c["z"] != 0 {
		t.Fatal("missing key not zero")
	}
	c["z"]++
	c.Add("a", "a")
	c.Subtract("c", "c", "q")
	if c["z"] != 1 || c["a"] != 7 || c["c"] != -1 || c["q"] != -1 {
		t.Fatalf("after ops %v", c)
	}
	if c.Total() != 7+2+2-1+1+1-1 {
		t.Fatalf("Total = %d", c.Total())
	}
	if New[int]().Total() != 0 {
		t.Fatal("empty Total")
	}
}

func TestCollectUpdate(t *testing.T) {
	c := Collect(slices.Values([]int{1, 1, 2}))
	c.Update(New(2, 3))
	c.Update(Counter[int]{1: -5})
	c.Update(nil)
	if !maps.Equal(c, Counter[int]{1: -3, 2: 2, 3: 1}) {
		t.Fatalf("got %v", c)
	}
}

func TestMostLeastCommon(t *testing.T) {
	c := New(strings.Split("abracadabra", "")...)
	tests := []struct {
		n    int
		most []Entry[string]
	}{
		{1, []Entry[string]{{"a", 5}}},
		{3, []Entry[string]{{"a", 5}, {"b", 2}, {"r", 2}}},
		{0, []Entry[string]{{"a", 5}, {"b", 2}, {"r", 2}, {"c", 1}, {"d", 1}}},
		{-1, []Entry[string]{{"a", 5}, {"b", 2}, {"r", 2}, {"c", 1}, {"d", 1}}},
		{99, []Entry[string]{{"a", 5}, {"b", 2}, {"r", 2}, {"c", 1}, {"d", 1}}},
	}
	for _, tt := range tests {
		if got := c.MostCommon(tt.n); !slices.Equal(got, tt.most) {
			t.Errorf("MostCommon(%d) = %v want %v", tt.n, got, tt.most)
		}
	}
	if got := c.LeastCommon(3); !slices.Equal(got, []Entry[string]{{"c", 1}, {"d", 1}, {"b", 2}}) {
		t.Errorf("LeastCommon(3) = %v", got)
	}
	if got := New[int]().MostCommon(3); len(got) != 0 {
		t.Errorf("empty MostCommon = %v", got)
	}
	var nilC Counter[int]
	if got := nilC.LeastCommon(0); len(got) != 0 {
		t.Errorf("nil LeastCommon = %v", got)
	}
	// Deterministic across runs despite random map order.
	for range 20 {
		got := New(3, 1, 2, 10, 20).MostCommon(0)
		want := []Entry[int]{{1, 1}, {10, 1}, {2, 1}, {20, 1}, {3, 1}}
		if !slices.Equal(got, want) {
			t.Fatalf("tie order %v", got)
		}
	}
	// Keys with identical %v output are still ordered deterministically.
	for range 20 {
		got := New[any](1, "1").MostCommon(0)
		if got[0].Key != "1" || got[1].Key != 1 {
			t.Fatalf("any tie order %v", got)
		}
	}
}

func TestEqual(t *testing.T) {
	a := New([]rune("listen")...)
	b := New([]rune("silent")...)
	if !a.Equal(b) {
		t.Fatal("anagrams not equal")
	}
	b['x'] = 0
	b.Subtract('s')
	b.Add('s')
	if !a.Equal(b) || !b.Equal(a) {
		t.Fatal("zero entries should be ignored")
	}
	b.Add('x')
	if a.Equal(b) || b.Equal(a) {
		t.Fatal("different counters equal")
	}
	var nilC Counter[int]
	if !nilC.Equal(Counter[int]{5: 0}) || nilC.Equal(New(5)) {
		t.Fatal("nil Equal wrong")
	}
	if New(1).Equal(Counter[int]{1: 1, 2: -1}) {
		t.Fatal("negative counts must matter")
	}
}

func TestCloneAndPositive(t *testing.T) {
	c := Counter[string]{"a": 2, "b": 0, "c": -1}
	cl := c.Clone()
	cl["a"] = 100
	if c["a"] != 2 {
		t.Fatal("Clone aliases")
	}
	p := c.Positive()
	if !maps.Equal(p, Counter[string]{"a": 2}) {
		t.Fatalf("Positive = %v", p)
	}
	if len(c) != 3 {
		t.Fatal("Positive mutated receiver")
	}
	var nilC Counter[int]
	if nc := nilC.Clone(); nc == nil {
		t.Fatal("nil Clone should be non-nil")
	}
}

func TestElementsAndString(t *testing.T) {
	c := New("b", "a", "b", "c")
	c["neg"] = -2
	c["zero"] = 0
	if got := c.Elements(); !slices.Equal(got, []string{"b", "b", "a", "c"}) {
		t.Fatalf("Elements = %v", got)
	}
	if got := c.String(); got != "Counter{b: 2, a: 1, c: 1, zero: 0, neg: -2}" {
		t.Fatalf("String = %q", got)
	}
	if got := New[int]().String(); got != "Counter{}" {
		t.Fatalf("empty String = %q", got)
	}
	if got := New[int]().Elements(); len(got) != 0 {
		t.Fatalf("empty Elements = %v", got)
	}
}
