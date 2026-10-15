// Package assert provides runtime assertions for quick programs and
// interview-style main() functions, where you want to check results without
// setting up a test file:
//
//	func main() {
//		assert.Equal(twoSum([]int{2, 7, 11}, 9), []int{0, 1})
//		assert.True(isPalindrome("racecar"), "racecar")
//		assert.ElementsMatch(permutations("ab"), []string{"ab", "ba"})
//	}
//
// A failed assertion reports the caller's file:line, the assertion name, the
// got/want values and an optional message:
//
//	assert.Equal failed at main.go:12: twoSum
//	  got:  []int{1, 0}
//	  want: []int{0, 1}
//
// By default a failure panics (like Python's assert statement). SetMode(Log)
// prints failures to Output and continues, so you can see every failing case
// in one run; SetMode(Exit) prints and exits with status 1. Every assertion
// returns true if it passed, so it can also be used in conditions.
//
// The optional msgAndArgs arguments form a message: a single value is printed
// as is, and if the first value is a format string the rest are its
// arguments: assert.True(ok, "case %d: %s", i, name).
//
// The mode and failure counter are safe for concurrent use. Output and
// Verbose are plain variables: set them before running assertions.
//
// This package is for programs, not tests: in _test.go files use the testing
// package's t.Errorf (or a library such as testify), which integrates with
// go test.
package assert

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/mikeland73/golibs/pp"
)

// Mode selects what happens when an assertion fails.
type Mode int

const (
	// Panic panics with the failure message. This is the default.
	Panic Mode = iota
	// Log writes the failure message to Output and continues.
	Log
	// Exit writes the failure message to Output and calls os.Exit(1).
	Exit
)

// String returns the mode name.
func (m Mode) String() string {
	switch m {
	case Panic:
		return "Panic"
	case Log:
		return "Log"
	case Exit:
		return "Exit"
	}
	return fmt.Sprintf("Mode(%d)", int(m))
}

// Output is where failures (in Log and Exit mode) and Verbose pass lines are
// written. It defaults to os.Stderr.
var Output io.Writer = os.Stderr

// Verbose makes every passing assertion print a line such as
// "✓ Equal at main.go:12" to Output.
var Verbose bool

var (
	mode     atomic.Int64 // Mode; zero value is Panic
	failures atomic.Int64
	outMu    sync.Mutex
	exit     = os.Exit // replaced in tests
)

// SetMode sets the failure behavior for all subsequent assertions and returns
// the previous mode, so it can be restored with
// defer assert.SetMode(assert.SetMode(assert.Log)).
func SetMode(m Mode) Mode {
	return Mode(mode.Swap(int64(m)))
}

// Failures returns the number of assertions that have failed so far, in any
// mode. It is useful in Log mode to print a summary or set an exit code.
func Failures() int {
	return int(failures.Load())
}

// callerSkip is the number of stack frames between report and the user's
// call site: report <- exported assertion <- user code.
const callerSkip = 2

// report records the outcome of an assertion. details are extra lines
// (e.g. "got:  ...") shown on failure.
func report(name string, ok bool, msgAndArgs []any, details ...string) bool {
	loc := "???"
	if _, file, line, found := runtime.Caller(callerSkip); found {
		loc = fmt.Sprintf("%s:%d", filepath.Base(file), line)
	}
	msg := message(msgAndArgs)
	if ok {
		if Verbose {
			s := "✓ " + name + " at " + loc
			if msg != "" {
				s += ": " + msg
			}
			write(s + "\n")
		}
		return true
	}
	failures.Add(1)
	var b strings.Builder
	fmt.Fprintf(&b, "assert.%s failed at %s", name, loc)
	if msg != "" {
		b.WriteString(": ")
		b.WriteString(msg)
	}
	for _, d := range details {
		b.WriteString("\n  ")
		b.WriteString(d)
	}
	switch Mode(mode.Load()) {
	case Log:
		write(b.String() + "\n")
	case Exit:
		write(b.String() + "\n")
		exit(1)
	default:
		panic(b.String())
	}
	return false
}

func write(s string) {
	outMu.Lock()
	defer outMu.Unlock()
	_, _ = io.WriteString(Output, s)
}

func message(msgAndArgs []any) string {
	switch len(msgAndArgs) {
	case 0:
		return ""
	case 1:
		if s, ok := msgAndArgs[0].(string); ok {
			return s
		}
		return fmt.Sprint(msgAndArgs[0])
	}
	if format, ok := msgAndArgs[0].(string); ok {
		return fmt.Sprintf(format, msgAndArgs[1:]...)
	}
	return fmt.Sprint(msgAndArgs...)
}

