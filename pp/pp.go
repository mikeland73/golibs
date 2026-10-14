// Package pp is a pretty printer for debugging, in the spirit of Python's
// print/repr/pprint and Rust's dbg! macro.
//
//	pp.Print("dist:", map[string]int{"b": 2, "a": 1}) // dist: {"a": 1, "b": 2}
//	pp.Print([]string{"x", "y"}, &Point{1, 2})        // ["x", "y"] Point{X: 1, Y: 2}
//	pp.Debug(i, xs)    // stderr: [main.go:12] i = 3, xs = [1, 2, 3]
//	pp.Grid(board)     // [][]byte / [][]rune print as rows of characters
//	pp.JSON(resp)      // indented JSON
//
// Formatting rules (deterministic, Python-like):
//
//   - Slices and arrays print as [1, 2, 3]; nil slices print as [].
//   - Maps print as {k: v, ...} sorted by key (numbers numerically, strings
//     lexically, structs and arrays field by field); nil maps print as {}.
//     Pointer and channel keys are ordered by address, which is stable
//     within a run but not across runs.
//     Maps whose values are struct{} are treated as sets: {1, 2, 3}.
//   - Strings are unquoted at the top level and quoted ("a") inside
//     containers, struct fields and Debug/Repr output.
//   - Structs print as Point{X: 1, Y: 2}, including unexported fields.
//   - Pointers are followed (nil prints as nil, even if its type has a
//     String or Error method); a pointer cycle prints as <cycle>.
//   - Values implementing error or fmt.Stringer (with value or pointer
//     receivers) print via Error()/String(), as long as they are accessible:
//     like fmt, methods are not invoked on values read from unexported struct
//     fields, which are formatted structurally instead.
//   - Elements of []rune (any slice/array of int32) and rune map keys print as
//     quoted characters ('a'). Bytes print as numbers, except in Grid.
//   - Channels and funcs print as their type, e.g. chan int.
//
// Print, Grid and JSON write to standard output; Debug writes to Output
// (standard error by default).
package pp

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// Output is where Debug writes. It defaults to os.Stderr. Set it before use
// (e.g. to a bytes.Buffer in tests); it is not synchronized itself, but each
// Debug line is written with a single locked Write call.
var Output io.Writer = os.Stderr

var outMu sync.Mutex

// Sprint formats each value according to the package rules and joins them
// with single spaces, like Python's str(a) + " " + str(b). Top-level strings
// are not quoted.
func Sprint(vals ...any) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = format(v, false)
	}
	return strings.Join(parts, " ")
}

// Print writes Sprint(vals...) followed by a newline to standard output, like
// Python's print(*vals).
func Print(vals ...any) {
	fmt.Fprintln(os.Stdout, Sprint(vals...))
}

// Repr formats a single value like Sprint, except that a top-level string is
// quoted, like Python's repr(v).
func Repr(v any) string {
	return format(v, true)
}

func format(x any, quote bool) string {
	p := printer{path: make(map[visitKey]bool)}
	p.value(reflect.ValueOf(x), quote, false)
	return p.b.String()
}

var (
	errorType    = reflect.TypeFor[error]()
	stringerType = reflect.TypeFor[fmt.Stringer]()
	durationType = reflect.TypeFor[time.Duration]()
	runeType     = reflect.TypeFor[rune]()
	byteType     = reflect.TypeFor[byte]()
)

type visitKey struct {
	typ reflect.Type
	ptr uintptr
	n   int
}

type printer struct {
	b    strings.Builder
	path map[visitKey]bool // containers on the current path, for cycle detection
}

// enter records that a container is being printed; it returns false if it is
// already on the current path (a cycle).
func (p *printer) enter(k visitKey) bool {
	if p.path[k] {
		p.b.WriteString("<cycle>")
		return false
	}
	p.path[k] = true
	return true
}

