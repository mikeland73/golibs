package sorted

import (
	"cmp"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
)

func expectPanic(t *testing.T, want string, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		if r == nil {
			t.Fatalf("expected panic containing %q, got none", want)
		}
		if msg := fmt.Sprint(r); !strings.Contains(msg, want) {
			t.Fatalf("panic = %q, want it to contain %q", msg, want)
		}
	}()
	f()
}

// checkInvariants verifies BST order, heap order on priorities and subtree
// sizes for the whole tree.
func checkInvariants[K, V any](t *testing.T, tr *treap[K, V]) {
	t.Helper()
	var walk func(n *node[K, V]) int
	walk = func(n *node[K, V]) int {
		if n == nil {
			return 0
		}
		if n.left != nil {
			if n.left.prio > n.prio {
				t.Fatalf("heap order violated")
			}
			if tr.cmp(n.left.key, n.key) > 0 {
				t.Fatalf("BST order violated")
			}
		}
		if n.right != nil {
			if n.right.prio > n.prio {
				t.Fatalf("heap order violated")
			}
			if c := tr.cmp(n.right.key, n.key); c < 0 || (c == 0 && !tr.multi) {
				t.Fatalf("BST order violated")
			}
		}
		size := 1 + walk(n.left) + walk(n.right)
		if size != n.size {
			t.Fatalf("size = %d, want %d", n.size, size)
		}
		return size
	}
	walk(tr.root)
}

func collect[T any](seq func(func(T) bool)) []T {
	var out []T
	for v := range seq {
		out = append(out, v)
	}
	return out
}

func collect2[K, V any](seq func(func(K, V) bool)) ([]K, []V) {
	var ks []K
	var vs []V
	for k, v := range seq {
		ks = append(ks, k)
		vs = append(vs, v)
	}
	return ks, vs
}

func reversed[T any](s []T) []T {
	s = slices.Clone(s)
	slices.Reverse(s)
	return s
}

// ---------- Map ----------

