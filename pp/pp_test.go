package pp

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

type point struct {
	X, Y int
}

type withHidden struct {
	Name  string
	count int
	tags  []string
}

type node struct {
	Val  int
	Next *node
}

type celsius float64

func (c celsius) String() string { return fmt.Sprintf("%.1f°C", float64(c)) }

type ptrStringer struct{ n int }

func (p *ptrStringer) String() string { return fmt.Sprintf("PS(%d)", p.n) }

type myErr struct{ code int }

func (e myErr) Error() string { return fmt.Sprintf("error %d", e.code) }

type both struct{}

func (both) Error() string  { return "as error" }
func (both) String() string { return "as string" }

type panicky struct{}

func (panicky) String() string { panic("boom") }

type color int32

func TestSprintBasics(t *testing.T) {
	var nilPtr *point
	var nilSlice []int
	var nilMap map[string]int
	var nilErr error
	str := "hi"
	tests := []struct {
		name string
		in   any
		want string
	}{
		{"nil", nil, "nil"},
		{"int", 42, "42"},
		{"negative", -7, "-7"},
		{"uint8", uint8(200), "200"},
		{"float", 3.5, "3.5"},
		{"float whole", 2.0, "2"},
		{"float32", float32(0.1), "0.1"},
		{"inf", math.Inf(1), "+Inf"},
		{"complex", complex(1, 2), "(1+2i)"},
		{"bool", true, "true"},
		{"string top-level", "hello world", "hello world"},
		{"string slice", []string{"a", "b c"}, `["a", "b c"]`},
		{"int slice", []int{1, 2, 3}, "[1, 2, 3]"},
		{"empty slice", []int{}, "[]"},
		{"nil slice", nilSlice, "[]"},
		{"array", [3]int{1, 2, 3}, "[1, 2, 3]"},
		{"nested slices", [][]int{{1, 2}, {}, {3}}, "[[1, 2], [], [3]]"},
		{"bytes", []byte("hi"), "[104, 105]"},
		{"runes", []rune("hé"), "['h', 'é']"},
		{"rune top-level is number", 'a', "97"},
		{"named int32 not rune", []color{1, 2}, "[1, 2]"},
		{"map sorted", map[string]int{"b": 2, "a": 1, "c": 3}, `{"a": 1, "b": 2, "c": 3}`},
		{"map int keys numeric", map[int]string{10: "x", 2: "y", -1: "z"}, `{-1: "z", 2: "y", 10: "x"}`},
		{"map rune keys", map[rune]int{'b': 1, 'a': 2}, "{'a': 2, 'b': 1}"},
		{"map bool keys", map[bool]int{true: 1, false: 0}, "{false: 0, true: 1}"},
		{"map float keys", map[float64]int{2.5: 1, -1: 2}, "{-1: 2, 2.5: 1}"},
		{"map struct keys", map[point]int{{2, 1}: 1, {1, 5}: 2, {1, 2}: 3}, "{point{X: 1, Y: 2}: 3, point{X: 1, Y: 5}: 2, point{X: 2, Y: 1}: 1}"},
		{"map array keys", map[[2]int]bool{{1, 2}: true, {0, 9}: false}, "{[0, 9]: false, [1, 2]: true}"},
		{"set", map[int]struct{}{3: {}, 1: {}, 2: {}}, "{1, 2, 3}"},
		{"empty map", map[string]int{}, "{}"},
		{"nil map", nilMap, "{}"},
		{"map of slices", map[string][]int{"x": {1}}, `{"x": [1]}`},
		{"interface keys", map[any]int{"b": 1, 2: 2, "a": 3}, `{2: 2, "a": 3, "b": 1}`},
		{"struct", point{1, 2}, "point{X: 1, Y: 2}"},
		{"struct pointer", &point{3, 4}, "point{X: 3, Y: 4}"},
		{"nil pointer", nilPtr, "nil"},
		{"unexported fields", withHidden{"a", 2, []string{"t"}}, `withHidden{Name: "a", count: 2, tags: ["t"]}`},
		{"anonymous struct", struct{ A string }{"x"}, `{A: "x"}`},
		{"empty struct", struct{}{}, "{}"},
		{"pointer to string", &str, "hi"},
		{"slice of pointers", []*point{{1, 2}, nil}, "[point{X: 1, Y: 2}, nil]"},
		{"any slice", []any{1, "a", nil, 2.5, []int{1}}, `[1, "a", nil, 2.5, [1]]`},
		{"linked list", &node{1, &node{2, nil}}, "node{Val: 1, Next: node{Val: 2, Next: nil}}"},
		{"stringer", celsius(21.5), "21.5°C"},
		{"stringer nested", []celsius{1, 2}, "[1.0°C, 2.0°C]"},
		{"pointer stringer via value", ptrStringer{3}, "PS(3)"},
		{"pointer stringer via pointer", &ptrStringer{4}, "PS(4)"},
		{"pointer stringer nested", []ptrStringer{{1}, {2}}, "[PS(1), PS(2)]"},
		{"pointer stringer in map value", map[string]ptrStringer{"k": {5}}, `{"k": PS(5)}`},
		{"error", myErr{404}, "error 404"},
		{"errors.New", errors.New("bad"), "bad"},
		{"nil error", nilErr, "nil"},
		{"error preferred over String", both{}, "as error"},
		{"duration", 1500 * time.Millisecond, "1.5s"},
		{"durations", []time.Duration{time.Second, time.Minute}, "[1s, 1m0s]"},
		{"panicking stringer", panicky{}, "<pp.panicky method panicked: boom>"},
		{"chan", make(chan int), "chan int"},
		{"func", func(int) string { return "" }, "func(int) string"},
		{"string with quotes nested", []string{`a"b`}, `["a\"b"]`},
	}
	for _, tt := range tests {
		if got := Sprint(tt.in); got != tt.want {
			t.Errorf("%s: Sprint(%#v) = %s, want %s", tt.name, tt.in, got, tt.want)
		}
	}
}

