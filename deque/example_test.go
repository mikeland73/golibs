package deque_test

import (
	"fmt"

	"github.com/mikeland73/golibs/deque"
)

func Example() {
	var d deque.Deque[int] // zero value is ready to use
	d.PushBack(2, 3)
	d.PushFront(1)
	fmt.Println(d.String())
	fmt.Println(d.Front(), d.Back(), d.At(1))
	d.Rotate(1)
	fmt.Println(d.String())
	fmt.Println(d.PopFront(), d.PopBack())
	// Output:
	// Deque[1 2 3]
	// 1 3 2
	// Deque[3 1 2]
	// 3 2
}

// Sliding window maximum using a monotonic deque of indices.
func Example_slidingWindowMax() {
	nums := []int{1, 3, -1, -3, 5, 3, 6, 7}
	k := 3
	var window deque.Deque[int] // indices; nums[window] is decreasing
	var res []int
	for i, x := range nums {
		for !window.IsEmpty() && nums[window.Back()] <= x {
			window.PopBack()
		}
		window.PushBack(i)
		if window.Front() <= i-k {
			window.PopFront()
		}
		if i >= k-1 {
			res = append(res, nums[window.Front()])
		}
	}
	fmt.Println(res)
	// Output: [3 3 5 5 6 7]
}