func TestMapBasic(t *testing.T) {
	m := NewMap[string, int]()
	if !m.IsEmpty() || m.Len() != 0 {
		t.Fatal("new map not empty")
	}
	if _, ok := m.Get("x"); ok {
		t.Fatal("Get on empty map ok")
	}
	if _, _, ok := m.Min(); ok {
		t.Fatal("Min on empty map ok")
	}
	if _, _, ok := m.Max(); ok {
		t.Fatal("Max on empty map ok")
	}
	if _, _, ok := m.PopMin(); ok {
		t.Fatal("PopMin on empty map ok")
	}
	if _, _, ok := m.PopMax(); ok {
		t.Fatal("PopMax on empty map ok")
	}
	if m.Delete("x") {
		t.Fatal("Delete on empty map true")
	}
	if got := m.String(); got != "Map{}" {
		t.Fatalf("String = %q", got)
	}

	m.Put("b", 2)
	m.Put("a", 1)
	m.Put("c", 3)
	m.Put("b", 20) // replace
	if m.Len() != 3 {
		t.Fatalf("Len = %d, want 3", m.Len())
	}
	if v, ok := m.Get("b"); !ok || v != 20 {
		t.Fatalf("Get(b) = %d, %v", v, ok)
	}
	if !m.Contains("a") || m.Contains("z") {
		t.Fatal("Contains wrong")
	}
	if got := m.String(); got != "Map{a: 1, b: 20, c: 3}" {
		t.Fatalf("String = %q", got)
	}
	if got := m.Keys(); !slices.Equal(got, []string{"a", "b", "c"}) {
		t.Fatalf("Keys = %v", got)
	}
	if k, v, ok := m.Min(); !ok || k != "a" || v != 1 {
		t.Fatalf("Min = %v %v %v", k, v, ok)
	}
	if k, v, ok := m.Max(); !ok || k != "c" || v != 3 {
		t.Fatalf("Max = %v %v %v", k, v, ok)
	}
	if k, v := m.At(1); k != "b" || v != 20 {
		t.Fatalf("At(1) = %v %v", k, v)
	}
	if r := m.Rank("bb"); r != 2 {
		t.Fatalf("Rank(bb) = %d", r)
	}
	if k, _, ok := m.Floor("bb"); !ok || k != "b" {
		t.Fatalf("Floor(bb) = %v %v", k, ok)
	}
	if k, _, ok := m.Ceiling("bb"); !ok || k != "c" {
		t.Fatalf("Ceiling(bb) = %v %v", k, ok)
	}
	if k, _, ok := m.Lower("b"); !ok || k != "a" {
		t.Fatalf("Lower(b) = %v %v", k, ok)
	}
	if k, _, ok := m.Higher("b"); !ok || k != "c" {
		t.Fatalf("Higher(b) = %v %v", k, ok)
	}
	if _, _, ok := m.Lower("a"); ok {
		t.Fatal("Lower(a) ok")
	}
	if _, _, ok := m.Higher("c"); ok {
		t.Fatal("Higher(c) ok")
	}
	ks, vs := collect2(m.Backward())
	if !slices.Equal(ks, []string{"c", "b", "a"}) || !slices.Equal(vs, []int{3, 20, 1}) {
		t.Fatalf("Backward = %v %v", ks, vs)
	}
	ks, _ = collect2(m.Range("a", "c"))
	if !slices.Equal(ks, []string{"a", "b"}) {
		t.Fatalf("Range(a, c) = %v", ks)
	}
	if k, v, ok := m.PopMin(); !ok || k != "a" || v != 1 {
		t.Fatalf("PopMin = %v %v %v", k, v, ok)
	}
	if k, v, ok := m.PopMax(); !ok || k != "c" || v != 3 {
		t.Fatalf("PopMax = %v %v %v", k, v, ok)
	}
	if !m.Delete("b") || !m.IsEmpty() {
		t.Fatal("Delete(b) failed")
	}
	m.Put("x", 1)
	m.Clear()
	if !m.IsEmpty() || m.Contains("x") {
		t.Fatal("Clear failed")
	}
}

func TestMapFuncReverse(t *testing.T) {
	m := NewMapFunc[int, string](func(a, b int) int { return cmp.Compare(b, a) })
	for i := range 5 {
		m.Put(i, fmt.Sprint(i))
	}
	if got := m.Keys(); !slices.Equal(got, []int{4, 3, 2, 1, 0}) {
		t.Fatalf("Keys = %v", got)
	}
	// "Floor" is relative to the comparator's order.
	if k, _, ok := m.Floor(10); ok {
		t.Fatalf("Floor(10) = %v, want none", k)
	}
	if k, _, _ := m.Ceiling(10); k != 4 {
		t.Fatalf("Ceiling(10) = %v, want 4", k)
	}
}

func TestMapIteratorEarlyBreak(t *testing.T) {
	m := NewMap[int, int]()
	for i := range 100 {
		m.Put(i, i*i)
	}
	n := 0
	for k := range m.All() {
		if k == 10 {
			break
		}
		n++
	}
	if n != 10 {
		t.Fatalf("All break: n = %d", n)
	}
	n = 0
	for range m.Backward() {
		n++
		if n == 3 {
			break
		}
	}
	for k := range m.Range(5, 50) {
		if k == 7 {
			break
		}
	}
}

func TestMapRangeEdges(t *testing.T) {
	m := NewMap[int, bool]()
	for _, k := range []int{10, 20, 30, 40} {
		m.Put(k, true)
	}
	tests := []struct {
		lo, hi int
		want   []int
	}{
		{0, 100, []int{10, 20, 30, 40}},
		{10, 40, []int{10, 20, 30}},
		{11, 41, []int{20, 30, 40}},
		{20, 20, nil},
		{30, 20, nil},
		{41, 100, nil},
		{-5, 10, nil},
	}
	for _, tt := range tests {
		got, _ := collect2(m.Range(tt.lo, tt.hi))
		if !slices.Equal(got, tt.want) {
			t.Errorf("Range(%d, %d) = %v, want %v", tt.lo, tt.hi, got, tt.want)
		}
	}
}

