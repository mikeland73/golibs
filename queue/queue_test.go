package queue

import (
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
		if s, ok := r.(string); !ok || !strings.Contains(s, want) {
			t.Fatalf("panic = %v, want containing %q", r, want)
		}
	}()
	f()
}

func TestZeroValue(t *testing.T) {
	var q Queue[int]
	if !q.IsEmpty() || q.Len() != 0 {
		t.Fatal("zero value not empty")
	}
	if _, ok := q.TryPop(); ok {
		t.Fatal("TryPop on empty ok")
	}
	if _, ok := q.TryPeek(); ok {
		t.Fatal("TryPeek on empty ok")
	}
	if q.String() != "Queue[]" || len(q.ToSlice()) != 0 {
		t.Fatal("empty String/ToSlice wrong")
	}
	for range q.All() {
		t.Fatal("All yielded on empty")
	}
	mustPanic(t, "queue: Pop called on empty Queue", func() { q.Pop() })
	mustPanic(t, "queue: Peek called on empty Queue", func() { q.Peek() })
	q.Push(1)
	if q.Peek() != 1 || q.Pop() != 1 || !q.IsEmpty() {
		t.Fatal("zero value push/pop failed")
	}
}

func TestFIFO(t *testing.T) {
	q := New(1, 2, 3)
	q.Push(4, 5)
	q.Push()
	if q.String() != "Queue[1 2 3 4 5]" {
		t.Fatalf("String = %q", q.String())
	}
	if v, ok := q.TryPeek(); !ok || v != 1 {
		t.Fatalf("TryPeek = %d,%v", v, ok)
	}
	var got []int
	for !q.IsEmpty() {
		got = append(got, q.Pop())
	}
	if !slices.Equal(got, []int{1, 2, 3, 4, 5}) {
		t.Fatalf("pop order %v", got)
	}
}

func TestWrapAround(t *testing.T) {
	var q Queue[int]
	q.Push(0, 1, 2, 3, 4, 5)
	for range 4 {
		q.Pop()
	}
	q.Push(6, 7, 8, 9, 10) // wraps in an 8-slot buffer
	want := []int{4, 5, 6, 7, 8, 9, 10}
	if got := q.ToSlice(); !slices.Equal(got, want) {
		t.Fatalf("ToSlice = %v", got)
	}
	if got := slices.Collect(q.All()); !slices.Equal(got, want) {
		t.Fatalf("All = %v", got)
	}
	q.Push(11, 12, 13) // grow while wrapped
	want = append(want, 11, 12, 13)
	if got := q.ToSlice(); !slices.Equal(got, want) {
		t.Fatalf("after grow ToSlice = %v", got)
	}
	n := 0
	for range q.All() {
		n++
		if n == 2 {
			break
		}
	}
	if n != 2 {
		t.Fatal("All did not stop early")
	}
}

func TestClear(t *testing.T) {
	q := New("a", "b")
	q.Clear()
	if !q.IsEmpty() || q.Len() != 0 {
		t.Fatal("Clear did not reset")
	}
	q.Push("c")
	if q.Pop() != "c" {
		t.Fatal("reuse after Clear failed")
	}
}

func TestRandomized(t *testing.T) {
	r := rand.New(rand.NewPCG(7, 8))
	var q Queue[int]
	var ref []int
	for i := range 20000 {
		switch r.IntN(5) {
		case 0, 1:
			k := r.IntN(4)
			vals := make([]int, k)
			for j := range vals {
				vals[j] = i*10 + j
			}
			q.Push(vals...)
			ref = append(ref, vals...)
		case 2, 3:
			v, ok := q.TryPop()
			if ok != (len(ref) > 0) || (ok && v != ref[0]) {
				t.Fatalf("step %d TryPop = %d,%v ref %v", i, v, ok, ref)
			}
			if ok {
				ref = ref[1:]
			}
		case 4:
			v, ok := q.TryPeek()
			if ok != (len(ref) > 0) || (ok && v != ref[0]) {
				t.Fatalf("step %d TryPeek = %d,%v", i, v, ok)
			}
		}
		if q.Len() != len(ref) {
			t.Fatalf("step %d Len = %d want %d", i, q.Len(), len(ref))
		}
		if i%500 == 0 && !slices.Equal(q.ToSlice(), ref) {
			t.Fatalf("step %d ToSlice = %v want %v", i, q.ToSlice(), ref)
		}
	}
}