func TestUnexportedDuration(t *testing.T) {
	type s struct{ d time.Duration }
	if got := Sprint(s{2 * time.Second}); got != "s{d: 2s}" {
		t.Errorf("got %s", got)
	}
}

func TestUnexportedStringerIsStructural(t *testing.T) {
	type s struct{ c celsius }
	if got := Sprint(s{1}); got != "s{c: 1}" {
		t.Errorf("got %s", got)
	}
}

func TestSprintMultiple(t *testing.T) {
	if got := Sprint("x:", 1, []string{"a"}, nil); got != `x: 1 ["a"] nil` {
		t.Errorf("got %s", got)
	}
	if Sprint() != "" {
		t.Error("empty Sprint")
	}
}

func TestRepr(t *testing.T) {
	if got := Repr("hi"); got != `"hi"` {
		t.Errorf("got %s", got)
	}
	if got := Repr(5); got != "5" {
		t.Errorf("got %s", got)
	}
}

func TestCycles(t *testing.T) {
	n := &node{Val: 1}
	n.Next = n
	if got := Sprint(n); got != "node{Val: 1, Next: <cycle>}" {
		t.Errorf("pointer cycle: %s", got)
	}

	a := &node{Val: 1}
	b := &node{Val: 2, Next: a}
	a.Next = b
	if got := Sprint(a); got != "node{Val: 1, Next: node{Val: 2, Next: <cycle>}}" {
		t.Errorf("2-cycle: %s", got)
	}

	s := make([]any, 2)
	s[0] = 1
	s[1] = s
	if got := Sprint(s); got != "[1, <cycle>]" {
		t.Errorf("slice cycle: %s", got)
	}

	m := map[string]any{"a": 1}
	m["self"] = m
	if got := Sprint(m); got != `{"a": 1, "self": <cycle>}` {
		t.Errorf("map cycle: %s", got)
	}

	// Shared (non-cyclic) references are printed in full each time.
	shared := &point{1, 2}
	if got := Sprint([]*point{shared, shared}); got != "[point{X: 1, Y: 2}, point{X: 1, Y: 2}]" {
		t.Errorf("shared: %s", got)
	}
	sub := []int{7}
	if got := Sprint([][]int{sub, sub}); got != "[[7], [7]]" {
		t.Errorf("shared slice: %s", got)
	}
}

