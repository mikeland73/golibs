package assert

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// capture switches to Log mode writing to a buffer for the duration of a test.
func capture(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	oldOut, oldMode, oldVerbose := Output, SetMode(Log), Verbose
	Output = &buf
	t.Cleanup(func() {
		Output = oldOut
		SetMode(oldMode)
		Verbose = oldVerbose
	})
	return &buf
}

func thisLine() int {
	_, _, line, _ := runtime.Caller(1)
	return line
}

type point struct{ X, Y int }

type tree struct {
	Val         int
	Left, Right *tree
	kids        []int
}

func TestEqual(t *testing.T) {
	buf := capture(t)
	type args struct {
		got, want any
		ok        bool
	}
	var nilSlice []int
	var nilMap map[string]int
	var nilPtr *point
	cases := []args{
		{1, 1, true},
		{1, 2, false},
		{"a", "a", true},
		{[]int{1, 2}, []int{1, 2}, true},
		{[]int{1, 2}, []int{2, 1}, false},
		{nilSlice, []int{}, true},
		{[]int{}, nilSlice, true},
		{nilMap, map[string]int{}, true},
		{map[string]int{"a": 1}, map[string]int{"a": 1}, true},
		{map[string]int{"a": 1}, map[string]int{"a": 2}, false},
		{map[string]int{"a": 1}, map[string]int{"b": 1}, false},
		{map[string][]int{"a": nil}, map[string][]int{"a": {}}, true},
		{[][]int{nil, {1}}, [][]int{{}, {1}}, true},
		{point{1, 2}, point{1, 2}, true},
		{&point{1, 2}, &point{1, 2}, true},
		{&point{1, 2}, &point{1, 3}, false},
		{nilPtr, &point{}, false},
		{tree{1, &tree{Val: 2}, nil, nil}, tree{1, &tree{Val: 2, kids: []int{}}, nil, []int{}}, true},
		{[]any{1, "a"}, []any{1, "a"}, true},
		{[]any{1}, []any{int64(1)}, false},
		{nil, nil, true},
		{math.NaN(), math.NaN(), false}, // like reflect.DeepEqual
		{[2]int{1, 2}, [2]int{1, 2}, true},
		{errors.New("x"), errors.New("x"), true}, // deep equality on *errorString
	}
	for i, c := range cases {
		if got := Equal(c.got, c.want); got != c.ok {
			t.Errorf("case %d: Equal(%#v, %#v) = %v, want %v", i, c.got, c.want, got, c.ok)
		}
		if got := NotEqual(c.got, c.want); got == c.ok {
			t.Errorf("case %d: NotEqual(%#v, %#v) = %v", i, c.got, c.want, got)
		}
	}
	if Failures() == 0 || buf.Len() == 0 {
		t.Error("expected failures to be logged")
	}
}

func TestEqualCycles(t *testing.T) {
	capture(t)
	a := &tree{Val: 1}
	a.Left = a
	b := &tree{Val: 1}
	b.Left = b
	if !Equal(a, b) {
		t.Error("equal cyclic structures")
	}
	c := &tree{Val: 2}
	c.Left = c
	if Equal(a, c) {
		t.Error("different cyclic structures")
	}
}

func TestEqualMessage(t *testing.T) {
	buf := capture(t)
	line := thisLine() + 1
	Equal([]int{1, 2}, []int{1, 3}, "case %d", 7)
	want := fmt.Sprintf("assert.Equal failed at assert_test.go:%d: case 7\n  got:  []int{1, 2}\n  want: []int{1, 3}\n", line)
	if buf.String() != want {
		t.Errorf("got:\n%s\nwant:\n%s", buf.String(), want)
	}

	buf.Reset()
	Equal(&point{1, 2}, &point{3, 4})
	if !strings.Contains(buf.String(), "got:  &assert.point{X:1, Y:2}") {
		t.Errorf("pointer formatting: %s", buf.String())
	}

	buf.Reset()
	Equal(1, 2, 42) // non-string single message
	if !strings.Contains(buf.String(), ": 42\n") {
		t.Errorf("non-string message: %s", buf.String())
	}

	buf.Reset()
	Equal(1, 2, 1, 2) // non-format multi message
	if !strings.Contains(buf.String(), ": 1 2\n") {
		t.Errorf("multi message: %s", buf.String())
	}
}

