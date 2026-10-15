package grid

import (
	"math/rand/v2"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/mikeland73/golibs/mathx"
)

func TestNewAndFill(t *testing.T) {
	g := New[int](2, 3)
	if len(g) != 2 || len(g[0]) != 3 || len(g[1]) != 3 {
		t.Fatalf("New dims wrong: %v", g)
	}
	g[0][2] = 7
	g[0] = append(g[0], 99) // must not clobber g[1][0]
	if g[1][0] != 0 {
		t.Errorf("append to row 0 overwrote row 1: %v", g)
	}
	if g[0][2] != 7 || g[0][3] != 99 {
		t.Errorf("row 0 after append = %v", g[0])
	}

	f := Fill(3, 2, "x")
	for _, row := range f {
		if !slices.Equal(row, []string{"x", "x"}) {
			t.Errorf("Fill row = %v", row)
		}
	}
	if got := New[int](0, 5); len(got) != 0 {
		t.Errorf("New(0,5) = %v", got)
	}
	if got := New[int](3, 0); len(got) != 3 || len(got[2]) != 0 {
		t.Errorf("New(3,0) = %v", got)
	}
}

func TestClone(t *testing.T) {
	g := [][]int{{1, 2}, {3}, nil}
	c := Clone(g)
	if !reflect.DeepEqual(c, g) {
		t.Fatalf("Clone = %v, want %v", c, g)
	}
	c[0][0] = 100
	if g[0][0] != 1 {
		t.Errorf("Clone shares rows")
	}
	if Clone[int](nil) != nil {
		t.Errorf("Clone(nil) != nil")
	}
	if c := Clone([][]int{}); c == nil || len(c) != 0 {
		t.Errorf("Clone(empty) = %#v", c)
	}
}

func TestTransposeRotate(t *testing.T) {
	g := [][]int{
		{1, 2, 3},
		{4, 5, 6},
	}
	tests := []struct {
		name string
		f    func([][]int) [][]int
		want [][]int
	}{
		{"Transpose", Transpose[int], [][]int{{1, 4}, {2, 5}, {3, 6}}},
		{"RotateCW", RotateCW[int], [][]int{{4, 1}, {5, 2}, {6, 3}}},
		{"RotateCCW", RotateCCW[int], [][]int{{3, 6}, {2, 5}, {1, 4}}},
	}
	for _, tt := range tests {
		got := tt.f(g)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s = %v, want %v", tt.name, got, tt.want)
		}
		if got := tt.f(nil); len(got) != 0 {
			t.Errorf("%s(nil) = %v", tt.name, got)
		}
		if got := tt.f([][]int{{7}}); !reflect.DeepEqual(got, [][]int{{7}}) {
			t.Errorf("%s(1x1) = %v", tt.name, got)
		}
		func() {
			defer func() {
				r := recover()
				if r == nil || !strings.Contains(r.(string), "rectangular") {
					t.Errorf("%s jagged: panic = %v", tt.name, r)
				}
			}()
			tt.f([][]int{{1, 2}, {3}})
		}()
	}
	if g[0][0] != 1 || g[1][2] != 6 {
		t.Errorf("input modified: %v", g)
	}
}

func TestRotateProperties(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	for range 100 {
		rows, cols := 1+r.IntN(6), 1+r.IntN(6)
		g := New[int](rows, cols)
		for p := range Cells(g) {
			g[p.R][p.C] = r.IntN(100)
		}
		if !reflect.DeepEqual(RotateCCW(RotateCW(g)), g) {
			t.Fatalf("CCW(CW(g)) != g")
		}
		if !reflect.DeepEqual(Transpose(Transpose(g)), g) {
			t.Fatalf("T(T(g)) != g")
		}
		four := g
		for range 4 {
			four = RotateCW(four)
		}
		if !reflect.DeepEqual(four, g) {
			t.Fatalf("CW^4(g) != g")
		}
		// CW = reverse rows of the transpose.
		tr := Transpose(g)
		for _, row := range tr {
			slices.Reverse(row)
		}
		if !reflect.DeepEqual(RotateCW(g), tr) {
			t.Fatalf("CW != reversed transpose")
		}
	}
}

func TestFromStrings(t *testing.T) {
	lines := []string{"#.", "..#"}
	g := FromStrings(lines)
	if string(g[0]) != "#." || string(g[1]) != "..#" {
		t.Errorf("FromStrings = %q", g)
	}
	g[0][0] = '.'
	if lines[0] != "#." {
		t.Errorf("FromStrings aliases input")
	}
	if got := FromStrings(nil); len(got) != 0 {
		t.Errorf("FromStrings(nil) = %v", got)
	}
}