func TestPanics(t *testing.T) {
	expectPanic(t, "sorted: Map.At index 0 out of range [0, 0)", func() { NewMap[int, int]().At(0) })
	expectPanic(t, "sorted: Set.At index -1 out of range [0, 1)", func() { NewSet(1).At(-1) })
	expectPanic(t, "sorted: List.At index 3 out of range [0, 3)", func() { NewList(1, 1, 1).At(3) })
	expectPanic(t, "nil comparator", func() { NewMapFunc[int, int](nil) })
	expectPanic(t, "nil comparator", func() { NewSetFunc[int](nil) })
	expectPanic(t, "nil comparator", func() { NewListFunc[int](nil) })
}

// ---------- Set ----------

func TestSetBasic(t *testing.T) {
	s := NewSet(5, 1, 3, 3, 1)
	if s.Len() != 3 {
		t.Fatalf("Len = %d", s.Len())
	}
	if got := s.String(); got != "Set{1, 3, 5}" {
		t.Fatalf("String = %q", got)
	}
	if got := s.ToSlice(); !slices.Equal(got, []int{1, 3, 5}) {
		t.Fatalf("ToSlice = %v", got)
	}
	if got := collect(s.Backward()); !slices.Equal(got, []int{5, 3, 1}) {
		t.Fatalf("Backward = %v", got)
	}
	if got := collect(s.Range(2, 5)); !slices.Equal(got, []int{3}) {
		t.Fatalf("Range = %v", got)
	}
	if v, ok := s.Floor(4); !ok || v != 3 {
		t.Fatalf("Floor(4) = %v %v", v, ok)
	}
	if v, ok := s.Floor(3); !ok || v != 3 {
		t.Fatalf("Floor(3) = %v %v", v, ok)
	}
	if _, ok := s.Floor(0); ok {
		t.Fatal("Floor(0) ok")
	}
	if v, ok := s.Ceiling(4); !ok || v != 5 {
		t.Fatalf("Ceiling(4) = %v %v", v, ok)
	}
	if _, ok := s.Ceiling(6); ok {
		t.Fatal("Ceiling(6) ok")
	}
	if v, ok := s.Lower(3); !ok || v != 1 {
		t.Fatalf("Lower(3) = %v %v", v, ok)
	}
	if v, ok := s.Higher(3); !ok || v != 5 {
		t.Fatalf("Higher(3) = %v %v", v, ok)
	}
	if s.Rank(3) != 1 || s.Rank(100) != 3 || s.Rank(-1) != 0 {
		t.Fatal("Rank wrong")
	}
	if s.At(2) != 5 {
		t.Fatal("At(2) wrong")
	}
	if s.Remove(4) || !s.Remove(3) || s.Contains(3) {
		t.Fatal("Remove wrong")
	}
	if v, ok := s.PopMin(); !ok || v != 1 {
		t.Fatalf("PopMin = %v %v", v, ok)
	}
	if v, ok := s.PopMax(); !ok || v != 5 {
		t.Fatalf("PopMax = %v %v", v, ok)
	}
	if !s.IsEmpty() {
		t.Fatal("not empty")
	}
	if _, ok := s.PopMin(); ok {
		t.Fatal("PopMin on empty ok")
	}
	if _, ok := s.PopMax(); ok {
		t.Fatal("PopMax on empty ok")
	}
	if _, ok := s.Min(); ok {
		t.Fatal("Min on empty ok")
	}
	if _, ok := s.Max(); ok {
		t.Fatal("Max on empty ok")
	}
	s.Add(-2, 7)
	s.Clear()
	if s.Len() != 0 || s.String() != "Set{}" {
		t.Fatal("Clear failed")
	}
}

