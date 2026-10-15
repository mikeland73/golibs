package unionfind_test

import (
	"fmt"

	"github.com/mikeland73/golibs/unionfind"
)

func Example() {
	uf := unionfind.New("alice", "bob", "carol", "dave", "erin")
	uf.Union("alice", "bob")
	uf.Union("carol", "dave")
	fmt.Println(uf.Union("bob", "alice")) // already connected
	uf.Union("bob", "dave")

	fmt.Println(uf.Connected("alice", "carol"))
	fmt.Println(uf.Size("alice"), uf.Count())
	fmt.Println(uf.Groups())
	// Output:
	// false
	// true
	// 4 2
	// [[alice bob carol dave] [erin]]
}

// Counting connected components in an edge list; vertices are added lazily by
// Union.
func ExampleUnionFind_Count() {
	edges := [][2]int{{0, 1}, {1, 2}, {3, 4}}
	uf := unionfind.New(0, 1, 2, 3, 4, 5)
	for _, e := range edges {
		uf.Union(e[0], e[1])
	}
	fmt.Println(uf.Count())
	// Output:
	// 3
}