func TestPoint(t *testing.T) {
	p := Point{2, 3}
	if got := p.Add(Point{-1, 4}); got != (Point{1, 7}) {
		t.Errorf("Add = %v", got)
	}
	if got := p.Manhattan(Point{-1, 5}); got != 5 {
		t.Errorf("Manhattan = %d", got)
	}
	if got := p.Manhattan(p); got != 0 {
		t.Errorf("Manhattan self = %d", got)
	}
	// Turning right four times returns to the start.
	pos, dir := Point{}, 0
	for range 4 {
		pos = pos.Add(Dirs4[dir])
		dir = (dir + 1) % 4
	}
	if pos != (Point{}) {
		t.Errorf("square walk ended at %v", pos)
	}
	seen := map[Point]bool{}
	for _, d := range Dirs8 {
		if d == (Point{}) || seen[d] || mathx.Abs(d.R) > 1 || mathx.Abs(d.C) > 1 {
			t.Errorf("bad Dirs8 entry %v", d)
		}
		seen[d] = true
	}
	for i, d := range Dirs4 {
		if Dirs8[2*i] != d {
			t.Errorf("Dirs8[%d] = %v, want Dirs4[%d] = %v", 2*i, Dirs8[2*i], i, d)
		}
	}
}

func TestInBounds(t *testing.T) {
	g := [][]int{{1, 2, 3}, {4}}
	tests := []struct {
		r, c int
		want bool
	}{
		{0, 0, true},
		{0, 2, true},
		{1, 0, true},
		{1, 1, false},
		{-1, 0, false},
		{0, -1, false},
		{2, 0, false},
		{0, 3, false},
	}
	for _, tt := range tests {
		if got := InBounds(g, tt.r, tt.c); got != tt.want {
			t.Errorf("InBounds(%d,%d) = %v", tt.r, tt.c, got)
		}
	}
	if InBounds[int](nil, 0, 0) {
		t.Errorf("InBounds(nil) = true")
	}
}

func collect(seq func(func(int, int) bool)) []Point {
	var out []Point
	seq(func(r, c int) bool {
		out = append(out, Point{r, c})
		return true
	})
	return out
}

func TestNeighbors(t *testing.T) {
	g := New[int](3, 3)
	tests := []struct {
		name string
		got  []Point
		want []Point
	}{
		{"4 center", collect(Neighbors4(g, 1, 1)), []Point{{0, 1}, {1, 2}, {2, 1}, {1, 0}}},
		{"4 corner", collect(Neighbors4(g, 0, 0)), []Point{{0, 1}, {1, 0}}},
		{"4 bottom-right", collect(Neighbors4(g, 2, 2)), []Point{{1, 2}, {2, 1}}},
		{"8 center", collect(Neighbors8(g, 1, 1)), []Point{{0, 1}, {0, 2}, {1, 2}, {2, 2}, {2, 1}, {2, 0}, {1, 0}, {0, 0}}},
		{"8 corner", collect(Neighbors8(g, 0, 0)), []Point{{0, 1}, {1, 1}, {1, 0}}},
		{"4 1x1", collect(Neighbors4(New[int](1, 1), 0, 0)), nil},
		{"4 jagged", collect(Neighbors4([][]int{{1, 2}, {3}}, 0, 1)), []Point{{0, 0}}},
	}
	for _, tt := range tests {
		if !slices.Equal(tt.got, tt.want) {
			t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
		}
	}
	n := 0
	for range Neighbors8(g, 1, 1) {
		n++
		if n == 2 {
			break
		}
	}
	if n != 2 {
		t.Errorf("early break failed")
	}
}

func TestCellsAndFind(t *testing.T) {
	g := [][]string{{"a", "b"}, {}, {"c", "b"}}
	var ps []Point
	var vs []string
	for p, v := range Cells(g) {
		ps = append(ps, p)
		vs = append(vs, v)
	}
	if !slices.Equal(ps, []Point{{0, 0}, {0, 1}, {2, 0}, {2, 1}}) || strings.Join(vs, "") != "abcb" {
		t.Errorf("Cells = %v %v", ps, vs)
	}
	for range Cells(g) {
		break
	}
	if p, ok := Find(g, "b"); !ok || p != (Point{0, 1}) {
		t.Errorf("Find(b) = %v %v", p, ok)
	}
	if p, ok := Find(g, "c"); !ok || p != (Point{2, 0}) {
		t.Errorf("Find(c) = %v %v", p, ok)
	}
	if _, ok := Find(g, "z"); ok {
		t.Errorf("Find(z) found")
	}
	if _, ok := Find[int](nil, 0); ok {
		t.Errorf("Find in nil found")
	}
}