// show formats a value for a failure message with pp.Repr: readable,
// deterministic and safe for values that contain themselves.
func show(v any) string { return pp.Repr(v) }

func gotWant(got, want any) []string {
	g, w := show(got), show(want)
	if g == w {
		// Same text, different values (e.g. int vs int64 inside an any):
		// show the dynamic types too.
		g, w = fmt.Sprintf("%s (%T)", g, got), fmt.Sprintf("%s (%T)", w, want)
	}
	return []string{"got:  " + g, "want: " + w}
}

// Equal asserts that got and want are deeply equal. It works like
// reflect.DeepEqual except that nil and empty slices are considered equal,
// as are nil and empty maps (at any depth), so Equal(result, []int{}) passes
// for a nil result. As with reflect.DeepEqual, NaN is not equal to NaN (use
// InDelta for floats).
func Equal[T any](got, want T, msgAndArgs ...any) bool {
	ok := equal(got, want)
	var details []string
	if !ok { // format lazily: passing assertions should be cheap
		details = gotWant(got, want)
	}
	return report("Equal", ok, msgAndArgs, details...)
}

// NotEqual asserts that got and want are not equal in the sense of Equal.
func NotEqual[T any](got, want T, msgAndArgs ...any) bool {
	if equal(got, want) {
		return report("NotEqual", false, msgAndArgs, "both: "+show(got))
	}
	return report("NotEqual", true, msgAndArgs)
}

// True asserts that cond is true.
func True(cond bool, msgAndArgs ...any) bool {
	return report("True", cond, msgAndArgs)
}

// False asserts that cond is false.
func False(cond bool, msgAndArgs ...any) bool {
	return report("False", !cond, msgAndArgs)
}

// isNil reports whether v is nil or holds a nil pointer, slice, map, channel,
// func or interface. Empty (non-nil) slices and maps are not nil.
func isNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer,
		reflect.Slice, reflect.UnsafePointer:
		return rv.IsNil()
	}
	return false
}

// Nil asserts that v is nil, including typed nils such as a nil *T, nil
// slice or nil map stored in v. An empty but non-nil slice or map is not nil.
func Nil(v any, msgAndArgs ...any) bool {
	if !isNil(v) {
		return report("Nil", false, msgAndArgs, "got: "+show(v))
	}
	return report("Nil", true, msgAndArgs)
}

// NotNil asserts that v is not nil (see Nil for what counts as nil).
func NotNil(v any, msgAndArgs ...any) bool {
	return report("NotNil", !isNil(v), msgAndArgs, fmt.Sprintf("got: nil (%T)", v))
}

// NoError asserts that err is nil.
func NoError(err error, msgAndArgs ...any) bool {
	if err == nil {
		return report("NoError", true, msgAndArgs)
	}
	return report("NoError", false, msgAndArgs, "error: "+err.Error())
}

// Error asserts that err is not nil.
func Error(err error, msgAndArgs ...any) bool {
	return report("Error", err != nil, msgAndArgs, "got: nil error")
}

// ErrorIs asserts that errors.Is(err, target) is true.
func ErrorIs(err, target error, msgAndArgs ...any) bool {
	ok := errors.Is(err, target)
	errStr := "nil"
	if err != nil {
		errStr = fmt.Sprintf("%q (%T)", err.Error(), err)
	}
	targetStr := "nil"
	if target != nil {
		targetStr = fmt.Sprintf("%q (%T)", target.Error(), target)
	}
	return report("ErrorIs", ok, msgAndArgs, "error:  "+errStr, "target: "+targetStr)
}

// Panics asserts that calling f panics. The panic is recovered.
func Panics(f func(), msgAndArgs ...any) bool {
	return report("Panics", didPanic(f), msgAndArgs, "function did not panic")
}

func didPanic(f func()) (panicked bool) {
	defer func() {
		// Since Go 1.21, panic(nil) is recovered as a *runtime.PanicNilError,
		// so recover() is non-nil for every panic.
		if recover() != nil {
			panicked = true
		}
	}()
	f()
	return false
}

