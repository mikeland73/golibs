package linkedlist

import (
	"slices"
	"strings"
	"testing"
)

func TestNewToSliceLen(t *testing.T) {
	tests := [][]int{nil, {1}, {1, 2}, {5, 5, 5}, {-1, 0, 1, 2, 3, 4, 5, 6, 7}}
	for _, vals := range tests {
		head := New(vals...)
		if len(vals) == 0 {
			if head != nil {
				t.Errorf("New() = %v, want nil", head)
			}
		}
		if got := ToSlice(head); !slices.Equal(got, vals) {
			t.Errorf("ToSlice(New(%v)) = %v", vals, got)
		}
		if got := Len(head); got != len(vals) {
			t.Errorf("Len = %d, want %d", got, len(vals))
		}
		if got := slices.Collect(All(head)); !slices.Equal(got, vals) {
			t.Errorf("All = %v, want %v", got, vals)
		}
	}
}

func TestNewIsMutable(t *testing.T) {
	head := New("a", "b", "c")
	// Reverse in place.
	var prev *Node[string]
	for cur := head; cur != nil; {
		next := cur.Next
		cur.Next = prev
		prev, cur = cur, next
	}
	if got := ToSlice(prev); !slices.Equal(got, []string{"c", "b", "a"}) {
		t.Errorf("reversed = %v", got)
	}
}

func TestWithCycle(t *testing.T) {
	head := WithCycle(1, 3, 2, 0, -4)
	if NodeAt(head, 3).Next != NodeAt(head, 1) {
		t.Errorf("tail does not link to index 1")
	}
	if got := head.String(); got != "3 -> 2 -> 0 -> -4 -> ... (cycle to index 1)" {
		t.Errorf("String = %q", got)
	}
	self := WithCycle(0, 7)
	if self.Next != self {
		t.Errorf("self cycle not created")
	}
	if got := self.String(); got != "7 -> ... (cycle to index 0)" {
		t.Errorf("String self = %q", got)
	}
	noCycle := WithCycle(-1, 1, 2)
	if got := noCycle.String(); got != "1 -> 2" {
		t.Errorf("String no cycle = %q", got)
	}
	if WithCycle[int](-1) != nil {
		t.Errorf("WithCycle(-1) with no values should be nil")
	}
	for _, pos := range []int{-2, 2, 5} {
		func() {
			defer func() {
				r := recover()
				msg, _ := r.(string)
				if !strings.Contains(msg, "linkedlist: WithCycle pos") {
					t.Errorf("WithCycle(%d) panic = %v", pos, r)
				}
			}()
			WithCycle(pos, 1, 2)
		}()
	}
}

func TestCyclePanics(t *testing.T) {
	for n := 1; n <= 12; n++ {
		vals := make([]int, n)
		for i := range vals {
			vals[i] = i
		}
		for pos := range n {
			head := WithCycle(pos, vals...)
			for name, f := range map[string]func(){
				"ToSlice": func() { ToSlice(head) },
				"Len":     func() { Len(head) },
			} {
				func() {
					defer func() {
						r := recover()
						msg, _ := r.(string)
						if msg != "linkedlist: "+name+" called on a list with a cycle" {
							t.Errorf("%s(n=%d,pos=%d) panic = %v", name, n, pos, r)
						}
					}()
					f()
				}()
			}
			// All with early break still works on cycles.
			var got []int
			for v := range All(head) {
				if len(got) == 2*n {
					break
				}
				got = append(got, v)
			}
			if len(got) != 2*n {
				t.Errorf("All on cycle yielded %d", len(got))
			}
		}
	}
}

func TestNodeAt(t *testing.T) {
	head := New(10, 20, 30)
	tests := []struct {
		i    int
		want *Node[int]
	}{
		{0, head}, {1, head.Next}, {2, head.Next.Next}, {3, nil}, {-1, nil}, {100, nil},
	}
	for _, tt := range tests {
		if got := NodeAt(head, tt.i); got != tt.want {
			t.Errorf("NodeAt(%d) = %v, want %v", tt.i, got, tt.want)
		}
	}
	if NodeAt[int](nil, 0) != nil {
		t.Errorf("NodeAt(nil, 0) != nil")
	}
	cyc := WithCycle(0, 1, 2)
	if NodeAt(cyc, 5) != cyc.Next {
		t.Errorf("NodeAt follows cycle incorrectly")
	}
}

func TestString(t *testing.T) {
	var empty *Node[int]
	if got := empty.String(); got != "nil" {
		t.Errorf("nil String = %q", got)
	}
	if got := New(1).String(); got != "1" {
		t.Errorf("single String = %q", got)
	}
	if got := New("a", "b").String(); got != "a -> b" {
		t.Errorf("String = %q", got)
	}
	// Starting in the middle of a list prints from there.
	head := New(1, 2, 3)
	if got := head.Next.String(); got != "2 -> 3" {
		t.Errorf("middle String = %q", got)
	}
}

func TestLongList(t *testing.T) {
	vals := make([]int, 100000)
	for i := range vals {
		vals[i] = i
	}
	head := New(vals...)
	if Len(head) != len(vals) {
		t.Fatalf("Len wrong")
	}
	if !slices.Equal(ToSlice(head), vals) {
		t.Fatalf("ToSlice wrong")
	}
}
