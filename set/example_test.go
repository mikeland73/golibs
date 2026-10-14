package set_test

import (
	"fmt"

	"github.com/mikeland73/golibs/set"
)

func Example() {
	a := set.New(1, 2, 3, 4)
	b := set.New(3, 4, 5)
	fmt.Println(a.Contains(2), len(a))
	fmt.Println(a.Union(b))
	fmt.Println(a.Intersection(b))
	fmt.Println(a.Difference(b))
	fmt.Println(a.SymmetricDifference(b))
	fmt.Println(set.New(3, 4).IsSubset(a))
	// Output:
	// true 4
	// Set{1, 2, 3, 4, 5}
	// Set{3, 4}
	// Set{1, 2}
	// Set{1, 2, 5}
	// true
}

func ExampleSorted() {
	seen := set.New[string]()
	for _, w := range []string{"go", "is", "fun", "go", "is"} {
		seen.Add(w)
	}
	fmt.Println(set.Sorted(seen))
	// Output: [fun go is]
}
