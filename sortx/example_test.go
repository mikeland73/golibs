package sortx_test

import (
	"fmt"
	"slices"

	"github.com/mikeland73/golibs/sortx"
)

type Person struct {
	Name string
	Age  int
}

func ExampleSortBy() {
	people := []Person{{"carol", 30}, {"alice", 25}, {"bob", 30}}
	sortx.SortBy(people, func(p Person) int { return p.Age })
	fmt.Println(people)
	// Output:
	// [{alice 25} {carol 30} {bob 30}]
}

func ExampleSortByDesc() {
	words := []string{"go", "python", "c", "rust"}
	sortx.SortByDesc(words, func(w string) int { return len(w) })
	fmt.Println(words)
	// Output:
	// [python rust go c]
}

func ExampleChain() {
	people := []Person{{"carol", 30}, {"alice", 25}, {"bob", 30}, {"dave", 25}}
	// Oldest first, ties broken by name.
	slices.SortFunc(people, sortx.Chain(
		sortx.Desc(sortx.By(func(p Person) int { return p.Age })),
		sortx.By(func(p Person) string { return p.Name }),
	))
	fmt.Println(people)
	// Output:
	// [{bob 30} {carol 30} {alice 25} {dave 25}]
}

func ExampleSorted() {
	xs := []int{3, 1, 2}
	ys := sortx.Sorted(xs)
	fmt.Println(xs, ys)
	// Output:
	// [3 1 2] [1 2 3]
}

func ExampleSortedBy() {
	words := []string{"banana", "Apple", "cherry"}
	fmt.Println(sortx.SortedBy(words, func(w string) int { return len(w) }))
	// Output:
	// [Apple banana cherry]
}

func ExampleArgSort() {
	scores := []int{70, 95, 80}
	for _, i := range sortx.ArgSort(scores) {
		fmt.Print(i, ":", scores[i], " ")
	}
	fmt.Println()
	// Output:
	// 0:70 2:80 1:95
}