// Len asserts that v (a slice, array, map, string, channel, or pointer to
// array) has length n. It fails if v has no length.
func Len(v any, n int, msgAndArgs ...any) bool {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map, reflect.String, reflect.Chan:
	case reflect.Pointer:
		if rv.IsNil() || rv.Elem().Kind() != reflect.Array {
			return report("Len", false, msgAndArgs, fmt.Sprintf("value of type %T has no length", v))
		}
	default:
		return report("Len", false, msgAndArgs, fmt.Sprintf("value of type %T has no length", v))
	}
	if l := rv.Len(); l != n {
		return report("Len", false, msgAndArgs,
			fmt.Sprintf("got len:  %d", l), fmt.Sprintf("want len: %d", n), "value: "+show(v))
	}
	return report("Len", true, msgAndArgs)
}

// ElementsMatch asserts that got and want contain the same elements with the
// same multiplicities, in any order (like comparing collections.Counter in
// Python). got and want must have the same length.
//
// Elements are compared with ==, except that NaN matches NaN (also inside
// arrays, structs and interfaces, like InDelta), and elements of an interface
// type holding values that == cannot compare (slices, maps, funcs) are
// compared like Equal instead of panicking. It runs in O(n) expected time;
// elements containing NaN or uncomparable values are matched pairwise, in
// O(k²) time for k such elements.
func ElementsMatch[T comparable](got, want []T, msgAndArgs ...any) bool {
	extra, missing := multisetDiff(got, want)
	if len(got) == len(want) && len(extra) == 0 && len(missing) == 0 {
		return report("ElementsMatch", true, msgAndArgs)
	}
	details := gotWant(got, want)
	if len(got) != len(want) {
		details = append(details, fmt.Sprintf("len(got) = %d, len(want) = %d", len(got), len(want)))
	}
	if len(extra) > 0 {
		details = append(details, "extra (in got, not in want):   "+show(extra))
	}
	if len(missing) > 0 {
		details = append(details, "missing (in want, not in got): "+show(missing))
	}
	return report("ElementsMatch", false, msgAndArgs, details...)
}

// slowElem is an ElementsMatch element that cannot be used as a map key
// (it contains NaN, or an uncomparable value inside an interface).
type slowElem[T any] struct {
	v          T
	comparable bool // == can compare it (it just contains NaN)
	used       bool
}

// classify reports whether x can be counted in a map (it is comparable and
// equal to itself), and whether it is comparable at all. check is false when
// T's static type guarantees both.
func classify[T comparable](x T, check bool) (hashable, comparable bool) {
	if !check {
		return true, true
	}
	if !reflect.ValueOf(&x).Elem().Comparable() {
		return false, false
	}
	return x == x, true // false iff x contains a NaN
}

// mayNeedSlowPath reports whether values of type t can contain NaN or hold
// uncomparable values in interfaces.
func mayNeedSlowPath(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.Interface:
		return true
	case reflect.Array:
		return mayNeedSlowPath(t.Elem())
	case reflect.Struct:
		for i := range t.NumField() {
			if mayNeedSlowPath(t.Field(i).Type) {
				return true
			}
		}
	}
	return false
}

// multisetDiff returns the elements of got not matched in want (extra) and of
// want not matched in got (missing), each in their original order.
func multisetDiff[T comparable](got, want []T) (extra, missing []T) {
	check := mayNeedSlowPath(reflect.TypeFor[T]())
	counts := make(map[T]int, len(want))
	var slow []*slowElem[T]
	for _, x := range want {
		if h, c := classify(x, check); h {
			counts[x]++
		} else {
			slow = append(slow, &slowElem[T]{v: x, comparable: c})
		}
	}
	for _, x := range got {
		h, c := classify(x, check)
		if h {
			if counts[x] > 0 {
				counts[x]--
			} else {
				extra = append(extra, x)
			}
			continue
		}
		matched := false
		for _, w := range slow {
			if !w.used && elementsEqual(x, w.v, c && w.comparable) {
				w.used, matched = true, true
				break
			}
		}
		if !matched {
			extra = append(extra, x)
		}
	}
	si := 0
	for _, x := range want {
		if h, _ := classify(x, check); h {
			if counts[x] > 0 {
				missing = append(missing, x)
				counts[x]--
			}
			continue
		}
		if !slow[si].used {
			missing = append(missing, x)
		}
		si++
	}
	return extra, missing
}

// elementsEqual compares two ElementsMatch elements with NaN equal to NaN.
// If both are comparable it follows == semantics (pointers by address);
// otherwise it uses Equal's deep equality.
func elementsEqual(a, b any, comparable bool) bool {
	c := comparer{visited: make(map[visit]bool), nanEqual: true, shallow: comparable}
	return c.equal(reflect.ValueOf(a), reflect.ValueOf(b))
}