// methodString returns the result of v's Error or String method, if any.
func methodString(v reflect.Value) (s string, ok bool) {
	if !v.CanInterface() {
		if v.Type() == durationType {
			return time.Duration(v.Int()).String(), true
		}
		return "", false
	}
	t := v.Type()
	var recv reflect.Value
	switch {
	case t.Implements(errorType) || t.Implements(stringerType):
		recv = v
	case t.Kind() != reflect.Pointer && t.Kind() != reflect.Interface &&
		(reflect.PointerTo(t).Implements(errorType) || reflect.PointerTo(t).Implements(stringerType)):
		if v.CanAddr() {
			recv = v.Addr()
		} else {
			recv = reflect.New(t)
			recv.Elem().Set(v)
		}
	default:
		return "", false
	}
	defer func() {
		if r := recover(); r != nil {
			s, ok = fmt.Sprintf("<%s method panicked: %v>", t, r), true
		}
	}()
	switch m := recv.Interface().(type) {
	case error:
		return m.Error(), true
	case fmt.Stringer:
		return m.String(), true
	}
	return "", false
}

// value formats v. quote controls whether strings are quoted; asRune formats
// int32 values as quoted characters.
func (p *printer) value(v reflect.Value, quote, asRune bool) {
	if !v.IsValid() {
		p.b.WriteString("nil")
		return
	}
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			p.b.WriteString("nil")
			return
		}
	}
	if v.Kind() == reflect.Interface {
		// Format the dynamic value, so that a nil pointer in an interface
		// prints as nil rather than calling (and panicking in) its methods.
		p.value(v.Elem(), quote, asRune)
		return
	}
	if s, ok := methodString(v); ok {
		p.b.WriteString(s)
		return
	}
	switch v.Kind() {
	case reflect.Pointer:
		k := visitKey{v.Type(), v.Pointer(), 0}
		if !p.enter(k) {
			return
		}
		p.value(v.Elem(), quote, asRune)
		delete(p.path, k)
	case reflect.Bool:
		p.b.WriteString(strconv.FormatBool(v.Bool()))
	case reflect.Int32:
		if asRune && v.Type() == runeType {
			p.b.WriteString(strconv.QuoteRune(rune(v.Int())))
			return
		}
		p.b.WriteString(strconv.FormatInt(v.Int(), 10))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int64:
		p.b.WriteString(strconv.FormatInt(v.Int(), 10))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p.b.WriteString(strconv.FormatUint(v.Uint(), 10))
	case reflect.Float32:
		p.b.WriteString(strconv.FormatFloat(v.Float(), 'g', -1, 32))
	case reflect.Float64:
		p.b.WriteString(strconv.FormatFloat(v.Float(), 'g', -1, 64))
	case reflect.Complex64:
		p.b.WriteString(strconv.FormatComplex(v.Complex(), 'g', -1, 64))
	case reflect.Complex128:
		p.b.WriteString(strconv.FormatComplex(v.Complex(), 'g', -1, 128))
	case reflect.String:
		if quote {
			p.b.WriteString(strconv.Quote(v.String()))
		} else {
			p.b.WriteString(v.String())
		}
	case reflect.Slice:
		if v.Len() > 0 {
			k := visitKey{v.Type(), v.Pointer(), v.Len()}
			if !p.enter(k) {
				return
			}
			defer delete(p.path, k)
		}
		p.list(v)
	case reflect.Array:
		p.list(v)
	case reflect.Map:
		if v.Len() > 0 {
			k := visitKey{v.Type(), v.Pointer(), 0}
			if !p.enter(k) {
				return
			}
			defer delete(p.path, k)
		}
		p.mapValue(v)
	case reflect.Struct:
		p.structValue(v)
	case reflect.UnsafePointer:
		fmt.Fprintf(&p.b, "%#x", v.Pointer())
	default: // Chan, Func
		p.b.WriteString(v.Type().String())
	}
}

func (p *printer) list(v reflect.Value) {
	asRune := v.Type().Elem() == runeType
	p.b.WriteByte('[')
	for i := range v.Len() {
		if i > 0 {
			p.b.WriteString(", ")
		}
		p.value(v.Index(i), true, asRune)
	}
	p.b.WriteByte(']')
}