func TestSetFuncByField(t *testing.T) {
	type person struct {
		name string
		age  int
	}
	s := NewSetFunc(func(a, b person) int { return cmp.Compare(a.age, b.age) },
		person{"ann", 30}, person{"bob", 25}, person{"cat", 30})
	// cat has the same age as ann, so it is a duplicate and is not added.
	if s.Len() != 2 {
		t.Fatalf("Len = %d", s.Len())
	}
	if p, _ := s.Max(); p.name != "ann" {
		t.Fatalf("Max = %v", p)
	}
}

// ---------- List ----------

func TestListBasic(t *testing.T) {
	l := NewList(3, 1, 2, 3, 3, -1)
	if got := l.ToSlice(); !slices.Equal(got, []int{-1, 1, 2, 3, 3, 3}) {
		t.Fatalf("ToSlice = %v", got)
	}
	if got := l.String(); got != "List{-1, 1, 2, 3, 3, 3}" {
		t.Fatalf("String = %q", got)
	}
	if l.Count(3) != 3 || l.Count(0) != 0 || l.Count(-1) != 1 {
		t.Fatal("Count wrong")
	}
	if l.BisectLeft(3) != 3 || l.BisectRight(3) != 6 || l.BisectLeft(0) != 1 || l.BisectRight(0) != 1 {
		t.Fatal("Bisect wrong")
	}
	if !l.Contains(2) || l.Contains(0) {
		t.Fatal("Contains wrong")
	}
	if !l.Remove(3) || l.Count(3) != 2 || l.Len() != 5 {
		t.Fatal("Remove wrong")
	}
	if l.Remove(42) {
		t.Fatal("Remove(42) true")
	}
	if n := l.RemoveAll(3); n != 2 || l.Contains(3) {
		t.Fatalf("RemoveAll = %d", n)
	}
	if n := l.RemoveAll(3); n != 0 {
		t.Fatalf("RemoveAll again = %d", n)
	}
	if got := collect(l.Backward()); !slices.Equal(got, []int{2, 1, -1}) {
		t.Fatalf("Backward = %v", got)
	}
	if v, ok := l.Floor(0); !ok || v != -1 {
		t.Fatalf("Floor(0) = %v %v", v, ok)
	}
	if v, ok := l.Ceiling(0); !ok || v != 1 {
		t.Fatalf("Ceiling(0) = %v %v", v, ok)
	}
	if v, ok := l.Lower(1); !ok || v != -1 {
		t.Fatalf("Lower(1) = %v %v", v, ok)
	}
	if v, ok := l.Higher(1); !ok || v != 2 {
		t.Fatalf("Higher(1) = %v %v", v, ok)
	}
	if v, _ := l.PopMax(); v != 2 {
		t.Fatalf("PopMax = %v", v)
	}
	if v, _ := l.PopMin(); v != -1 {
		t.Fatalf("PopMin = %v", v)
	}
	l.Clear()
	if !l.IsEmpty() {
		t.Fatal("Clear failed")
	}
	if _, ok := l.Min(); ok {
		t.Fatal("Min on empty ok")
	}
	if _, ok := l.PopMax(); ok {
		t.Fatal("PopMax on empty ok")
	}
}

func TestListStableAndRemoveFirst(t *testing.T) {
	type item struct {
		key int
		id  string
	}
	l := NewListFunc(func(a, b item) int { return cmp.Compare(a.key, b.key) })
	l.Add(item{2, "a"}, item{1, "b"}, item{2, "c"}, item{2, "d"}, item{3, "e"})
	var ids []string
	for it := range l.All() {
		ids = append(ids, it.id)
	}
	if got := strings.Join(ids, ""); got != "bacde" {
		t.Fatalf("order = %q, want bacde (equal keys in insertion order)", got)
	}
	if v, _ := l.Floor(item{key: 2}); v.id != "d" {
		t.Fatalf("Floor = %v, want last equal", v)
	}
	if v, _ := l.Ceiling(item{key: 2}); v.id != "a" {
		t.Fatalf("Ceiling = %v, want first equal", v)
	}
	l.Remove(item{key: 2})
	ids = ids[:0]
	for it := range l.All() {
		ids = append(ids, it.id)
	}
	if got := strings.Join(ids, ""); got != "bcde" {
		t.Fatalf("after Remove = %q, want bcde (earliest equal removed)", got)
	}
}

