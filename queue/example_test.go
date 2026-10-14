package queue_test

import (
	"fmt"

	"github.com/mikeland73/golibs/queue"
)

func Example() {
	var q queue.Queue[string] // zero value is ready to use
	q.Push("a", "b", "c")
	fmt.Println(q.String())
	fmt.Println(q.Peek())
	fmt.Println(q.Pop(), q.Pop())
	fmt.Println(q.Len())
	// Output:
	// Queue[a b c]
	// a
	// a b
	// 1
}

func Example_bfs() {
	graph := map[int][]int{1: {2, 3}, 2: {4}, 3: {4}, 4: {}}
	dist := map[int]int{1: 0}
	q := queue.New(1)
	for !q.IsEmpty() {
		node := q.Pop()
		for _, next := range graph[node] {
			if _, seen := dist[next]; !seen {
				dist[next] = dist[node] + 1
				q.Push(next)
			}
		}
	}
	fmt.Println(dist)
	// Output: map[1:0 2:1 3:1 4:2]
}
