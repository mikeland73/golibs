package stack

import (
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
	var s Stack[int]
	if !s.IsEmpty() || s.Len() != 0 {
		t.Fatal("zero value not empty")
	}
	if _, ok := s.TryPop(); ok {
		t.Fatal("TryPop on empty ok")
	}
	if _, ok := s.TryPeek(); ok {
		t.Fatal("TryPeek on empty ok")
	}
	if got := s.String(); got != "Stack[]" {
		t.Fatalf("String = %q", got)
	}
	s.Push(1)
	if s.Pop() != 1 {
		t.Fatal("zero value push/pop failed")
	}
	mustPanic(t, "stack: Pop called on empty Stack", func() { s.Pop() })
	mustPanic(t, "stack: Peek called on empty Stack", func() { s.Peek() })
}

func TestPushPopOrder(t *testing.T) {
	s := New(1, 2)
	s.Push(3, 4)
	s.Push()
	if s.Len() != 4 {
		t.Fatalf("Len = %d", s.Len())
	}
	if v, ok := s.TryPeek(); !ok || v != 4 {
		t.Fatalf("TryPeek = %d,%v", v, ok)
	}
	if s.Peek() != 4 || s.Len() != 4 {
		t.Fatal("Peek wrong or modified")
	}
	var got []int
	for !s.IsEmpty() {
		got = append(got, s.Pop())
	}
	if !slices.Equal(got, []int{4, 3, 2, 1}) {
		t.Fatalf("pop order %v", got)
	}
	if v, ok := s.TryPop(); ok || v != 0 {
		t.Fatalf("TryPop on empty = %d,%v", v, ok)
	}
}

func TestNewCopies(t *testing.T) {
	in := []string{"a", "b"}
	s := New(in...)
	in[1] = "z"
	if s.Peek() != "b" {
		t.Fatal("New aliased input slice")
	}
	if New[int]().Len() != 0 {
		t.Fatal("New() not empty")
	}
}

func TestIterationAndString(t *testing.T) {
	s := New(1, 2, 3)
	if got := slices.Collect(s.All()); !slices.Equal(got, []int{3, 2, 1}) {
		t.Fatalf("All = %v", got)
	}
	for v := range s.All() {
		if v != 3 {
			t.Fatalf("first = %d", v)
		}
		break
	}
	ts := s.ToSlice()
	if !slices.Equal(ts, []int{1, 2, 3}) {
		t.Fatalf("ToSlice = %v", ts)
	}
	ts[2] = 99
	if s.Peek() != 3 {
		t.Fatal("ToSlice aliased storage")
	}
	if got := s.String(); got != "Stack[1 2 3]" {
		t.Fatalf("String = %q", got)
	}
	s.Clear()
	if !s.IsEmpty() {
		t.Fatal("Clear failed")
	}
	s.Push(7)
	if s.Peek() != 7 || s.Len() != 1 {
		t.Fatal("reuse after Clear failed")
	}
}
