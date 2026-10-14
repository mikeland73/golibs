package deque

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

func values[T any](d *Deque[T]) []T {
	var out []T
	for _, v := range d.All() {
		out = append(out, v)
	}
	return out
}

func TestZeroValue(t *testing.T) {
	var d Deque[int]
	if !d.IsEmpty() || d.Len() != 0 || d.String() != "Deque[]" || len(d.ToSlice()) != 0 {
		t.Fatal("zero value not empty")
	}
	for _, f := range []func() (int, bool){d.TryPopBack, d.TryPopFront, d.TryFront, d.TryBack} {
		if v, ok := f(); ok || v != 0 {
			t.Fatalf("Try* on empty = %d,%v", v, ok)
		}
	}
	d.Rotate(3) // no-op, no panic
	mustPanic(t, "deque: PopBack called on empty Deque", func() { d.PopBack() })
	mustPanic(t, "deque: PopFront called on empty Deque", func() { d.PopFront() })
	mustPanic(t, "deque: Front called on empty Deque", func() { d.Front() })
	mustPanic(t, "deque: Back called on empty Deque", func() { d.Back() })
	mustPanic(t, "deque: At index 0 out of range [0, 0)", func() { d.At(0) })
	mustPanic(t, "deque: Set index -1 out of range", func() { d.Set(-1, 0) })
	d.PushBack(1)
	if d.Front() != 1 || d.Back() != 1 {
		t.Fatal("single element front/back")
	}
}

func TestBasics(t *testing.T) {
	d := New(2, 3)
	d.PushFront(1)
	d.PushBack(4, 5)
	if got := d.ToSlice(); !slices.Equal(got, []int{1, 2, 3, 4, 5}) {
		t.Fatalf("ToSlice = %v", got)
	}
	if d.Front() != 1 || d.Back() != 5 {
		t.Fatal("Front/Back wrong")
	}
	if v, ok := d.TryFront(); !ok || v != 1 {
		t.Fatal("TryFront wrong")
	}
	if v, ok := d.TryBack(); !ok || v != 5 {
		t.Fatal("TryBack wrong")
	}
	if d.At(2) != 3 {
		t.Fatal("At wrong")
	}
	d.Set(2, 30)
	if d.At(2) != 30 {
		t.Fatal("Set wrong")
	}
	mustPanic(t, "deque: At index 5 out of range [0, 5)", func() { d.At(5) })
	if d.PopFront() != 1 || d.PopBack() != 5 {
		t.Fatal("Pop wrong")
	}
	if d.String() != "Deque[2 30 4]" {
		t.Fatalf("String = %q", d.String())
	}
	d.Clear()
	if !d.IsEmpty() || len(d.buf) != 0 {
		t.Fatal("Clear failed")
	}
	d.PushFront(9)
	if d.Back() != 9 {
		t.Fatal("reuse after Clear")
	}
}

func TestPushFrontOrder(t *testing.T) {
	d := New(10)
	d.PushFront(1, 2, 3)
	if got := d.ToSlice(); !slices.Equal(got, []int{3, 2, 1, 10}) {
		t.Fatalf("PushFront order %v", got)
	}
}

func TestIterators(t *testing.T) {
	d := New("a", "b", "c")
	d.PushFront("z") // force head to wrap
	var idx []int
	var vals []string
	for i, v := range d.All() {
		idx = append(idx, i)
		vals = append(vals, v)
	}
	if !slices.Equal(idx, []int{0, 1, 2, 3}) || !slices.Equal(vals, []string{"z", "a", "b", "c"}) {
		t.Fatalf("All = %v %v", idx, vals)
	}
	idx, vals = nil, nil
	for i, v := range d.Backward() {
		idx = append(idx, i)
		vals = append(vals, v)
	}
	if !slices.Equal(idx, []int{3, 2, 1, 0}) || !slices.Equal(vals, []string{"c", "b", "a", "z"}) {
		t.Fatalf("Backward = %v %v", idx, vals)
	}
	for _, v := range d.All() {
		if v != "z" {
			t.Fatal("early break All")
		}
		break
	}
	for _, v := range d.Backward() {
		if v != "c" {
			t.Fatal("early break Backward")
		}
		break
	}
}

func refRotate(s []int, n int) []int {
	if len(s) == 0 {
		return s
	}
	k := ((n % len(s)) + len(s)) % len(s)
	return append(slices.Clone(s[len(s)-k:]), s[:len(s)-k]...)
}

