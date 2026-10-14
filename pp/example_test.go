package pp_test

import (
	"os"

	"github.com/mikeland73/golibs/pp"
)

type Point struct {
	X, Y int
}

func ExamplePrint() {
	pp.Print("scores:", map[string]int{"bob": 7, "alice": 9})
	pp.Print([]string{"a", "b"}, &Point{1, 2}, []rune("hi"))
	var empty []int
	pp.Print(empty, map[int]struct{}{3: {}, 1: {}})
	// Output:
	// scores: {"alice": 9, "bob": 7}
	// ["a", "b"] Point{X: 1, Y: 2} ['h', 'i']
	// [] {1, 3}
}

func ExampleDebug() {
	pp.Output = os.Stdout // Debug writes to stderr by default.
	defer func() { pp.Output = os.Stderr }()

	i := 3
	xs := []int{1, 2, 3}
	pp.Debug("loop", i, xs[:i-1])
	// Output:
	// [example_test.go:30] loop, i = 3, xs[:i-1] = [1, 2]
}

func ExampleGrid() {
	pp.Grid([][]byte{
		[]byte("#.."),
		[]byte(".#."),
		[]byte("..#"),
	})
	pp.Grid([][]int{{1, 20, 3}, {400, 5, 60}})
	// Output:
	// #..
	// .#.
	// ..#
	//   1 20  3
	// 400  5 60
}

func ExampleJSON() {
	pp.JSON(map[string]any{"name": "go", "tags": []string{"fast"}})
	// Output:
	// {
	//   "name": "go",
	//   "tags": [
	//     "fast"
	//   ]
	// }
}
