package bisect_test

import (
	"fmt"

	"github.com/mikeland73/golibs/bisect"
)

func Example() {
	s := []int{1, 2, 2, 2, 5}
	fmt.Println(bisect.Left(s, 2), bisect.Right(s, 2))
	fmt.Println(bisect.CountRange(s, 2, 5))
	s = bisect.Insort(s, 3)
	fmt.Println(s)
	// Output:
	// 1 4
	// 3
	// [1 2 2 2 3 5]
}

// Binary search on the answer: the minimum ship capacity needed to ship all
// packages, in order, within the given number of days.
func ExampleFirstTrue() {
	weights := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	days := 5
	canShip := func(capacity int) bool {
		need, load := 1, 0
		for _, w := range weights {
			if w > capacity {
				return false
			}
			if load+w > capacity {
				need, load = need+1, 0
			}
			load += w
		}
		return need <= days
	}
	fmt.Println(bisect.FirstTrue(1, 1000, canShip))
	// Output: 15
}

func ExampleLastTrue() {
	// Integer square root: the largest x with x*x <= n.
	n := 1_000_000_007
	fmt.Println(bisect.LastTrue(0, n+1, func(x int) bool { return x*x <= n }))
	// Output: 31622
}

func ExampleFloat() {
	x := bisect.Float(0, 2, func(x float64) bool { return x*x >= 2 }, 1e-9)
	fmt.Printf("%.6f\n", x)
	// Output: 1.414214
}