func TestRotate(t *testing.T) {
	tests := []struct {
		n    int
		want []int
	}{
		{0, []int{1, 2, 3, 4, 5}},
		{1, []int{5, 1, 2, 3, 4}},
		{2, []int{4, 5, 1, 2, 3}},
		{4, []int{2, 3, 4, 5, 1}},
		{5, []int{1, 2, 3, 4, 5}},
		{7, []int{4, 5, 1, 2, 3}},
		{-1, []int{2, 3, 4, 5, 1}},
		{-7, []int{3, 4, 5, 1, 2}},
	}
	for _, tt := range tests {
		d := New(1, 2, 3, 4, 5)
		d.Rotate(tt.n)
		if got := d.ToSlice(); !slices.Equal(got, tt.want) {
			t.Errorf("Rotate(%d) = %v want %v", tt.n, got, tt.want)
		}
	}
	// Full buffer path.
	full := New(1, 2, 3, 4, 5, 6, 7, 8)
	if len(full.buf) != full.n {
		t.Fatalf("expected full buffer, cap %d len %d", len(full.buf), full.n)
	}
	full.Rotate(3)
	if got := full.ToSlice(); !slices.Equal(got, []int{6, 7, 8, 1, 2, 3, 4, 5}) {
		t.Fatalf("full Rotate = %v", got)
	}
	single := New(1)
	single.Rotate(-3)
	if single.Front() != 1 {
		t.Fatal("single rotate")
	}
}

func TestShrinks(t *testing.T) {
	var d Deque[int]
	for i := range 10000 {
		d.PushBack(i)
	}
	for range 9995 {
		d.PopFront()
	}
	if len(d.buf) > 64 {
		t.Fatalf("buffer did not shrink: %d", len(d.buf))
	}
	for i := range 100000 {
		d.PushFront(i)
		d.PopBack()
	}
	if len(d.buf) > 64 {
		t.Fatalf("buffer grew in steady state: %d", len(d.buf))
	}
}

func TestRandomized(t *testing.T) {
	r := rand.New(rand.NewPCG(9, 10))
	var d Deque[int]
	var ref []int
	for step := range 30000 {
		switch r.IntN(10) {
		case 0, 1:
			k := r.IntN(4)
			vals := make([]int, k)
			for j := range vals {
				vals[j] = r.IntN(1000)
			}
			d.PushBack(vals...)
			ref = append(ref, vals...)
		case 2, 3:
			k := r.IntN(4)
			vals := make([]int, k)
			for j := range vals {
				vals[j] = r.IntN(1000)
			}
			d.PushFront(vals...)
			for _, v := range vals {
				ref = slices.Insert(ref, 0, v)
			}
		case 4:
			v, ok := d.TryPopBack()
			if ok != (len(ref) > 0) || (ok && v != ref[len(ref)-1]) {
				t.Fatalf("step %d TryPopBack", step)
			}
			if ok {
				ref = ref[:len(ref)-1]
			}
		case 5:
			v, ok := d.TryPopFront()
			if ok != (len(ref) > 0) || (ok && v != ref[0]) {
				t.Fatalf("step %d TryPopFront", step)
			}
			if ok {
				ref = ref[1:]
			}
		case 6:
			n := r.IntN(21) - 10
			d.Rotate(n)
			ref = refRotate(ref, n)
		case 7:
			if len(ref) > 0 {
				i := r.IntN(len(ref))
				v := r.IntN(1000)
				d.Set(i, v)
				ref[i] = v
			}
		case 8:
			if len(ref) > 0 {
				i := r.IntN(len(ref))
				if d.At(i) != ref[i] {
					t.Fatalf("step %d At(%d)", step, i)
				}
			}
		case 9:
			f, okf := d.TryFront()
			b, okb := d.TryBack()
			if okf != (len(ref) > 0) || okb != okf || (okf && (f != ref[0] || b != ref[len(ref)-1])) {
				t.Fatalf("step %d TryFront/TryBack", step)
			}
		}
		if d.Len() != len(ref) {
			t.Fatalf("step %d Len %d want %d", step, d.Len(), len(ref))
		}
		if step%250 == 0 {
			if !slices.Equal(d.ToSlice(), ref) || !slices.Equal(values(&d), ref) {
				t.Fatalf("step %d contents %v want %v", step, d.ToSlice(), ref)
			}
		}
	}
}