func TestPanicModeDefault(t *testing.T) {
	if SetMode(Panic) != Panic {
		t.Fatal("default mode should be Panic")
	}
	var msg any
	line := 0
	func() {
		defer func() { msg = recover() }()
		line = thisLine() + 1
		True(false, "should panic")
	}()
	s, ok := msg.(string)
	if !ok {
		t.Fatalf("expected string panic, got %#v", msg)
	}
	want := fmt.Sprintf("assert.True failed at assert_test.go:%d: should panic", line)
	if s != want {
		t.Errorf("got %q want %q", s, want)
	}
	// Passing assertions don't panic.
	if !True(true) {
		t.Error("True(true) should pass")
	}
}

func TestExitMode(t *testing.T) {
	buf := capture(t)
	SetMode(Exit)
	code := -1
	oldExit := exit
	exit = func(c int) { code = c }
	defer func() { exit = oldExit }()
	False(true, "boom")
	if code != 1 {
		t.Errorf("exit code %d", code)
	}
	if !strings.Contains(buf.String(), "assert.False failed") || !strings.Contains(buf.String(), "boom") {
		t.Errorf("output: %s", buf.String())
	}
}

func TestVerbose(t *testing.T) {
	buf := capture(t)
	Verbose = true
	line := thisLine() + 1
	Equal(1, 1, "ones")
	want := fmt.Sprintf("✓ Equal at assert_test.go:%d: ones\n", line)
	if buf.String() != want {
		t.Errorf("got %q want %q", buf.String(), want)
	}
	buf.Reset()
	Verbose = false
	Equal(1, 1)
	if buf.Len() != 0 {
		t.Errorf("non-verbose pass printed %q", buf.String())
	}
}

func TestFailuresCount(t *testing.T) {
	capture(t)
	before := Failures()
	True(false)
	False(true)
	True(true)
	if Failures()-before != 2 {
		t.Errorf("failures delta %d", Failures()-before)
	}
}

func TestBoolAsserts(t *testing.T) {
	capture(t)
	if !True(true) || True(false) || !False(false) || False(true) {
		t.Error("True/False")
	}
}

func TestNil(t *testing.T) {
	capture(t)
	var p *point
	var s []int
	var m map[int]int
	var f func()
	var ch chan int
	var err error
	var iface any = p // typed nil in interface
	for i, v := range []any{nil, p, s, m, f, ch, err, iface} {
		if !Nil(v) || NotNil(v) {
			t.Errorf("case %d: %#v should be nil", i, v)
		}
	}
	for i, v := range []any{0, "", []int{}, map[int]int{}, &point{}, errors.New("x"), point{}} {
		if Nil(v) || !NotNil(v) {
			t.Errorf("case %d: %#v should not be nil", i, v)
		}
	}
}

func TestErrors(t *testing.T) {
	buf := capture(t)
	e := errors.New("bad")
	if !NoError(nil) || NoError(e) {
		t.Error("NoError")
	}
	if !strings.Contains(buf.String(), "error: bad") {
		t.Errorf("NoError message: %s", buf.String())
	}
	if !Error(e) || Error(nil) {
		t.Error("Error")
	}
	wrapped := fmt.Errorf("open: %w", fs.ErrNotExist)
	if !ErrorIs(wrapped, fs.ErrNotExist) || ErrorIs(wrapped, fs.ErrPermission) || ErrorIs(nil, fs.ErrNotExist) {
		t.Error("ErrorIs")
	}
	if !ErrorIs(nil, nil) {
		t.Error("ErrorIs(nil, nil)")
	}
}

func TestPanics(t *testing.T) {
	capture(t)
	if !Panics(func() { panic("x") }) {
		t.Error("panic not detected")
	}
	if !Panics(func() { panic(nil) }) {
		t.Error("panic(nil) not detected")
	}
	if !Panics(func() {
		var s []int
		_ = s[len(s)]
	}) {
		t.Error("runtime panic not detected")
	}
	if Panics(func() {}) {
		t.Error("no panic reported as panic")
	}
}

