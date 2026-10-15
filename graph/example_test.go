package graph_test

import (
	"fmt"

	"github.com/mikeland73/golibs/graph"
)

func Example() {
	// LeetCode-style undirected edges.
	adj := graph.AdjListN(5, [][]int{{0, 1}, {0, 2}, {1, 3}, {3, 4}}, false)
	neighbors := func(u int) []int { return adj[u] }

	for node, dist := range graph.BFS(0, neighbors) {
		fmt.Printf("node %d at distance %d\n", node, dist)
	}

	path, ok := graph.ShortestPath(2, 4, neighbors)
	fmt.Println(path, ok)
	// Output:
	// node 0 at distance 0
	// node 1 at distance 1
	// node 2 at distance 1
	// node 3 at distance 2
	// node 4 at distance 3
	// [2 0 1 3 4] true
}

// Implicit graph: minimum knight moves on an infinite board.
func ExampleBFS_implicit() {
	type pos struct{ x, y int }
	knight := func(p pos) []pos {
		moves := []pos{{1, 2}, {2, 1}, {2, -1}, {1, -2}, {-1, -2}, {-2, -1}, {-2, 1}, {-1, 2}}
		out := make([]pos, len(moves))
		for i, m := range moves {
			out[i] = pos{p.x + m.x, p.y + m.y}
		}
		return out
	}
	for p, d := range graph.BFS(pos{0, 0}, knight) {
		if p == (pos{5, 5}) {
			fmt.Println(d)
			break
		}
	}
	// Output: 4
}

func ExampleTopoSort() {
	// Course schedule: an edge a -> b means a must be taken before b.
	prereqs := map[string][]string{
		"intro":      {"algorithms", "systems"},
		"algorithms": {"compilers"},
		"systems":    {"compilers"},
	}
	order, ok := graph.TopoSort([]string{"intro"}, func(c string) []string { return prereqs[c] })
	fmt.Println(order, ok)
	// Output: [intro algorithms systems compilers] true
}

func ExampleDijkstra() {
	// Network delay time: edges [u, v, w].
	times := [][]int{{2, 1, 1}, {2, 3, 1}, {3, 4, 1}, {1, 4, 5}}
	adj := graph.WeightedAdjListN(5, times, true)
	dist, prev := graph.Dijkstra(2, func(u int) []graph.Edge[int, int] { return adj[u] })
	fmt.Println(dist[4], graph.PathTo(prev, 2, 4))
	// Output: 2 [2 3 4]
}

func ExampleConnectedComponents() {
	adj := graph.AdjListN(6, [][]int{{0, 1}, {1, 2}, {3, 4}}, false)
	nodes := []int{0, 1, 2, 3, 4, 5}
	fmt.Println(graph.ConnectedComponents(nodes, func(u int) []int { return adj[u] }))
	// Output: [[0 1 2] [3 4] [5]]
}

func ExampleBFSMulti() {
	// Distance from every cell to the nearest 0.
	grid := [][]int{
		{0, 1, 1},
		{1, 1, 1},
		{1, 1, 0},
	}
	type cell struct{ r, c int }
	var zeros []cell
	for r, row := range grid {
		for c, v := range row {
			if v == 0 {
				zeros = append(zeros, cell{r, c})
			}
		}
	}
	neighbors := func(p cell) []cell {
		var out []cell
		for _, d := range []cell{{-1, 0}, {0, 1}, {1, 0}, {0, -1}} {
			q := cell{p.r + d.r, p.c + d.c}
			if q.r >= 0 && q.r < 3 && q.c >= 0 && q.c < 3 {
				out = append(out, q)
			}
		}
		return out
	}
	for p, d := range graph.BFSMulti(zeros, neighbors) {
		grid[p.r][p.c] = d
	}
	fmt.Println(grid)
	// Output: [[0 1 2] [1 2 1] [2 1 0]]
}
