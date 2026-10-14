package pq

import (
	"cmp"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
)

func mustPanic(t *testing.T, want string, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		if r == nil {
			t.Fatalf("expected panic containing %q", want)
		}
		if s, ok := r.(string); !ok || !strings.Contains(s, want) {
			t.Fatalf("panic = %v, want containing %q", r, want)
		}
	}()
	f()
}

func drain[T any](q *PriorityQueue[T]) []T {
	var out []T
	for !q.IsEmpty() {
		out = append(out, q.Pop())
	}
	return out
}

func TestMinMax(t *testing.T) {
	tests := []struct {
		name string
		in   []int
	}{
		{"empty", nil},
		{"single", []int{7}},
		{"sorted", []int{1, 2, 3, 4, 5}},
		{"reversed", []int{5, 4, 3, 2, 1}},
		{"dups", []int{3, 1, 3, 1, 2, 2}},
		{"negatives", []int{-5, 3, 0, -1, 10, -20}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mn := NewMin[int]()
			mn.Push(tt.in...)
			want := slices.Sorted(slices.Values(tt.in))
			if got := drain(mn); !slices.Equal(got, want) {
				t.Errorf("min: got %v want %v", got, want)
			}
			mx := NewMax[int]()
			mx.Push(tt.in...)
			slices.Reverse(want)
			if got := drain(mx); !slices.Equal(got, want) {
				t.Errorf("max: got %v want %v", got, want)
			}
		})
	}
}

func TestPeek(t *testing.T) {
	q := NewMin[int]()
	if _, ok := q.TryPeek(); ok {
		t.Fatal("TryPeek on empty returned ok")
	}
	if _, ok := q.TryPop(); ok {
		t.Fatal("TryPop on empty returned ok")
	}
	q.Push(4, 2, 8)
	if got := q.Peek(); got != 2 {
		t.Fatalf("Peek = %d, want 2", got)
	}
	if q.Len() != 3 {
		t.Fatalf("Peek changed Len to %d", q.Len())
	}
	if v, ok := q.TryPeek(); !ok || v != 2 {
		t.Fatalf("TryPeek = %d,%v", v, ok)
	}
	if v, ok := q.TryPop(); !ok || v != 2 {
		t.Fatalf("TryPop = %d,%v", v, ok)
	}
	if got := q.Peek(); got != 4 {
		t.Fatalf("Peek = %d, want 4", got)
	}
}

func TestPanics(t *testing.T) {
	q := NewMin[int]()
	mustPanic(t, "pq: Pop called on empty PriorityQueue", func() { q.Pop() })
	mustPanic(t, "pq: Peek called on empty PriorityQueue", func() { q.Peek() })
	mustPanic(t, "pq: Replace called on empty PriorityQueue", func() { q.Replace(1) })
	mustPanic(t, "nil comparator", func() { New[int](nil) })
}

func TestFrom(t *testing.T) {
	in := []int{9, 4, 7, 1, 8, 2, 2, 6}
	orig := slices.Clone(in)
	q := From(in, cmp.Compare[int])
	if !slices.Equal(in, orig) {
		t.Fatal("From modified input")
	}
	in[0] = -100 // must not affect queue
	if got, want := drain(q), slices.Sorted(slices.Values(orig)); !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
	if e := From([]int{}, cmp.Compare[int]); !e.IsEmpty() {
		t.Fatal("From empty not empty")
	}
	if e := From[int](nil, cmp.Compare[int]); !e.IsEmpty() {
		t.Fatal("From nil not empty")
	}
}

func TestPushPop(t *testing.T) {
	q := NewMin[int]()
	if got := q.PushPop(5); got != 5 || !q.IsEmpty() {
		t.Fatalf("PushPop on empty = %d, len %d", got, q.Len())
	}
	q.Push(3, 6)
	if got := q.PushPop(1); got != 1 {
		t.Fatalf("PushPop(1) = %d, want 1", got)
	}
	if got := q.PushPop(3); got != 3 { // equal to top: returns x
		t.Fatalf("PushPop(3) = %d, want 3", got)
	}
	if got := q.PushPop(10); got != 3 {
		t.Fatalf("PushPop(10) = %d, want 3", got)
	}
	if got := drain(q); !slices.Equal(got, []int{6, 10}) {
		t.Fatalf("remaining %v", got)
	}
}

