package sorted_test

import (
	"fmt"

	"github.com/mikeland73/golibs/sorted"
)

func ExampleMap() {
	m := sorted.NewMap[string, int]()
	m.Put("banana", 3)
	m.Put("apple", 5)
	m.Put("cherry", 7)

	fmt.Println(m)
	k, v, _ := m.Ceiling("b")
	fmt.Println("ceiling(b):", k, v)
	k, _, _ = m.Lower("banana")
	fmt.Println("lower(banana):", k)
	for k, v := range m.Range("apple", "cherry") {
		fmt.Println(k, v)
	}
	// Output:
	// Map{apple: 5, banana: 3, cherry: 7}
	// ceiling(b): banana 3
	// lower(banana): apple
	// apple 5
	// banana 3
}

func ExampleSet() {
	s := sorted.NewSet(10, 30, 20, 10)
	fmt.Println(s, s.Len())
	fmt.Println(s.Floor(25))
	fmt.Println(s.Higher(30))
	fmt.Println("rank of 25:", s.Rank(25), "second smallest:", s.At(1))
	// Output:
	// Set{10, 20, 30} 3
	// 20 true
	// 0 false
	// rank of 25: 2 second smallest: 20
}

// A sliding-window median using List as a sorted multiset.
func ExampleList() {
	nums := []int{1, 3, -1, -3, 5, 3, 6, 7}
	const k = 3
	window := sorted.NewList[int]()
	var medians []int
	for i, x := range nums {
		window.Add(x)
		if i >= k {
			window.Remove(nums[i-k])
		}
		if i >= k-1 {
			medians = append(medians, window.At(k/2))
		}
	}
	fmt.Println(medians)

	l := sorted.NewList(5, 1, 3, 3, 3)
	fmt.Println(l, l.Count(3), l.BisectLeft(3), l.BisectRight(3))
	// Output:
	// [1 -1 -1 3 5 6]
	// List{1, 3, 3, 3, 5} 3 1 4
}
