package linkedlist_test

import (
	"fmt"

	"github.com/mikeland73/golibs/linkedlist"
)

func Example() {
	head := linkedlist.New(1, 2, 3, 4)

	// Reverse the list in place.
	var prev *linkedlist.Node[int]
	for cur := head; cur != nil; {
		next := cur.Next
		cur.Next = prev
		prev, cur = cur, next
	}

	fmt.Println(prev)
	fmt.Println(linkedlist.ToSlice(prev), linkedlist.Len(prev))
	// Output:
	// 4 -> 3 -> 2 -> 1
	// [4 3 2 1] 4
}

func ExampleWithCycle() {
	head := linkedlist.WithCycle(1, 3, 2, 0, -4)
	fmt.Println(head)

	// Floyd's cycle detection finds where the cycle starts.
	slow, fast := head, head
	for fast != nil && fast.Next != nil {
		slow, fast = slow.Next, fast.Next.Next
		if slow == fast {
			break
		}
	}
	for p := head; p != slow; p, slow = p.Next, slow.Next {
	}
	fmt.Println(slow == linkedlist.NodeAt(head, 1))
	// Output:
	// 3 -> 2 -> 0 -> -4 -> ... (cycle to index 1)
	// true
}

func ExampleAll() {
	for v := range linkedlist.All(linkedlist.New("a", "b", "c")) {
		fmt.Print(v)
	}
	fmt.Println()
	// Output: abc
}
