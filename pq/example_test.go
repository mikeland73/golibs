package pq_test

import (
	"cmp"
	"fmt"

	"github.com/mikeland73/golibs/pq"
)

func ExampleNewMin() {
	q := pq.NewMin[int]()
	q.Push(5, 1, 4, 2)
	fmt.Println(q.Peek())
	for !q.IsEmpty() {
		fmt.Print(q.Pop(), " ")
	}
	fmt.Println()
	// Output:
	// 1
	// 1 2 4 5
}

func ExampleNew() {
	type job struct {
		name     string
		priority int
	}
	// Highest priority first.
	q := pq.New(func(a, b job) int { return cmp.Compare(b.priority, a.priority) })
	q.Push(job{"email", 1}, job{"deploy", 10}, job{"review", 5})
	for !q.IsEmpty() {
		fmt.Println(q.Pop().name)
	}
	// Output:
	// deploy
	// review
	// email
}

func ExamplePriorityQueue_PushPop() {
	// Keep the 3 largest values seen in a stream using a min-heap of size 3.
	q := pq.NewMin[int]()
	for _, v := range []int{4, 9, 1, 7, 3, 8} {
		if q.Len() < 3 {
			q.Push(v)
		} else {
			q.PushPop(v)
		}
	}
	fmt.Println(q)
	// Output: PriorityQueue[7 8 9]
}

func ExampleNLargest() {
	nums := []int{3, 1, 4, 1, 5, 9, 2, 6}
	fmt.Println(pq.NLargest(nums, 3))
	fmt.Println(pq.NSmallest(nums, 3))
	// Output:
	// [9 6 5]
	// [1 1 2]
}
