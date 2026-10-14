// Package grid provides helpers for 2D grids and matrices represented as
// [][]T: allocation, copying, transposition and rotation, bounds-checked
// neighbor iteration, and pretty printing.
//
//	g := grid.FromStrings([]string{
//		"#..",
//		".#.",
//	})
//	for nr, nc := range grid.Neighbors4(g, 0, 1) {
//		fmt.Println(nr, nc, string(g[nr][nc]))
//	}
//	fmt.Println(grid.String(grid.RotateCW(g)))
//
// Rows are indexed first: g[r][c]. Transpose, RotateCW and RotateCCW need a
// rectangular grid and panic otherwise; the other functions also work on
// jagged grids (rows of different lengths).
package grid

import (
	"fmt"
	"iter"
	"strings"
	"unicode/utf8"
)

// Point is a cell position: row R and column C.
type Point struct {
	R, C int
}

// Add returns the component-wise sum p + o. It is handy for stepping in a
// direction: p.Add(grid.Dirs4[i]).
func (p Point) Add(o Point) Point {
	return Point{p.R + o.R, p.C + o.C}
}

// Manhattan returns the Manhattan (taxicab) distance |p.R-o.R| + |p.C-o.C|.
func (p Point) Manhattan(o Point) int {
	return abs(p.R-o.R) + abs(p.C-o.C)
}

// Dirs4 holds the four orthogonal directions in clockwise order starting
// with up: up, right, down, left. Turning right from direction i is
// direction (i+1)%4 and turning left is (i+3)%4.
var Dirs4 = [4]Point{{-1, 0}, {0, 1}, {1, 0}, {0, -1}}

// Dirs8 holds the eight orthogonal and diagonal directions in clockwise
// order starting with up: up, up-right, right, down-right, down, down-left,
// left, up-left.
var Dirs8 = [8]Point{{-1, 0}, {-1, 1}, {0, 1}, {1, 1}, {1, 0}, {1, -1}, {0, -1}, {-1, -1}}

// New returns a rows x cols grid filled with the zero value of T. All rows
// share one backing array (a single allocation), but each row's capacity is
// limited to its length, so appending to a row reallocates it instead of
// overwriting the next row. O(rows*cols).
func New[T any](rows, cols int) [][]T {
	g := make([][]T, rows)
	buf := make([]T, rows*cols)
	for r := range g {
		g[r] = buf[r*cols : (r+1)*cols : (r+1)*cols]
	}
	return g
}

// Fill returns a rows x cols grid with every cell set to v. Like New it uses
// a single backing array. O(rows*cols).
func Fill[T any](rows, cols int, v T) [][]T {
	g := New[T](rows, cols)
	for _, row := range g {
		for c := range row {
			row[c] = v
		}
	}
	return g
}

// Clone returns a deep copy of g: the rows are copied, so modifying the
// clone does not affect g. The elements themselves are copied by assignment.
// Jagged grids are supported. Clone(nil) returns nil. O(cells).
func Clone[T any](g [][]T) [][]T {
	if g == nil {
		return nil
	}
	out := make([][]T, len(g))
	for r, row := range g {
		if row != nil {
			out[r] = append(make([]T, 0, len(row)), row...)
		}
	}
	return out
}

// Transpose returns a new grid t with t[c][r] == g[r][c]. A rows x cols grid
// becomes cols x rows. It panics if g is not rectangular. O(rows*cols).
func Transpose[T any](g [][]T) [][]T {
	rows, cols := dims(g, "Transpose")
	t := New[T](cols, rows)
	for r := range rows {
		for c := range cols {
			t[c][r] = g[r][c]
		}
	}
	return t
}

// RotateCW returns a new grid that is g rotated 90 degrees clockwise. A
// rows x cols grid becomes cols x rows. It panics if g is not rectangular.
// O(rows*cols).
func RotateCW[T any](g [][]T) [][]T {
	rows, cols := dims(g, "RotateCW")
	out := New[T](cols, rows)
	for r := range rows {
		for c := range cols {
			out[c][rows-1-r] = g[r][c]
		}
	}
	return out
}

// RotateCCW returns a new grid that is g rotated 90 degrees counterclockwise.
// A rows x cols grid becomes cols x rows. It panics if g is not rectangular.
// O(rows*cols).
func RotateCCW[T any](g [][]T) [][]T {
	rows, cols := dims(g, "RotateCCW")
	out := New[T](cols, rows)
	for r := range rows {
		for c := range cols {
			out[cols-1-c][r] = g[r][c]
		}
	}
	return out
}