func TestLen(t *testing.T) {
	buf := capture(t)
	ch := make(chan int, 3)
	ch <- 1
	arr := [4]int{}
	cases := []struct {
		v  any
		n  int
		ok bool
	}{
		{[]int{1, 2}, 2, true},
		{[]int(nil), 0, true},
		{"héllo", 6, true}, // bytes, like len()
		{map[int]int{1: 1}, 1, true},
		{[3]int{}, 3, true},
		{&arr, 4, true},
		{ch, 1, true},
		{[]int{1}, 2, false},
		{5, 0, false},
		{nil, 0, false},
		{&point{}, 0, false},
	}
	for i, c := range cases {
		if got := Len(c.v, c.n); got != c.ok {
			t.Errorf("case %d: Len(%#v, %d) = %v", i, c.v, c.n, got)
		}
	}
	if !strings.Contains(buf.String(), "has no length") {
		t.Errorf("message: %s", buf.String())
	}
}

func TestElementsMatch(t *testing.T) {
	buf := capture(t)
	cases := []struct {
		got, want []int
		ok        bool
	}{
		{nil, nil, true},
		{nil, []int{}, true},
		{[]int{1, 2, 3}, []int{3, 1, 2}, true},
		{[]int{1, 1, 2}, []int{1, 2, 1}, true},
		{[]int{1, 1, 2}, []int{1, 2, 2}, false},
		{[]int{1, 2}, []int{1, 2, 3}, false},
		{[]int{1}, nil, false},
	}
	for i, c := range cases {
		if got := ElementsMatch(c.got, c.want); got != c.ok {
			t.Errorf("case %d: ElementsMatch(%v, %v) = %v", i, c.got, c.want, got)
		}
	}
	buf.Reset()
	ElementsMatch([]string{"a", "a", "b"}, []string{"a", "c"})
	out := buf.String()
	if !strings.Contains(out, `extra (in got, not in want):   []string{"a", "b"}`) ||
		!strings.Contains(out, `missing (in want, not in got): []string{"c"}`) {
		t.Errorf("message:\n%s", out)
	}
}

func TestInDelta(t *testing.T) {
	capture(t)
	cases := []struct {
		got, want, delta float64
		ok               bool
	}{
		{1.0, 1.05, 0.1, true},
		{1.0, 1.2, 0.1, false},
		{-1, 1, 2, true},
		{0.1 + 0.2, 0.3, 1e-9, true},
		{math.NaN(), math.NaN(), 0, true},
		{math.NaN(), 1, 10, false},
		{1, math.NaN(), 10, false},
		{math.Inf(1), math.Inf(1), 0, true},
		{math.Inf(1), math.Inf(-1), 1, false},
	}
	for i, c := range cases {
		if got := InDelta(c.got, c.want, c.delta); got != c.ok {
			t.Errorf("case %d: InDelta(%v, %v, %v) = %v", i, c.got, c.want, c.delta, got)
		}
	}
}

func TestModeString(t *testing.T) {
	if Panic.String() != "Panic" || Log.String() != "Log" || Exit.String() != "Exit" || Mode(9).String() != "Mode(9)" {
		t.Error("Mode.String")
	}
}

func TestConcurrent(t *testing.T) {
	buf := capture(t)
	before := Failures()
	var wg sync.WaitGroup
	for i := range 50 {
		wg.Go(func() {
			Equal(i, -1)
			True(true)
		})
	}
	wg.Wait()
	if Failures()-before != 50 {
		t.Errorf("failures delta %d", Failures()-before)
	}
	if n := strings.Count(buf.String(), "assert.Equal failed"); n != 50 {
		t.Errorf("logged %d failures", n)
	}
}

func TestEqualSubslicesSharingArray(t *testing.T) {
	capture(t)
	// Regression: the cycle memo was keyed on the data pointer only, so after
	// s[:1] matched t[:1], s and t were assumed equal.
	s, u := []int{1, 2}, []int{1, 3}
	type pair struct{ A, B []int }
	if Equal(pair{s[:1], s}, pair{u[:1], u}) {
		t.Error("struct of subslices: unequal values reported equal")
	}
	if Equal([][]int{s[:1], s}, [][]int{u[:1], u}) {
		t.Error("slice of subslices: unequal values reported equal")
	}
	if !Equal([][]int{s[:1], s}, [][]int{{1}, {1, 2}}) {
		t.Error("equal subslices reported unequal")
	}
	// Self-referencing slices still terminate.
	a := make([]any, 1)
	a[0] = a
	b := make([]any, 1)
	b[0] = b
	if !Equal(a, b) {
		t.Error("self-referencing slices")
	}
}