func TestDebug(t *testing.T) {
	var buf bytes.Buffer
	old := Output
	Output = &buf
	defer func() { Output = old }()

	x := 5
	xs := []int{1, 2}
	name := "bob"
	_, _, line, _ := runtime.Caller(0)
	Debug(x, xs, name) // line+1
	Debug(x+1, len(xs), "label", 42, -1)
	Debug()
	Debug(map[string]int{"a": 1})

	want := []string{
		fmt.Sprintf(`[pp_test.go:%d] x = 5, xs = [1, 2], name = "bob"`, line+1),
		fmt.Sprintf(`[pp_test.go:%d] x+1 = 6, len(xs) = 2, label, 42, -1`, line+2),
		fmt.Sprintf(`[pp_test.go:%d]`, line+3),
		fmt.Sprintf(`[pp_test.go:%d] map[string]int{"a": 1} = {"a": 1}`, line+4),
	}
	got := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("got:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestDebugMultiline(t *testing.T) {
	var buf bytes.Buffer
	old := Output
	Output = &buf
	defer func() { Output = old }()

	a, b := 1, 2
	Debug(
		a,
		b,
	)
	out := buf.String()
	if !strings.Contains(out, "a = 1, b = 2") {
		t.Errorf("got %q", out)
	}
}

func TestDebugFallback(t *testing.T) {
	var buf bytes.Buffer
	old := Output
	Output = &buf
	defer func() { Output = old }()

	vals := []any{5, []int{1, 2}, "s"}
	_, _, line, _ := runtime.Caller(0)
	Debug(vals...) // spread: no per-arg expressions
	dbg := Debug
	dbg(7) // not recognisable as a Debug call

	want := fmt.Sprintf("[pp_test.go:%d] 5, [1, 2], \"s\"\n[pp_test.go:%d] 7\n", line+1, line+3)
	if buf.String() != want {
		t.Errorf("got %q want %q", buf.String(), want)
	}
}

func TestDebugUnreadableSource(t *testing.T) {
	if s, exprs := debugArgs("/nonexistent/file.go", 1, 1, true); s != nil || exprs != nil {
		t.Error("expected nil for missing file")
	}
	// Cached negative result.
	if s, _ := debugArgs("/nonexistent/file.go", 1, 1, true); s != nil {
		t.Error("expected nil from cache")
	}
}

func TestDebugConcurrent(t *testing.T) {
	var buf bytes.Buffer
	old := Output
	Output = &buf
	defer func() { Output = old }()

	var wg sync.WaitGroup
	for i := range 20 {
		wg.Go(func() { Debug(i) })
	}
	wg.Wait()
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 20 {
		t.Fatalf("got %d lines", len(lines))
	}
	for _, l := range lines {
		if !strings.Contains(l, "] i = ") {
			t.Errorf("bad line %q", l)
		}
	}
}

func TestSprintGrid(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{"ints aligned", [][]int{{1, 10, 100}, {1000, 2, 3}}, "   1 10 100\n1000  2   3"},
		{"bytes", [][]byte{[]byte("#.#"), []byte(".#.")}, "#.#\n.#."},
		{"runes", [][]rune{[]rune("ab"), []rune("cé")}, "ab\ncé"},
		{"strings rows", []string{"ab", "cd"}, "ab\ncd"},
		{"string cells", [][]string{{"a", "bbb"}, {"cc", "d"}}, " a bbb\ncc   d"},
		{"array", [2][2]int{{1, 2}, {3, 4}}, "1 2\n3 4"},
		{"pointer to grid", &[][]int{{1}}, "1"},
		{"bools", [][]bool{{true, false}}, "true false"},
		{"ragged", [][]int{{1, 2, 3}, {4}}, "1 2 3\n4"},
		{"negative", [][]int{{-1, 2}, {3, -40}}, "-1   2\n 3 -40"},
		{"empty", [][]int{}, ""},
		{"not a grid", []int{1, 2}, "[1, 2]"},
		{"scalar", 5, "5"},
		{"nil", nil, "nil"},
	}
	for _, tt := range tests {
		if got := SprintGrid(tt.in); got != tt.want {
			t.Errorf("%s: got\n%q\nwant\n%q", tt.name, got, tt.want)
		}
	}
}

func TestSprintJSON(t *testing.T) {
	type resp struct {
		Name string   `json:"name"`
		Tags []string `json:"tags"`
		HTML string   `json:"html"`
	}
	got := SprintJSON(resp{"x", []string{"a"}, "<b>"})
	want := "{\n  \"name\": \"x\",\n  \"tags\": [\n    \"a\"\n  ],\n  \"html\": \"<b>\"\n}"
	if got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
	if got := SprintJSON(make(chan int)); !strings.HasPrefix(got, "<json error: ") {
		t.Errorf("error case: %s", got)
	}
	if got := SprintJSON(map[int]string{2: "b", 1: "a"}); got != "{\n  \"1\": \"a\",\n  \"2\": \"b\"\n}" {
		t.Errorf("map: %s", got)
	}
}

func TestMapNaNKeys(t *testing.T) {
	nan := math.NaN()
	if got := Sprint(map[float64]int{nan: 1, 2: 3}); got != "{NaN: 1, 2: 3}" {
		t.Errorf("got %s", got)
	}
	m := map[float64]string{nan: "b", 1: "x"}
	m[nan] = "a" // a second, distinct NaN key
	if got := Sprint(m); got != `{NaN: "a", NaN: "b", 1: "x"}` {
		t.Errorf("got %s", got)
	}
}