func TestReplace(t *testing.T) {
	q := NewMin[int]()
	q.Push(3, 6)
	if got := q.Replace(1); got != 3 { // returns old top even though x is smaller
		t.Fatalf("Replace(1) = %d, want 3", got)
	}
	if got := drain(q); !slices.Equal(got, []int{1, 6}) {
		t.Fatalf("remaining %v", got)
	}
}

func TestCustomCmp(t *testing.T) {
	type task struct {
		name string
		prio int
	}
	q := New(func(a, b task) int { return cmp.Compare(b.prio, a.prio) })
	q.Push(task{"low", 1}, task{"high", 10}, task{"mid", 5})
	var names []string
	for !q.IsEmpty() {
		names = append(names, q.Pop().name)
	}
	if !slices.Equal(names, []string{"high", "mid", "low"}) {
		t.Fatalf("got %v", names)
	}
}

func TestClearAllToSliceString(t *testing.T) {
	q := NewMin[int]()
	if s := q.String(); s != "PriorityQueue[]" {
		t.Fatalf("String = %q", s)
	}
	q.Push(3, 1, 2)
	if s := q.String(); s != "PriorityQueue[1 2 3]" {
		t.Fatalf("String = %q", s)
	}
	if q.Peek() != 1 || q.Len() != 3 {
		t.Fatal("String modified queue")
	}
	got := slices.Sorted(q.All())
	if !slices.Equal(got, []int{1, 2, 3}) {
		t.Fatalf("All = %v", got)
	}
	// early break
	n := 0
	for range q.All() {
		n++
		break
	}
	if n != 1 {
		t.Fatal("All did not stop")
	}
	s := q.ToSlice()
	slices.Sort(s)
	if !slices.Equal(s, []int{1, 2, 3}) {
		t.Fatalf("ToSlice = %v", s)
	}
	s[0] = 99
	if q.Peek() != 1 {
		t.Fatal("ToSlice aliased internal storage")
	}
	q.Clear()
	if !q.IsEmpty() || q.Len() != 0 {
		t.Fatal("Clear did not empty")
	}
	q.Push(5)
	if q.Pop() != 5 {
		t.Fatal("reuse after Clear failed")
	}
}

// TestRandomized compares against a sorted-slice reference implementation.
func TestRandomized(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	for range 50 {
		q := NewMin[int]()
		var ref []int // kept sorted ascending
		insert := func(v int) {
			i, _ := slices.BinarySearch(ref, v)
			ref = slices.Insert(ref, i, v)
		}
		for range 500 {
			switch op := r.IntN(6); op {
			case 0, 1:
				v := r.IntN(100) - 50
				q.Push(v)
				insert(v)
			case 2:
				v, ok := q.TryPop()
				if ok != (len(ref) > 0) {
					t.Fatalf("TryPop ok=%v ref len %d", ok, len(ref))
				}
				if ok {
					if v != ref[0] {
						t.Fatalf("TryPop=%d want %d", v, ref[0])
					}
					ref = ref[1:]
				}
			case 3:
				v, ok := q.TryPeek()
				if ok != (len(ref) > 0) || (ok && v != ref[0]) {
					t.Fatalf("TryPeek=%d,%v ref %v", v, ok, ref)
				}
			case 4:
				x := r.IntN(100) - 50
				got := q.PushPop(x)
				insert(x)
				want := ref[0]
				ref = ref[1:]
				if got != want {
					t.Fatalf("PushPop=%d want %d", got, want)
				}
			case 5:
				if len(ref) == 0 {
					continue
				}
				x := r.IntN(100) - 50
				got := q.Replace(x)
				want := ref[0]
				ref = ref[1:]
				insert(x)
				if got != want {
					t.Fatalf("Replace=%d want %d", got, want)
				}
			}
			if q.Len() != len(ref) {
				t.Fatalf("Len=%d want %d", q.Len(), len(ref))
			}
		}
		if got := drain(q); !slices.Equal(got, ref) {
			t.Fatalf("final drain %v want %v", got, ref)
		}
	}
}

func TestFromRandomized(t *testing.T) {
	r := rand.New(rand.NewPCG(3, 4))
	for n := range 40 {
		in := make([]int, n)
		for i := range in {
			in[i] = r.IntN(20) - 10
		}
		got := drain(From(in, cmp.Compare[int]))
		if want := slices.Sorted(slices.Values(in)); !slices.Equal(got, want) {
			t.Fatalf("n=%d got %v want %v", n, got, want)
		}
	}
}