func (p *printer) mapValue(v reflect.Value) {
	t := v.Type()
	// Iterate over entries rather than looking values up by key: NaN keys
	// cannot be looked up.
	type entry struct{ k, v reflect.Value }
	entries := make([]entry, 0, v.Len())
	for iter := v.MapRange(); iter.Next(); {
		entries = append(entries, entry{iter.Key(), iter.Value()})
	}
	slices.SortFunc(entries, func(a, b entry) int {
		return cmp.Or(compareValues(a.k, b.k), compareValues(a.v, b.v))
	})
	keyRune := t.Key() == runeType
	isSet := t.Elem().Kind() == reflect.Struct && t.Elem().NumField() == 0
	p.b.WriteByte('{')
	for i, e := range entries {
		if i > 0 {
			p.b.WriteString(", ")
		}
		p.value(e.k, true, keyRune)
		if !isSet {
			p.b.WriteString(": ")
			p.value(e.v, true, false)
		}
	}
	p.b.WriteByte('}')
}

func (p *printer) structValue(v reflect.Value) {
	t := v.Type()
	p.b.WriteString(t.Name())
	p.b.WriteByte('{')
	for i := range t.NumField() {
		if i > 0 {
			p.b.WriteString(", ")
		}
		p.b.WriteString(t.Field(i).Name)
		p.b.WriteString(": ")
		p.value(v.Field(i), true, false)
	}
	p.b.WriteByte('}')
}

// compareValues orders map keys deterministically. a and b have the same
// static type.
func compareValues(a, b reflect.Value) int {
	switch a.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return cmp.Compare(a.Int(), b.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return cmp.Compare(a.Uint(), b.Uint())
	case reflect.Float32, reflect.Float64:
		return cmp.Compare(a.Float(), b.Float())
	case reflect.Complex64, reflect.Complex128:
		ca, cb := a.Complex(), b.Complex()
		return cmp.Or(cmp.Compare(real(ca), real(cb)), cmp.Compare(imag(ca), imag(cb)))
	case reflect.String:
		return cmp.Compare(a.String(), b.String())
	case reflect.Bool:
		switch {
		case a.Bool() == b.Bool():
			return 0
		case !a.Bool():
			return -1
		default:
			return 1
		}
	case reflect.Pointer, reflect.Chan, reflect.UnsafePointer:
		return cmp.Compare(a.Pointer(), b.Pointer())
	case reflect.Struct:
		for i := range a.NumField() {
			if c := compareValues(a.Field(i), b.Field(i)); c != 0 {
				return c
			}
		}
		return 0
	case reflect.Array:
		for i := range a.Len() {
			if c := compareValues(a.Index(i), b.Index(i)); c != 0 {
				return c
			}
		}
		return 0
	case reflect.Interface:
		switch {
		case a.IsNil() && b.IsNil():
			return 0
		case a.IsNil():
			return -1
		case b.IsNil():
			return 1
		}
		ea, eb := a.Elem(), b.Elem()
		if ea.Type() != eb.Type() {
			return cmp.Compare(ea.Type().String(), eb.Type().String())
		}
		return compareValues(ea, eb)
	}
	return 0
}

// ---------------------------------------------------------------------------
// Debug

// Debug prints its arguments to Output together with the caller's file and
// line and the source text of each argument, like Rust's dbg! macro:
//
//	pp.Debug(i, xs, "here")  // [main.go:12] i = 3, xs = [1, 2, 3], here
//
// Values are formatted like Repr (strings quoted). Literal arguments are
// printed without a "expr = " prefix, so string literals act as labels.
//
// The argument text is recovered by parsing the caller's source file (parsed
// files are cached) and finding the call on the caller's line: pkg.Debug(...)
// where pkg is the name under which the file imports this package, or a
// plain Debug(...) in this package's own files or with a dot import. If the
// source is unavailable (e.g. binaries built with -trimpath or run on another
// machine), the call cannot be identified unambiguously (e.g. two Debug calls
// with the same number of arguments on one line, or Debug called through a
// variable), or the arguments are passed with "...", Debug falls back to
// printing only the values: [main.go:12] 3, [1, 2, 3].
func Debug(vals ...any) {
	pc, file, line, ok := runtime.Caller(1)
	var exprs []ast.Expr
	var src *srcFile
	if ok {
		src, exprs = debugArgs(file, line, len(vals), funcPkgPath(pc) == ownFuncPkg)
	}
	var b strings.Builder
	if ok {
		fmt.Fprintf(&b, "[%s:%d]", filepath.Base(file), line)
	} else {
		b.WriteString("[???]")
	}
	for i, v := range vals {
		if i == 0 {
			b.WriteByte(' ')
		} else {
			b.WriteString(", ")
		}
		switch {
		case exprs == nil:
			b.WriteString(Repr(v))
		case isLiteral(exprs[i]):
			b.WriteString(Sprint(v))
		default:
			b.WriteString(src.text(exprs[i]))
			b.WriteString(" = ")
			b.WriteString(Repr(v))
		}
	}
	b.WriteByte('\n')
	outMu.Lock()
	defer outMu.Unlock()
	_, _ = io.WriteString(Output, b.String())
}