func TestSlidingWindowMedian(t *testing.T) {
	nums := []int{1, 3, -1, -3, 5, 3, 6, 7}
	k := 3
	want := []int{1, -1, -1, 3, 5, 6}
	l := NewList[int]()
	var got []int
	for i, x := range nums {
		l.Add(x)
		if i >= k {
			l.Remove(nums[i-k])
		}
		if i >= k-1 {
			got = append(got, l.At(k/2))
		}
	}
	if !slices.Equal(got, want) {
		t.Fatalf("medians = %v, want %v", got, want)
	}
}

func TestDeterministic(t *testing.T) {
	shape := func() string {
		s := NewSet[int]()
		for i := range 200 {
			s.Add((i * 7919) % 211)
		}
		var sb strings.Builder
		var walk func(n *node[int, struct{}])
		walk = func(n *node[int, struct{}]) {
			if n == nil {
				sb.WriteByte('.')
				return
			}
			fmt.Fprint(&sb, n.key, "(")
			walk(n.left)
			walk(n.right)
			sb.WriteByte(')')
		}
		walk(s.t.root)
		return sb.String()
	}
	if a, b := shape(), shape(); a != b {
		t.Fatal("tree shape differs between identical runs")
	}
}

func TestLargeSequentialHeight(t *testing.T) {
	// Sorted insertion must not degenerate into a linked list.
	s := NewSet[int]()
	const n = 100000
	for i := range n {
		s.Add(i)
	}
	var height func(n *node[int, struct{}]) int
	height = func(n *node[int, struct{}]) int {
		if n == nil {
			return 0
		}
		return 1 + max(height(n.left), height(n.right))
	}
	if h := height(s.t.root); h > 60 {
		t.Fatalf("height = %d for %d sorted inserts", h, n)
	}
	checkInvariants(t, &s.t)
}

// ---------- Randomized tests vs sorted-slice references ----------

// refList is a sorted slice with duplicates allowed.
type refList []int

func (r refList) bisectLeft(v int) int {
	i, _ := slices.BinarySearch(r, v)
	return i
}

func (r refList) bisectRight(v int) int {
	return r.bisectLeft(v + 1)
}

