package memo_test

import (
	"fmt"

	"github.com/mikeland73/golibs/memo"
)

func ExampleRecursive() {
	fib := memo.Recursive(func(fib func(int) int, n int) int {
		if n < 2 {
			return n
		}
		return fib(n-1) + fib(n-2)
	})
	fmt.Println(fib(90))
	// Output:
	// 2880067194370816120
}

func ExampleRecursive2() {
	// Unique paths from the top-left to the bottom-right of a grid, moving only
	// right or down, avoiding obstacles ('#').
	grid := []string{
		"...",
		".#.",
		"...",
	}
	rows, cols := len(grid), len(grid[0])
	paths := memo.Recursive2(func(paths func(int, int) int, r, c int) int {
		if r >= rows || c >= cols || grid[r][c] == '#' {
			return 0
		}
		if r == rows-1 && c == cols-1 {
			return 1
		}
		return paths(r+1, c) + paths(r, c+1)
	})
	fmt.Println(paths(0, 0))
	// Output:
	// 2
}

func ExampleFunc() {
	calls := 0
	slowSquare := func(x int) int { calls++; return x * x }
	square := memo.Func(slowSquare)
	fmt.Println(square(12), square(12), square(12))
	fmt.Println("calls:", calls)
	// Output:
	// 144 144 144
	// calls: 1
}

func ExampleRecursive3() {
	// Number of ways to make `amount` from coins[i:] using at most `left` coins.
	coins := []int{1, 2, 5}
	ways := memo.Recursive3(func(ways func(int, int, int) int, i, amount, left int) int {
		if amount == 0 {
			return 1
		}
		if i == len(coins) || amount < 0 || left == 0 {
			return 0
		}
		return ways(i, amount-coins[i], left-1) + ways(i+1, amount, left)
	})
	fmt.Println(ways(0, 5, 5), ways(0, 5, 2))
	// Output:
	// 4 1
}
