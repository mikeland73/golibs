package fenwick_test

import (
	"fmt"

	"github.com/mikeland73/golibs/fenwick"
)

func Example() {
	t := fenwick.From([]int{5, 2, 9, 1})
	fmt.Println(t.PrefixSum(2))   // 5 + 2
	t.Add(1, 10)                  // [5 12 9 1]
	fmt.Println(t.RangeSum(1, 3)) // 12 + 9
	t.Set(0, 0)                   // [0 12 9 1]
	fmt.Println(t, t.Get(2))
	// Output:
	// 7
	// 21
	// Tree[0 12 9 1] 9
}

// Counting inversions: for each element, count the larger elements seen
// before it using a frequency table indexed by value.
func ExampleTree_PrefixSum() {
	nums := []int{3, 1, 4, 1, 5, 0, 2}
	freq := fenwick.New[int](6) // values are in [0, 6)
	inversions := 0
	for i, x := range nums {
		inversions += i - freq.PrefixSum(x+1) // seen so far minus those <= x
		freq.Add(x, 1)
	}
	fmt.Println(inversions)
	// Output:
	// 11
}

// Finding the k-th smallest element (1-based) of a multiset of small integers.
func ExampleTree_LowerBound() {
	freq := fenwick.New[int](10)
	for _, x := range []int{7, 2, 2, 9, 4} {
		freq.Add(x, 1)
	}
	var sortedVals []int
	for k := 1; k <= freq.PrefixSum(freq.Len()); k++ {
		sortedVals = append(sortedVals, freq.LowerBound(k))
	}
	fmt.Println(sortedVals)
	// Output:
	// [2 2 4 7 9]
}