// InDelta asserts that |got - want| <= delta. Two NaNs are considered equal;
// a NaN and a number are not.
func InDelta(got, want, delta float64, msgAndArgs ...any) bool {
	var ok bool
	switch {
	case math.IsNaN(got) || math.IsNaN(want):
		ok = math.IsNaN(got) && math.IsNaN(want)
	case got == want: // handles equal infinities
		ok = true
	default:
		ok = math.Abs(got-want) <= delta
	}
	return report("InDelta", ok, msgAndArgs,
		fmt.Sprintf("got:   %v", got), fmt.Sprintf("want:  %v", want),
		fmt.Sprintf("delta: %v (actual difference %v)", delta, math.Abs(got-want)))
}

// ---------------------------------------------------------------------------
// Deep equality treating nil and empty slices/maps as equal.

type visit struct {
	a1, a2 uintptr
	typ    reflect.Type
	n      int // slice length: s[:1] and s share a pointer but differ
}

// comparer implements deep equality. visited records container pairs being
// (or already found to be) equal, to terminate on cycles. nanEqual makes NaN
// equal to NaN; shallow compares pointers by address, as == does.
type comparer struct {
	visited  map[visit]bool
	nanEqual bool
	shallow  bool
}

func equal(a, b any) bool {
	c := comparer{visited: make(map[visit]bool)}
	return c.equal(reflect.ValueOf(a), reflect.ValueOf(b))
}

func (c *comparer) floatEqual(a, b float64) bool {
	return a == b || (c.nanEqual && math.IsNaN(a) && math.IsNaN(b))
}

func (c *comparer) equal(v1, v2 reflect.Value) bool {
	if !v1.IsValid() || !v2.IsValid() {
		return v1.IsValid() == v2.IsValid()
	}
	if v1.Type() != v2.Type() {
		return false
	}
	if c.shallow && v1.Kind() == reflect.Pointer {
		return v1.Pointer() == v2.Pointer()
	}
	switch v1.Kind() {
	case reflect.Map, reflect.Slice, reflect.Pointer:
		if v1.Kind() != reflect.Pointer && v1.Len() == 0 && v2.Len() == 0 {
			return true
		}
		if v1.IsNil() || v2.IsNil() {
			return v1.IsNil() == v2.IsNil()
		}
		n := 0
		if v1.Kind() == reflect.Slice {
			if v1.Len() != v2.Len() {
				return false
			}
			n = v1.Len()
		}
		p1, p2 := v1.Pointer(), v2.Pointer()
		if p1 == p2 {
			return true // same memory (like reflect.DeepEqual, even with NaNs)
		}
		if p1 > p2 {
			p1, p2 = p2, p1
		}
		k := visit{p1, p2, v1.Type(), n}
		if c.visited[k] {
			return true // already being compared higher up
		}
		c.visited[k] = true
	}
	switch v1.Kind() {
	case reflect.Array, reflect.Slice:
		for i := range v1.Len() {
			if !c.equal(v1.Index(i), v2.Index(i)) {
				return false
			}
		}
		return true
	case reflect.Interface:
		if v1.IsNil() || v2.IsNil() {
			return v1.IsNil() == v2.IsNil()
		}
		return c.equal(v1.Elem(), v2.Elem())
	case reflect.Pointer:
		return c.equal(v1.Elem(), v2.Elem())
	case reflect.Struct:
		for i := range v1.NumField() {
			if !c.equal(v1.Field(i), v2.Field(i)) {
				return false
			}
		}
		return true
	case reflect.Map:
		if v1.Len() != v2.Len() {
			return false
		}
		iter := v1.MapRange()
		for iter.Next() {
			val2 := v2.MapIndex(iter.Key())
			if !val2.IsValid() || !c.equal(iter.Value(), val2) {
				return false
			}
		}
		return true
	case reflect.Func:
		return v1.IsNil() && v2.IsNil()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v1.Int() == v2.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v1.Uint() == v2.Uint()
	case reflect.String:
		return v1.String() == v2.String()
	case reflect.Bool:
		return v1.Bool() == v2.Bool()
	case reflect.Float32, reflect.Float64:
		return c.floatEqual(v1.Float(), v2.Float())
	case reflect.Complex64, reflect.Complex128:
		z1, z2 := v1.Complex(), v2.Complex()
		return c.floatEqual(real(z1), real(z2)) && c.floatEqual(imag(z1), imag(z2))
	case reflect.Chan, reflect.UnsafePointer:
		return v1.Pointer() == v2.Pointer()
	}
	return false
}