func TestListRandomized(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	l := NewList[int]()
	var ref refList
	for op := range 20000 {
		v := rng.IntN(200) - 100
		switch rng.IntN(12) {
		case 0, 1, 2, 3:
			l.Add(v)
			i := ref.bisectRight(v)
			ref = slices.Insert(ref, i, v)
		case 4:
			got := l.Remove(v)
			i := ref.bisectLeft(v)
			want := i < len(ref) && ref[i] == v
			if want {
				ref = slices.Delete(ref, i, i+1)
			}
			if got != want {
				t.Fatalf("op %d: Remove(%d) = %v, want %v", op, v, got, want)
			}
		case 5:
			if rng.IntN(10) == 0 {
				got := l.RemoveAll(v)
				i, j := ref.bisectLeft(v), ref.bisectRight(v)
				ref = slices.Delete(ref, i, j)
				if got != j-i {
					t.Fatalf("op %d: RemoveAll(%d) = %d, want %d", op, v, got, j-i)
				}
			}
		case 6:
			if got, want := l.BisectLeft(v), ref.bisectLeft(v); got != want {
				t.Fatalf("op %d: BisectLeft(%d) = %d, want %d", op, v, got, want)
			}
			if got, want := l.BisectRight(v), ref.bisectRight(v); got != want {
				t.Fatalf("op %d: BisectRight(%d) = %d, want %d", op, v, got, want)
			}
			if got, want := l.Count(v), ref.bisectRight(v)-ref.bisectLeft(v); got != want {
				t.Fatalf("op %d: Count(%d) = %d, want %d", op, v, got, want)
			}
			if got, want := l.Contains(v), slices.Contains(ref, v); got != want {
				t.Fatalf("op %d: Contains(%d) = %v, want %v", op, v, got, want)
			}
		case 7:
			if len(ref) > 0 {
				i := rng.IntN(len(ref))
				if got := l.At(i); got != ref[i] {
					t.Fatalf("op %d: At(%d) = %d, want %d", op, i, got, ref[i])
				}
			}
		case 8:
			checkNeighbors(t, op, v, ref, l.Floor, l.Ceiling, l.Lower, l.Higher)
		case 9:
			got, ok := l.PopMin()
			if ok != (len(ref) > 0) || (ok && got != ref[0]) {
				t.Fatalf("op %d: PopMin = %d %v", op, got, ok)
			}
			if ok {
				ref = ref[1:]
			}
		case 10:
			got, ok := l.PopMax()
			if ok != (len(ref) > 0) || (ok && got != ref[len(ref)-1]) {
				t.Fatalf("op %d: PopMax = %d %v", op, got, ok)
			}
			if ok {
				ref = ref[:len(ref)-1]
			}
		case 11:
			mn, ok1 := l.Min()
			mx, ok2 := l.Max()
			if ok1 != (len(ref) > 0) || ok2 != ok1 || (ok1 && (mn != ref[0] || mx != ref[len(ref)-1])) {
				t.Fatalf("op %d: Min/Max = %d %v %d %v", op, mn, ok1, mx, ok2)
			}
		}
		if l.Len() != len(ref) {
			t.Fatalf("op %d: Len = %d, want %d", op, l.Len(), len(ref))
		}
		if op%1000 == 0 {
			checkInvariants(t, &l.t)
			if got := l.ToSlice(); !slices.Equal(got, []int(ref)) {
				t.Fatalf("op %d: ToSlice = %v, want %v", op, got, ref)
			}
			if got := collect(l.Backward()); !slices.Equal(got, reversed(ref)) {
				t.Fatalf("op %d: Backward mismatch", op)
			}
		}
	}
	checkInvariants(t, &l.t)
}

func checkNeighbors(t *testing.T, op, v int, ref []int,
	floor, ceiling, lower, higher func(int) (int, bool),
) {
	t.Helper()
	i, j := refList(ref).bisectLeft(v), refList(ref).bisectRight(v)
	type q struct {
		name string
		f    func(int) (int, bool)
		idx  int
	}
	for _, c := range []q{
		{"Floor", floor, j - 1},
		{"Ceiling", ceiling, i},
		{"Lower", lower, i - 1},
		{"Higher", higher, j},
	} {
		got, ok := c.f(v)
		wantOK := c.idx >= 0 && c.idx < len(ref)
		if ok != wantOK || (ok && got != ref[c.idx]) {
			t.Fatalf("op %d: %s(%d) = %d %v, want ok=%v", op, c.name, v, got, ok, wantOK)
		}
	}
}

