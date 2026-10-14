package grid_test

import (
	"fmt"

	"github.com/mikeland73/golibs/grid"
)

// Number of islands with an explicit-stack flood fill.
func Example() {
	g := grid.FromStrings([]string{
		"11000",
		"11000",
		"00100",
		"00011",
	})
	islands := 0
	for p, v := range grid.Cells(g) {
		if v != '1' {
			continue
		}
		islands++
		g[p.R][p.C] = '0'
		stack := []grid.Point{p}
		for len(stack) > 0 {
			cur := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			for nr, nc := range grid.Neighbors4(g, cur.R, cur.C) {
				if g[nr][nc] == '1' {
					g[nr][nc] = '0'
					stack = append(stack, grid.Point{R: nr, C: nc})
				}
			}
		}
	}
	fmt.Println(islands)
	// Output: 3
}

func ExampleRotateCW() {
	g := [][]int{
		{1, 2, 3},
		{4, 5, 6},
	}
	fmt.Println(grid.String(grid.RotateCW(g)))
	// Output:
	// 4 1
	// 5 2
	// 6 3
}

func ExampleString() {
	fmt.Println(grid.String(grid.FromStrings([]string{"#.#", ".#."})))
	fmt.Println(grid.String([][]int{{1, 20}, {300, 4}}))
	// Output:
	// #.#
	// .#.
	//   1 20
	// 300  4
}

func ExampleFind() {
	maze := grid.FromStrings([]string{
		"#####",
		"#S.E#",
		"#####",
	})
	start, _ := grid.Find(maze, 'S')
	end, _ := grid.Find(maze, 'E')
	fmt.Println(start, end, start.Manhattan(end))
	// Output: {1 1} {1 3} 2
}

func ExampleNeighbors8() {
	g := grid.New[int](3, 3)
	for nr, nc := range grid.Neighbors8(g, 0, 0) {
		g[nr][nc] = 1
	}
	fmt.Println(grid.String(g))
	// Output:
	// 0 1 0
	// 1 1 0
	// 0 0 0
}