func TestNilPointerInInterfaceWithValueMethod(t *testing.T) {
	type wrap struct{ Err error }
	var e *myErr
	if got := Sprint(wrap{Err: e}); got != "wrap{Err: nil}" {
		t.Errorf("got %s", got)
	}
	var c *celsius
	if got := Sprint([]fmt.Stringer{c, celsius(1)}); got != "[nil, 1.0°C]" {
		t.Errorf("got %s", got)
	}
	if got := Sprint(map[string]error{"k": e}); got != `{"k": nil}` {
		t.Errorf("got %s", got)
	}
}

type debugLogger struct{}

func (debugLogger) Debug(...any) {}

func TestDebugPicksRightCall(t *testing.T) {
	var buf bytes.Buffer
	old := Output
	Output = &buf
	defer func() { Output = old }()

	a, b := 1, 2
	id := func(x int) int { return x }
	var log debugLogger
	_, _, line, _ := runtime.Caller(0)
	func() { Debug(a); Debug(b) }()                // two calls on one line: ambiguous
	func() { log.Debug(a); Debug(b) }()            // only Debug from this package counts
	Debug(id(func() int { Debug(b); return a }())) // nested: ambiguous
	func() { Debug(a, b); Debug(b) }()             // different arg counts: unambiguous

	want := []string{
		fmt.Sprintf("[pp_test.go:%d] 1", line+1),
		fmt.Sprintf("[pp_test.go:%d] 2", line+1),
		fmt.Sprintf("[pp_test.go:%d] b = 2", line+2),
		fmt.Sprintf("[pp_test.go:%d] 2", line+3),
		fmt.Sprintf("[pp_test.go:%d] 1", line+3),
		fmt.Sprintf("[pp_test.go:%d] a = 1, b = 2", line+4),
		fmt.Sprintf("[pp_test.go:%d] b = 2", line+4),
	}
	got := strings.TrimSuffix(buf.String(), "\n")
	if got != strings.Join(want, "\n") {
		t.Errorf("got:\n%s\nwant:\n%s", got, strings.Join(want, "\n"))
	}
}

func TestDebugArgsImports(t *testing.T) {
	const call = "\tx.Debug(a); y.Debug(a); Debug(a)\n"
	tests := []struct {
		name    string
		imports string
		samePkg bool
		want    string // text of the matched argument's call, "" for none
	}{
		{"default name", `import "` + ownPkgPath + `"`, false, ""},
		{"alias x", `import x "` + ownPkgPath + `"`, false, "x"},
		{"alias y", `import y "` + ownPkgPath + `"`, false, "y"},
		{"dot import", `import . "` + ownPkgPath + `"`, false, "bare"},
		{"same package", ``, true, "bare"},
		{"other package", `import x "example.com/other"`, false, ""},
		{"blank import", `import _ "` + ownPkgPath + `"`, false, ""},
	}
	for _, tt := range tests {
		dir := t.TempDir()
		file := dir + "/f.go"
		src := "package p\n" + tt.imports + "\nfunc f() {\n\ta := 1\n" + call + "}\n"
		if err := os.WriteFile(file, []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
		s, exprs := debugArgs(file, 5, 1, tt.samePkg)
		var got string
		if exprs != nil {
			got = "bare"
			for _, c := range []string{"x", "y"} {
				if strings.Index(src, c+".Debug(a)")+len(c+".Debug(") == s.fset.Position(exprs[0].Pos()).Offset {
					got = c
				}
			}
		}
		if got != tt.want {
			t.Errorf("%s: matched %q, want %q", tt.name, got, tt.want)
		}
	}
	// With the default import name, pp.Debug is recognized.
	dir := t.TempDir()
	file := dir + "/g.go"
	src := "package p\nimport \"" + ownPkgPath + "\"\nfunc f() {\n\ta := 1\n\tpp.Debug(a)\n}\n"
	if err := os.WriteFile(file, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, exprs := debugArgs(file, 5, 1, false); exprs == nil {
		t.Error("pp.Debug not recognized")
	}
}

func TestFuncPkgPath(t *testing.T) {
	pc, _, _, _ := runtime.Caller(0)
	if got := funcPkgPath(pc); got != ownPkgPath || ownFuncPkg != ownPkgPath {
		t.Errorf("funcPkgPath = %q, ownFuncPkg = %q, want %q", got, ownFuncPkg, ownPkgPath)
	}
	if funcPkgPath(0) != "" {
		t.Error("unknown pc")
	}
}
