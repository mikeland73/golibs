package lru

import (
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
)

func TestNewPanics(t *testing.T) {
	for _, capacity := range []int{0, -1} {
		func() {
			defer func() {
				r := recover()
				if s, ok := r.(string); !ok || !strings.Contains(s, "lru: New called with non-positive capacity") {
					t.Fatalf("panic = %v", r)
				}
			}()
			New[int, int](capacity)
		}()
	}
}

func TestBasics(t *testing.T) {
	c := New[string, int](2)
	if c.Len() != 0 || c.Cap() != 2 || c.String() != "LRU[]" || len(c.Keys()) != 0 {
		t.Fatal("empty cache wrong")
	}
	if _, ok := c.Get("x"); ok {
		t.Fatal("Get on empty ok")
	}
	if _, ok := c.Peek("x"); ok {
		t.Fatal("Peek on empty ok")
	}
	if _, _, ev := c.Put("a", 1); ev {
		t.Fatal("unexpected eviction")
	}
	c.Put("b", 2)
	if v, ok := c.Get("a"); !ok || v != 1 {
		t.Fatalf("Get a = %d,%v", v, ok)
	}
	// a is MRU now, so b gets evicted.
	k, v, ev := c.Put("c", 3)
	if !ev || k != "b" || v != 2 {
		t.Fatalf("evicted %q %d %v", k, v, ev)
	}
	if c.Contains("b") || !c.Contains("a") || !c.Contains("c") {
		t.Fatal("Contains wrong after eviction")
	}
	if got := c.Keys(); !slices.Equal(got, []string{"c", "a"}) {
		t.Fatalf("Keys = %v", got)
	}
	if c.String() != "LRU[c:3 a:1]" {
		t.Fatalf("String = %q", c.String())
	}
	// Update existing: no eviction, becomes MRU.
	if _, _, ev := c.Put("a", 10); ev {
		t.Fatal("update evicted")
	}
	if got := c.Keys(); !slices.Equal(got, []string{"a", "c"}) {
		t.Fatalf("Keys after update = %v", got)
	}
	// Peek and Contains do not change recency.
	if v, ok := c.Peek("c"); !ok || v != 3 {
		t.Fatal("Peek wrong")
	}
	c.Contains("c")
	if got := c.Keys(); !slices.Equal(got, []string{"a", "c"}) {
		t.Fatalf("Peek changed order: %v", got)
	}
	if k, _, _ := c.Put("d", 4); k != "c" {
		t.Fatalf("expected c evicted, got %q", k)
	}
}

func TestRemoveClear(t *testing.T) {
	c := New[int, string](3)
	c.Put(1, "a")
	c.Put(2, "b")
	c.Put(3, "c")
	if !c.Remove(2) || c.Remove(2) || c.Remove(99) {
		t.Fatal("Remove results wrong")
	}
	if c.Len() != 2 {
		t.Fatalf("Len = %d", c.Len())
	}
	if _, _, ev := c.Put(4, "d"); ev {
		t.Fatal("should have room after Remove")
	}
	if got := c.Keys(); !slices.Equal(got, []int{4, 3, 1}) {
		t.Fatalf("Keys = %v", got)
	}
	var keys []int
	var vals []string
	for k, v := range c.All() {
		keys = append(keys, k)
		vals = append(vals, v)
		if len(keys) == 2 {
			break
		}
	}
	if !slices.Equal(keys, []int{4, 3}) || !slices.Equal(vals, []string{"d", "c"}) {
		t.Fatalf("All = %v %v", keys, vals)
	}
	c.Clear()
	if c.Len() != 0 || c.Contains(1) || len(c.Keys()) != 0 || c.Cap() != 3 {
		t.Fatal("Clear failed")
	}
	c.Put(5, "e")
	if v, ok := c.Get(5); !ok || v != "e" {
		t.Fatal("reuse after Clear failed")
	}
}

func TestCapacityOne(t *testing.T) {
	c := New[int, int](1)
	c.Put(1, 1)
	if k, v, ev := c.Put(2, 2); !ev || k != 1 || v != 1 {
		t.Fatal("cap 1 eviction wrong")
	}
	if _, ok := c.Get(1); ok {
		t.Fatal("1 should be gone")
	}
	c.Remove(2)
	if c.Len() != 0 {
		t.Fatal("not empty")
	}
}

// refLRU is a simple O(n) reference implementation: keys ordered MRU first.
type refLRU struct {
	capacity int
	keys     []int
	vals     map[int]int
}

func (r *refLRU) touch(k int) {
	i := slices.Index(r.keys, k)
	r.keys = slices.Delete(r.keys, i, i+1)
	r.keys = slices.Insert(r.keys, 0, k)
}

func TestRandomized(t *testing.T) {
	rng := rand.New(rand.NewPCG(11, 12))
	for _, capacity := range []int{1, 2, 5, 16} {
		c := New[int, int](capacity)
		ref := &refLRU{capacity: capacity, vals: map[int]int{}}
		for step := range 5000 {
			k := rng.IntN(capacity * 3)
			switch rng.IntN(4) {
			case 0:
				v, ok := c.Get(k)
				rv, rok := ref.vals[k]
				if ok != rok || v != rv {
					t.Fatalf("step %d Get(%d) = %d,%v want %d,%v", step, k, v, ok, rv, rok)
				}
				if ok {
					ref.touch(k)
				}
			case 1, 2:
				v := rng.IntN(1000)
				ek, ev, evicted := c.Put(k, v)
				_, exists := ref.vals[k]
				wantEvict := !exists && len(ref.keys) == capacity
				if evicted != wantEvict {
					t.Fatalf("step %d Put evicted=%v want %v", step, evicted, wantEvict)
				}
				if wantEvict {
					lk := ref.keys[len(ref.keys)-1]
					if ek != lk || ev != ref.vals[lk] {
						t.Fatalf("step %d evicted %d:%d want %d:%d", step, ek, ev, lk, ref.vals[lk])
					}
					ref.keys = ref.keys[:len(ref.keys)-1]
					delete(ref.vals, lk)
				}
				if exists {
					ref.touch(k)
				} else {
					ref.keys = slices.Insert(ref.keys, 0, k)
				}
				ref.vals[k] = v
			case 3:
				_, exists := ref.vals[k]
				if c.Remove(k) != exists {
					t.Fatalf("step %d Remove(%d)", step, k)
				}
				if exists {
					i := slices.Index(ref.keys, k)
					ref.keys = slices.Delete(ref.keys, i, i+1)
					delete(ref.vals, k)
				}
			}
			if !slices.Equal(c.Keys(), ref.keys) {
				t.Fatalf("step %d Keys = %v want %v", step, c.Keys(), ref.keys)
			}
		}
	}
}