func isLiteral(e ast.Expr) bool {
	switch e := e.(type) {
	case *ast.BasicLit:
		return true
	case *ast.UnaryExpr: // -1
		_, ok := e.X.(*ast.BasicLit)
		return ok
	case *ast.ParenExpr:
		return isLiteral(e.X)
	}
	return false
}

type srcFile struct {
	fset *token.FileSet
	file *ast.File
	src  []byte
}

func (s *srcFile) text(e ast.Expr) string {
	start := s.fset.Position(e.Pos()).Offset
	end := s.fset.Position(e.End()).Offset
	t := string(s.src[start:end])
	if strings.Contains(t, "\n") {
		t = strings.Join(strings.Fields(t), " ")
	}
	return t
}

var (
	parseMu    sync.Mutex
	parseCache = map[string]*srcFile{} // nil entry = unreadable/unparsable
)

func loadSource(path string) *srcFile {
	parseMu.Lock()
	defer parseMu.Unlock()
	if s, ok := parseCache[path]; ok {
		return s
	}
	var s *srcFile
	if src, err := os.ReadFile(path); err == nil {
		fset := token.NewFileSet()
		if f, err := parser.ParseFile(fset, path, src, parser.SkipObjectResolution); err == nil {
			s = &srcFile{fset: fset, file: f, src: src}
		}
	}
	parseCache[path] = s
	return s
}

var (
	// ownPkgPath is this package's import path; ownFuncPkg is the same path as
	// it appears in function symbol names (see funcPkgPath).
	ownPkgPath = reflect.TypeFor[printer]().PkgPath()
	ownFuncPkg = funcPkgPath(reflect.ValueOf(Sprint).Pointer())
)

// funcPkgPath returns the package path of the function containing pc, taken
// from its symbol name (e.g. "example.com/m/pkg.(*T).Method" ->
// "example.com/m/pkg"), or "" if unknown.
func funcPkgPath(pc uintptr) string {
	f := runtime.FuncForPC(pc)
	if f == nil {
		return ""
	}
	name := f.Name()
	if i := strings.IndexByte(name, '['); i >= 0 { // generic instantiation
		name = name[:i]
	}
	slash := strings.LastIndexByte(name, '/') + 1
	dot := strings.IndexByte(name[slash:], '.')
	if dot < 0 {
		return ""
	}
	return name[:slash+dot]
}

// debugNames returns the qualifiers under which f refers to this package's
// Debug function (e.g. "pp", or an import alias), and whether a bare Debug
// refers to it (in this package's own files, or with a dot import).
func debugNames(f *ast.File, samePkg bool) (quals map[string]bool, bare bool) {
	quals = make(map[string]bool)
	for _, imp := range f.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil || path != ownPkgPath {
			continue
		}
		switch {
		case imp.Name == nil:
			quals[pkgName] = true
		case imp.Name.Name == ".":
			bare = true
		case imp.Name.Name != "_":
			quals[imp.Name.Name] = true
		}
	}
	return quals, bare || samePkg
}

// pkgName is the name this package is imported under by default.
const pkgName = "pp"