func TestNLargestNSmallest(t *testing.T) {
	in := []int{5, -1, 3, 3, 9, 0, 7, -4}
	orig := slices.Clone(in)
	tests := []struct {
		k        int
		largest  []int
		smallest []int
	}{
		{-1, []int{}, []int{}},
		{0, []int{}, []int{}},
		{1, []int{9}, []int{-4}},
		{3, []int{9, 7, 5}, []int{-4, -1, 0}},
		{5, []int{9, 7, 5, 3, 3}, []int{-4, -1, 0, 3, 3}},
		{8, []int{9, 7, 5, 3, 3, 0, -1, -4}, []int{-4, -1, 0, 3, 3, 5, 7, 9}},
		{100, []int{9, 7, 5, 3, 3, 0, -1, -4}, []int{-4, -1, 0, 3, 3, 5, 7, 9}},
	}
	for _, tt := range tests {
		if got := NLargest(in, tt.k); !slices.Equal(got, tt.largest) || got == nil {
			t.Errorf("NLargest(k=%d) = %v want %v", tt.k, got, tt.largest)
		}
		if got := NSmallest(in, tt.k); !slices.Equal(got, tt.smallest) || got == nil {
			t.Errorf("NSmallest(k=%d) = %v want %v", tt.k, got, tt.smallest)
		}
	}
	if !slices.Equal(in, orig) {
		t.Fatal("input modified")
	}
	if got := NLargest([]int(nil), 3); len(got) != 0 {
		t.Fatalf("NLargest(nil) = %v", got)
	}
}

func TestNFuncRandomized(t *testing.T) {
	type item struct{ key, id int }
	byKey := func(a, b item) int { return cmp.Compare(a.key, b.key) }
	r := rand.New(rand.NewPCG(5, 6))
	for range 200 {
		n := r.IntN(30)
		in := make([]item, n)
		for i := range in {
			in[i] = item{r.IntN(10), i}
		}
		k := r.IntN(n+3) - 1
		sortedAsc := slices.Clone(in)
		slices.SortStableFunc(sortedAsc, byKey)
		keys := func(s []item) []int {
			out := make([]int, len(s))
			for i, v := range s {
				out[i] = v.key
			}
			return out
		}
		wantN := max(0, min(k, n))
		gotS := NSmallestFunc(in, k, byKey)
		if !slices.Equal(keys(gotS), keys(sortedAsc[:wantN])) {
			t.Fatalf("NSmallestFunc k=%d got %v want %v", k, gotS, sortedAsc[:wantN])
		}
		desc := slices.Clone(sortedAsc)
		slices.Reverse(desc)
		gotL := NLargestFunc(in, k, byKey)
		if !slices.Equal(keys(gotL), keys(desc[:wantN])) {
			t.Fatalf("NLargestFunc k=%d got %v want %v", k, gotL, desc[:wantN])
		}
	}
}

func TestNLargestNSmallestStable(t *testing.T) {
	type kv struct {
		k int
		v string
	}
	byK := func(a, b kv) int { return cmp.Compare(a.k, b.k) }
	in := []kv{{2, "x"}, {1, "a"}, {3, "y"}, {1, "b"}, {3, "z"}, {1, "c"}}
	tests := []struct {
		name string
		got  []kv
		want []kv
	}{
		{"smallest k=2", NSmallestFunc(in, 2, byK), []kv{{1, "a"}, {1, "b"}}},
		{"smallest k=4", NSmallestFunc(in, 4, byK), []kv{{1, "a"}, {1, "b"}, {1, "c"}, {2, "x"}}},
		{"smallest all", NSmallestFunc(in, 10, byK), []kv{{1, "a"}, {1, "b"}, {1, "c"}, {2, "x"}, {3, "y"}, {3, "z"}}},
		{"largest k=1", NLargestFunc(in, 1, byK), []kv{{3, "y"}}},
		{"largest k=3", NLargestFunc(in, 3, byK), []kv{{3, "y"}, {3, "z"}, {2, "x"}}},
		{"largest all", NLargestFunc(in, 6, byK), []kv{{3, "y"}, {3, "z"}, {2, "x"}, {1, "a"}, {1, "b"}, {1, "c"}}},
	}
	for _, tt := range tests {
		if !slices.Equal(tt.got, tt.want) {
			t.Errorf("%s: got %v, want %v", tt.name, tt.got, tt.want)
		}
	}
}
