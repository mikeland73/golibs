package segtree_test

import (
	"fmt"

	"github.com/mikeland73/golibs/segtree"
)

func Example() {
	t := segtree.NewMin([]int{5, 2, 8, 1, 9})
	fmt.Println(t.Query(0, 3)) // min of 5 2 8
	t.Set(1, 7)
	fmt.Println(t.Query(0, 3)) // min of 5 7 8
	fmt.Println(t, t.Len())

	s := segtree.NewSum([]float64{0.5, 1.5, 2})
	fmt.Println(s.Query(1, 3), s.Query(2, 2))
	// Output:
	// 2
	// 5
	// Tree[5 7 8 1 9] 5
	// 3.5 0
}

// A custom associative operation: range GCD with identity 0.
func ExampleNew() {
	gcd := func(a, b int) int {
		for b != 0 {
			a, b = b, a%b
		}
		return a
	}
	t := segtree.New([]int{12, 18, 24, 7}, 0, gcd)
	fmt.Println(t.Query(0, 3), t.Query(0, 4), t.Query(2, 2))
	// Output:
	// 6 1 0
}