func TestShowPointerCycle(t *testing.T) {
	buf := capture(t)
	y := new(any)
	*y = y
	z := new(any)
	*z = 3
	if Equal(y, z) {
		t.Error("expected failure")
	}
	out := buf.String()
	if !strings.Contains(out, "got:  &<cycle>") || !strings.Contains(out, "want: &3") {
		t.Errorf("message:\n%s", out)
	}
	p := new(any)
	q := new(any)
	*p, *q = q, p
	if got := show(p); got != "&&<cycle>" {
		t.Errorf("2-cycle: %s", got)
	}
}

func TestElementsMatchNaN(t *testing.T) {
	buf := capture(t)
	nan := math.NaN()
	cases := []struct {
		got, want []float64
		ok        bool
	}{
		{[]float64{nan}, []float64{nan, nan}, false},
		{[]float64{nan, nan}, []float64{nan}, false},
		{[]float64{nan}, []float64{nan}, true},
		{[]float64{1, nan, 2}, []float64{2, nan, 1}, true},
		{[]float64{nan}, []float64{1}, false},
		{[]float64{0}, []float64{math.Copysign(0, -1)}, true}, // 0 == -0
	}
	for i, c := range cases {
		if got := ElementsMatch(c.got, c.want); got != c.ok {
			t.Errorf("case %d: ElementsMatch(%v, %v) = %v", i, c.got, c.want, got)
		}
	}
	type pt struct{ X, Y float64 }
	if !ElementsMatch([]pt{{nan, 1}, {2, 3}}, []pt{{2, 3}, {nan, 1}}) {
		t.Error("structs with NaN")
	}
	if !ElementsMatch([]any{nan, 1}, []any{1, nan}) {
		t.Error("interfaces holding NaN")
	}
	buf.Reset()
	ElementsMatch([]float64{nan}, []float64{nan, nan})
	if !strings.Contains(buf.String(), "missing (in want, not in got): []float64{NaN}") {
		t.Errorf("message:\n%s", buf.String())
	}
}

func TestElementsMatchUnhashable(t *testing.T) {
	buf := capture(t)
	x := 1
	cases := []struct {
		got, want []any
		ok        bool
	}{
		{[]any{[]int{1}, map[string]int{"a": 1}, 3}, []any{3, map[string]int{"a": 1}, []int{1}}, true},
		{[]any{[]int{1}, []int{1}}, []any{[]int{1}, []int{2}}, false},
		{[]any{[]int{1}}, []any{[]int{1}, []int{1}}, false},
		{[]any{[]int(nil)}, []any{[]int{}}, true}, // like Equal
		{[]any{[]int{1}}, []any{1}, false},
		{[]any{&x, nil}, []any{nil, &x}, true},
		{[]any{struct{ A any }{[]int{1}}}, []any{struct{ A any }{[]int{1}}}, true},
	}
	for i, c := range cases {
		if got := ElementsMatch(c.got, c.want); got != c.ok {
			t.Errorf("case %d: ElementsMatch(%v, %v) = %v", i, c.got, c.want, got)
		}
	}
	buf.Reset()
	ElementsMatch([]any{[]int{1}, 2, []int{3}}, []any{[]int{3}, []int{4}})
	out := buf.String()
	if !strings.Contains(out, "extra (in got, not in want):   []interface {}{[]int{1}, 2}") ||
		!strings.Contains(out, "missing (in want, not in got): []interface {}{[]int{4}}") {
		t.Errorf("message:\n%s", out)
	}
	// Pointers are compared by address, as with ==, even when the element
	// also contains a NaN.
	type withPtr struct {
		P *int
		F float64
	}
	y := 1
	if ElementsMatch([]withPtr{{&x, math.NaN()}}, []withPtr{{&y, math.NaN()}}) {
		t.Error("distinct pointers matched")
	}
}

func TestShowContainerCycle(t *testing.T) {
	buf := capture(t)
	a := make([]any, 2)
	a[0], a[1] = 1, a
	m := map[string]any{}
	m["m"] = m
	if Equal[any](a, m) {
		t.Error("expected failure")
	}
	out := buf.String()
	if !strings.Contains(out, "got:  []interface {}{<cycle>}") ||
		!strings.Contains(out, "want: map[string]interface {}{<cycle>}") {
		t.Errorf("message:\n%s", out)
	}
	// Shared, non-cyclic sub-slices are printed normally.
	sub := []int{7}
	if got := show([][]int{sub, sub}); got != "[][]int{[]int{7}, []int{7}}" {
		t.Errorf("shared: %s", got)
	}
}