// debugArgs finds the Debug call at file:line with n arguments and returns
// its argument expressions, or nil if it cannot be determined unambiguously.
// samePkg reports whether the caller is in this package (so that a bare
// Debug refers to it).
func debugArgs(file string, line, n int, samePkg bool) (*srcFile, []ast.Expr) {
	s := loadSource(file)
	if s == nil {
		return nil, nil
	}
	quals, bare := debugNames(s.file, samePkg)
	var exact, spanning []*ast.CallExpr
	ast.Inspect(s.file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || !isDebugCall(call, quals, bare) || call.Ellipsis.IsValid() || len(call.Args) != n {
			return true
		}
		start := s.fset.Position(call.Pos()).Line
		end := s.fset.Position(call.End()).Line
		switch {
		case start == line || s.fset.Position(call.Lparen).Line == line:
			exact = append(exact, call)
		case start <= line && line <= end:
			spanning = append(spanning, call)
		}
		return true
	})
	// runtime.Caller reports only a line, so several candidate calls on it
	// are ambiguous.
	switch {
	case len(exact) == 1:
		return s, exact[0].Args
	case len(exact) == 0 && len(spanning) == 1:
		return s, spanning[0].Args
	}
	return nil, nil
}

// isDebugCall reports whether call is qual.Debug(...) for one of quals, or
// Debug(...) if bare is set.
func isDebugCall(call *ast.CallExpr, quals map[string]bool, bare bool) bool {
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		return bare && fn.Name == "Debug"
	case *ast.SelectorExpr:
		x, ok := fn.X.(*ast.Ident)
		return ok && fn.Sel.Name == "Debug" && quals[x.Name]
	}
	return false
}

// ---------------------------------------------------------------------------
// Grid and JSON

// SprintGrid formats a 2D slice or array (e.g. [][]int, [][]string,
// [3][3]int) as rows of right-aligned, space-separated columns. [][]byte and
// [][]rune grids, and []string, are printed as rows of characters with no
// separators, which is the natural way to view a character grid:
//
//	#.#
//	.#.
//
// Values that are not 2D are formatted with Sprint. The result has no
// trailing newline; an empty grid yields "".
func SprintGrid(g any) string {
	v := reflect.ValueOf(g)
	for v.IsValid() && (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) && !v.IsNil() {
		v = v.Elem()
	}
	if !v.IsValid() || (v.Kind() != reflect.Slice && v.Kind() != reflect.Array) {
		return Sprint(g)
	}
	elem := v.Type().Elem()
	if elem.Kind() == reflect.String {
		lines := make([]string, v.Len())
		for i := range v.Len() {
			lines[i] = v.Index(i).String()
		}
		return strings.Join(lines, "\n")
	}
	if elem.Kind() != reflect.Slice && elem.Kind() != reflect.Array {
		return Sprint(g)
	}
	if cell := elem.Elem(); cell == byteType || cell == runeType {
		lines := make([]string, v.Len())
		for i := range v.Len() {
			row := v.Index(i)
			var b strings.Builder
			for j := range row.Len() {
				if cell == byteType {
					b.WriteByte(byte(row.Index(j).Uint()))
				} else {
					b.WriteRune(rune(row.Index(j).Int()))
				}
			}
			lines[i] = b.String()
		}
		return strings.Join(lines, "\n")
	}
	rows := make([][]string, v.Len())
	var widths []int
	for i := range v.Len() {
		row := v.Index(i)
		rows[i] = make([]string, row.Len())
		for j := range row.Len() {
			p := printer{path: make(map[visitKey]bool)}
			p.value(row.Index(j), false, false)
			s := p.b.String()
			rows[i][j] = s
			if j == len(widths) {
				widths = append(widths, 0)
			}
			widths[j] = max(widths[j], utf8.RuneCountInString(s))
		}
	}
	var b strings.Builder
	for i, row := range rows {
		if i > 0 {
			b.WriteByte('\n')
		}
		for j, s := range row {
			if j > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(strings.Repeat(" ", widths[j]-utf8.RuneCountInString(s)))
			b.WriteString(s)
		}
	}
	return b.String()
}

// Grid prints SprintGrid(g) followed by a newline to standard output.
func Grid(g any) {
	fmt.Fprintln(os.Stdout, SprintGrid(g))
}

// SprintJSON returns v encoded as JSON indented with two spaces, without
// HTML escaping. Since it is meant for debugging, an encoding error is
// returned as the string "<json error: ...>" rather than as an error.
func SprintJSON(v any) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return "<json error: " + err.Error() + ">"
	}
	return strings.TrimSuffix(buf.String(), "\n")
}

// JSON prints SprintJSON(v) followed by a newline to standard output.
func JSON(v any) {
	fmt.Fprintln(os.Stdout, SprintJSON(v))
}