func TestSetAndMapRandomized(t *testing.T) {
	rng := rand.New(rand.NewPCG(3, 4))
	s := NewSet[int]()
	m := NewMap[int, int]()
	var ref []int // sorted unique keys
	vals := map[int]int{}
	for op := range 20000 {
		v := rng.IntN(300) - 150
		i, found := slices.BinarySearch(ref, v)
		switch rng.IntN(10) {
		case 0, 1, 2, 3:
			s.Add(v)
			m.Put(v, op)
			vals[v] = op
			if !found {
				ref = slices.Insert(ref, i, v)
			}
		case 4, 5:
			gotS, gotM := s.Remove(v), m.Delete(v)
			if gotS != found || gotM != found {
				t.Fatalf("op %d: Remove(%d) = %v/%v, want %v", op, v, gotS, gotM, found)
			}
			if found {
				ref = slices.Delete(ref, i, i+1)
				delete(vals, v)
			}
		case 6:
			if s.Rank(v) != i || m.Rank(v) != i {
				t.Fatalf("op %d: Rank(%d) = %d/%d, want %d", op, v, s.Rank(v), m.Rank(v), i)
			}
			if s.Contains(v) != found || m.Contains(v) != found {
				t.Fatalf("op %d: Contains(%d) wrong", op, v)
			}
			got, ok := m.Get(v)
			if ok != found || (ok && got != vals[v]) {
				t.Fatalf("op %d: Get(%d) = %d %v", op, v, got, ok)
			}
		case 7:
			checkNeighbors(t, op, v, ref, s.Floor, s.Ceiling, s.Lower, s.Higher)
			mapAdapt := func(f func(int) (int, int, bool)) func(int) (int, bool) {
				return func(k int) (int, bool) {
					key, val, ok := f(k)
					if ok && val != vals[key] {
						t.Fatalf("op %d: value for %d = %d, want %d", op, key, val, vals[key])
					}
					return key, ok
				}
			}
			checkNeighbors(t, op, v, ref, mapAdapt(m.Floor), mapAdapt(m.Ceiling), mapAdapt(m.Lower), mapAdapt(m.Higher))
		case 8:
			if len(ref) > 0 {
				j := rng.IntN(len(ref))
				k, val := m.At(j)
				if s.At(j) != ref[j] || k != ref[j] || val != vals[k] {
					t.Fatalf("op %d: At(%d) wrong", op, j)
				}
			}
		case 9:
			lo := rng.IntN(300) - 150
			hi := lo + rng.IntN(60)
			a, b := refList(ref).bisectLeft(lo), refList(ref).bisectLeft(hi)
			want := ref[a:b]
			gotS := collect(s.Range(lo, hi))
			gotM, _ := collect2(m.Range(lo, hi))
			if !slices.Equal(gotS, want) {
				t.Fatalf("op %d: Set.Range(%d, %d) = %v, want %v", op, lo, hi, gotS, want)
			}
			if !slices.Equal(gotM, want) {
				t.Fatalf("op %d: Map.Range(%d, %d) = %v, want %v", op, lo, hi, gotM, want)
			}
		}
		if s.Len() != len(ref) || m.Len() != len(ref) {
			t.Fatalf("op %d: Len = %d/%d, want %d", op, s.Len(), m.Len(), len(ref))
		}
		if op%1000 == 0 {
			checkInvariants(t, &s.t)
			checkInvariants(t, &m.t)
			if !slices.Equal(s.ToSlice(), ref) || !slices.Equal(m.Keys(), ref) {
				t.Fatalf("op %d: contents mismatch", op)
			}
			ks, _ := collect2(m.Backward())
			if !slices.Equal(ks, reversed(ref)) || !slices.Equal(collect(s.Backward()), reversed(ref)) {
				t.Fatalf("op %d: Backward mismatch", op)
			}
		}
	}
	// Drain with PopMin / PopMax alternately.
	for len(ref) > 0 {
		if len(ref)%2 == 0 {
			v, _ := s.PopMin()
			k, _, _ := m.PopMin()
			if v != ref[0] || k != ref[0] {
				t.Fatalf("PopMin = %d/%d, want %d", v, k, ref[0])
			}
			ref = ref[1:]
		} else {
			v, _ := s.PopMax()
			k, _, _ := m.PopMax()
			if v != ref[len(ref)-1] || k != ref[len(ref)-1] {
				t.Fatalf("PopMax = %d/%d, want %d", v, k, ref[len(ref)-1])
			}
			ref = ref[:len(ref)-1]
		}
	}
	if !s.IsEmpty() || !m.IsEmpty() {
		t.Fatal("not empty after drain")
	}
}