// FromStrings converts lines of text into a byte grid, one row per line,
// e.g. a maze or a LeetCode [][]byte board. The rows are copies, so the grid
// can be modified freely. Lines may have different lengths.
func FromStrings(lines []string) [][]byte {
	g := make([][]byte, len(lines))
	for r, line := range lines {
		g[r] = []byte(line)
	}
	return g
}

// InBounds reports whether (r, c) is a valid cell of g, that is
// 0 <= r < len(g) and 0 <= c < len(g[r]).
func InBounds[T any](g [][]T, r, c int) bool {
	return r >= 0 && r < len(g) && c >= 0 && c < len(g[r])
}

// Neighbors4 returns an iterator over the in-bounds orthogonal neighbors of
// cell (r, c), in Dirs4 order (up, right, down, left):
//
//	for nr, nc := range grid.Neighbors4(g, r, c) { ... }
func Neighbors4[T any](g [][]T, r, c int) iter.Seq2[int, int] {
	return neighbors(g, r, c, Dirs4[:])
}

// Neighbors8 returns an iterator over the in-bounds orthogonal and diagonal
// neighbors of cell (r, c), in Dirs8 order (clockwise from up).
func Neighbors8[T any](g [][]T, r, c int) iter.Seq2[int, int] {
	return neighbors(g, r, c, Dirs8[:])
}

func neighbors[T any](g [][]T, r, c int, dirs []Point) iter.Seq2[int, int] {
	return func(yield func(int, int) bool) {
		for _, d := range dirs {
			nr, nc := r+d.R, c+d.C
			if InBounds(g, nr, nc) && !yield(nr, nc) {
				return
			}
		}
	}
}

// Cells returns an iterator over every cell of g in row-major order, yielding
// its position and value.
func Cells[T any](g [][]T) iter.Seq2[Point, T] {
	return func(yield func(Point, T) bool) {
		for r, row := range g {
			for c, v := range row {
				if !yield(Point{r, c}, v) {
					return
				}
			}
		}
	}
}

// Find returns the position of the first cell equal to v in row-major order,
// and whether one was found. O(rows*cols).
func Find[T comparable](g [][]T, v T) (Point, bool) {
	for p, x := range Cells(g) {
		if x == v {
			return p, true
		}
	}
	return Point{}, false
}

// String formats g for debugging, one line per row with no trailing newline.
// A [][]byte or [][]rune grid is printed as characters with no separators,
// e.g. "#.#\n..#". Any other grid is printed with its cells formatted by
// fmt.Sprint, right-aligned in columns separated by a space:
//
//	 1 20  3
//	40  5  6
func String[T any](g [][]T) string {
	var sb strings.Builder
	switch g := any(g).(type) {
	case [][]byte:
		for r, row := range g {
			if r > 0 {
				sb.WriteByte('\n')
			}
			sb.Write(row)
		}
		return sb.String()
	case [][]rune:
		for r, row := range g {
			if r > 0 {
				sb.WriteByte('\n')
			}
			sb.WriteString(string(row))
		}
		return sb.String()
	}

	cells := make([][]string, len(g))
	var widths []int
	for r, row := range g {
		cells[r] = make([]string, len(row))
		for c, v := range row {
			s := fmt.Sprint(v)
			cells[r][c] = s
			if c == len(widths) {
				widths = append(widths, 0)
			}
			widths[c] = max(widths[c], utf8.RuneCountInString(s))
		}
	}
	for r, row := range cells {
		if r > 0 {
			sb.WriteByte('\n')
		}
		for c, s := range row {
			if c > 0 {
				sb.WriteByte(' ')
			}
			sb.WriteString(strings.Repeat(" ", widths[c]-utf8.RuneCountInString(s)))
			sb.WriteString(s)
		}
	}
	return sb.String()
}

// dims returns the dimensions of g and panics if g is not rectangular.
func dims[T any](g [][]T, fn string) (rows, cols int) {
	if len(g) == 0 {
		return 0, 0
	}
	rows, cols = len(g), len(g[0])
	for r, row := range g {
		if len(row) != cols {
			panic(fmt.Sprintf("grid: %s requires a rectangular grid, but row %d has length %d, want %d", fn, r, len(row), cols))
		}
	}
	return rows, cols
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
