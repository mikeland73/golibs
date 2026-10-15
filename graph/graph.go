// Package graph provides function-based graph algorithms: BFS, DFS,
// shortest paths, topological sort, Dijkstra and connected components.
//
// Graphs are described by a neighbors function rather than a concrete type,
// so the same algorithms work on explicit adjacency lists and on implicit
// graphs such as grid cells, chess moves or word ladders:
//
//	adj := graph.AdjListN(n, edges, false) // LeetCode-style [][]int edges
//	for node, dist := range graph.BFS(0, func(u int) []int { return adj[u] }) {
//		fmt.Println(node, dist)
//	}
//
//	type cell struct{ r, c int }
//	path, ok := graph.ShortestPath(cell{0, 0}, cell{7, 7}, knightMoves)
//
// Node types must be comparable because visited sets are kept in maps.
// In the complexities below, V is the number of nodes reached and E the
// number of edges examined; the cost of calling neighbors is not included.
// None of the algorithms recurse, so very deep graphs are fine.
package graph

import (
	"cmp"
	"iter"
	"strconv"

	"github.com/mikeland73/golibs/mathx"
	"github.com/mikeland73/golibs/pq"
)

// Edge is a weighted edge to node To, as returned by the neighbors function
// passed to Dijkstra.
type Edge[T comparable, W mathx.Number] struct {
	To     T
	Weight W
}

// AdjList builds an adjacency list from a list of edges. If directed is false
// each edge is added in both directions. Neighbors appear in edge order.
// Nodes that only appear as edge targets of a directed graph have no entry.
// O(E).
func AdjList[T comparable](edges [][2]T, directed bool) map[T][]T {
	adj := make(map[T][]T)
	for _, e := range edges {
		adj[e[0]] = append(adj[e[0]], e[1])
		if !directed {
			adj[e[1]] = append(adj[e[1]], e[0])
		}
	}
	return adj
}

// AdjListN builds an adjacency list for nodes 0..n-1 from LeetCode-style
// edges such as [][]int{{0, 1}, {1, 2}}. Only the first two columns of each
// edge are used, so weighted edges [u, v, w] are accepted and the weight is
// ignored. If directed is false each edge is added in both directions.
// It panics if an edge has fewer than two columns or a node is out of range.
// O(n + E).
func AdjListN(n int, edges [][]int, directed bool) [][]int {
	adj := make([][]int, n)
	for i, e := range edges {
		if len(e) < 2 {
			panic("graph: AdjListN edge " + strconv.Itoa(i) + " has fewer than 2 columns")
		}
		u, v := e[0], e[1]
		adj[u] = append(adj[u], v)
		if !directed {
			adj[v] = append(adj[v], u)
		}
	}
	return adj
}

// WeightedAdjListN builds a weighted adjacency list for nodes 0..n-1 from
// LeetCode-style edges [u, v, w], for use with Dijkstra:
//
//	adj := graph.WeightedAdjListN(n, times, true)
//	dist, _ := graph.Dijkstra(k, func(u int) []graph.Edge[int, int] { return adj[u] })
//
// If directed is false each edge is added in both directions. It panics if an
// edge has fewer than three columns or a node is out of range. O(n + E).
func WeightedAdjListN(n int, edges [][]int, directed bool) [][]Edge[int, int] {
	adj := make([][]Edge[int, int], n)
	for i, e := range edges {
		if len(e) < 3 {
			panic("graph: WeightedAdjListN edge " + strconv.Itoa(i) + " has fewer than 3 columns")
		}
		u, v, w := e[0], e[1], e[2]
		adj[u] = append(adj[u], Edge[int, int]{To: v, Weight: w})
		if !directed {
			adj[v] = append(adj[v], Edge[int, int]{To: u, Weight: w})
		}
	}
	return adj
}

// BFS returns an iterator over the nodes reachable from start in
// breadth-first order, yielding each node once together with its distance
// (number of edges) from start. start itself is yielded first with distance 0.
// Breaking out of the loop stops the search early. O(V + E).
func BFS[T comparable](start T, neighbors func(T) []T) iter.Seq2[T, int] {
	return BFSMulti([]T{start}, neighbors)
}

// BFSMulti is a multi-source BFS: all starts are at distance 0 and every other
// node is yielded with its distance to the nearest start. This is the standard
// technique for problems like "rotting oranges" or "distance to nearest 0".
// Duplicate starts are yielded once. O(V + E).
func BFSMulti[T comparable](starts []T, neighbors func(T) []T) iter.Seq2[T, int] {
	return func(yield func(T, int) bool) {
		dist := make(map[T]int, len(starts))
		queue := make([]T, 0, len(starts))
		for _, s := range starts {
			if _, seen := dist[s]; !seen {
				dist[s] = 0
				queue = append(queue, s)
			}
		}
		for head := 0; head < len(queue); head++ {
			u := queue[head]
			d := dist[u]
			if !yield(u, d) {
				return
			}
			for _, v := range neighbors(u) {
				if _, seen := dist[v]; !seen {
					dist[v] = d + 1
					queue = append(queue, v)
				}
			}
		}
	}
}

// Distances returns the BFS distance (number of edges) from start to every
// reachable node, including start itself at distance 0. Unreachable nodes
// are absent from the map. O(V + E).
func Distances[T comparable](start T, neighbors func(T) []T) map[T]int {
	dist := make(map[T]int)
	for u, d := range BFS(start, neighbors) {
		dist[u] = d
	}
	return dist
}

// ShortestPath returns a shortest path (fewest edges) from start to goal,
// including both endpoints, and true; or nil and false if goal is not
// reachable. If start == goal the path is []T{start}. The search stops as soon
// as goal is discovered. O(V + E).
func ShortestPath[T comparable](start, goal T, neighbors func(T) []T) ([]T, bool) {
	if start == goal {
		return []T{start}, true
	}
	prev := map[T]T{start: start}
	queue := []T{start}
	for head := 0; head < len(queue); head++ {
		u := queue[head]
		for _, v := range neighbors(u) {
			if _, seen := prev[v]; seen {
				continue
			}
			prev[v] = u
			if v == goal {
				return PathTo(prev, start, goal), true
			}
			queue = append(queue, v)
		}
	}
	return nil, false
}

// DFS returns an iterator over the nodes reachable from start in depth-first
// preorder, visiting each node once. Neighbors are explored in the order
// returned by neighbors, so the order matches the classic recursive DFS.
// It uses an explicit stack, so deep graphs cannot overflow the call stack.
// Breaking out of the loop stops the search early. O(V + E) time; the stack
// may hold O(E) entries.
func DFS[T comparable](start T, neighbors func(T) []T) iter.Seq[T] {
	return func(yield func(T) bool) {
		visited := make(map[T]bool)
		stack := []T{start}
		for len(stack) > 0 {
			u := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if visited[u] {
				continue
			}
			visited[u] = true
			if !yield(u) {
				return
			}
			nbrs := neighbors(u)
			for i := len(nbrs) - 1; i >= 0; i-- {
				if !visited[nbrs[i]] {
					stack = append(stack, nbrs[i])
				}
			}
		}
	}
}

// TopoSort returns a topological order of a directed graph using Kahn's
// algorithm: every node appears before all nodes it has an edge to. nodes
// should list every node of the graph; neighbors that are not in nodes are
// appended to the node set in the order they are discovered. Duplicate nodes
// are ignored.
//
// The result is deterministic: among nodes that are ready at the same time,
// those earlier in nodes (or discovered earlier) come first. If the graph
// contains a cycle TopoSort returns nil, false. neighbors is called once per
// node. O(V + E).
func TopoSort[T comparable](nodes []T, neighbors func(T) []T) ([]T, bool) {
	index := make(map[T]int, len(nodes)) // node -> position in all
	var all []T
	add := func(u T) {
		if _, ok := index[u]; !ok {
			index[u] = len(all)
			all = append(all, u)
		}
	}
	for _, u := range nodes {
		add(u)
	}
	var adj [][]int
	for i := 0; i < len(all); i++ { // all grows as new nodes are discovered
		nbrs := neighbors(all[i])
		out := make([]int, len(nbrs))
		for j, v := range nbrs {
			add(v)
			out[j] = index[v]
		}
		adj = append(adj, out)
	}
	indeg := make([]int, len(all))
	for _, out := range adj {
		for _, v := range out {
			indeg[v]++
		}
	}
	order := make([]T, 0, len(all))
	queue := make([]int, 0, len(all))
	for i, d := range indeg {
		if d == 0 {
			queue = append(queue, i)
		}
	}
	for head := 0; head < len(queue); head++ {
		u := queue[head]
		order = append(order, all[u])
		for _, v := range adj[u] {
			indeg[v]--
			if indeg[v] == 0 {
				queue = append(queue, v)
			}
		}
	}
	if len(order) != len(all) {
		return nil, false
	}
	return order, true
}

// Dijkstra computes single-source shortest paths from start in a graph with
// non-negative edge weights. dist maps every reachable node to its distance
// from start (dist[start] == 0); prev maps every reachable node except start
// to its predecessor on a shortest path. Use PathTo to reconstruct a path.
// Unreachable nodes are absent from both maps. Results are undefined if any
// weight is negative.
//
// It uses a binary heap with lazy deletion: O((V + E) log E).
func Dijkstra[T comparable, W mathx.Number](start T, neighbors func(T) []Edge[T, W]) (dist map[T]W, prev map[T]T) {
	var zero W
	dist = map[T]W{start: zero}
	prev = make(map[T]T)
	done := make(map[T]bool)
	h := pq.New(func(a, b distItem[T, W]) int { return cmp.Compare(a.dist, b.dist) })
	h.Push(distItem[T, W]{node: start, dist: zero})
	for !h.IsEmpty() {
		it := h.Pop()
		u := it.node
		if done[u] {
			continue
		}
		done[u] = true
		for _, e := range neighbors(u) {
			if done[e.To] {
				continue
			}
			nd := it.dist + e.Weight
			if old, ok := dist[e.To]; !ok || nd < old {
				dist[e.To] = nd
				prev[e.To] = u
				h.Push(distItem[T, W]{node: e.To, dist: nd})
			}
		}
	}
	return dist, prev
}

// PathTo reconstructs the path from start to goal (both included) from a
// predecessor map such as the one returned by Dijkstra. It returns
// []T{start} if start == goal, and nil if goal is unreachable or the
// predecessor chain does not lead back to start. O(path length).
func PathTo[T comparable](prev map[T]T, start, goal T) []T {
	if start == goal {
		return []T{start}
	}
	if _, ok := prev[goal]; !ok {
		return nil
	}
	path := []T{goal}
	for u := goal; u != start; {
		p, ok := prev[u]
		if !ok || len(path) > len(prev) {
			return nil // chain does not lead back to start (or loops)
		}
		path = append(path, p)
		u = p
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

// ConnectedComponents returns the connected components of an undirected
// graph. Each component lists its nodes in BFS order from its first node, and
// components are ordered by their first node's position in nodes. Neighbors
// that are not in nodes are still included in their component. The neighbors
// function must be symmetric (v in neighbors(u) iff u in neighbors(v)); for a
// directed graph the result is the set of nodes reachable from each
// not-yet-visited node, not the strongly connected components. O(V + E).
func ConnectedComponents[T comparable](nodes []T, neighbors func(T) []T) [][]T {
	seen := make(map[T]bool, len(nodes))
	var comps [][]T
	for _, s := range nodes {
		if seen[s] {
			continue
		}
		seen[s] = true
		comp := []T{s}
		for head := 0; head < len(comp); head++ {
			for _, v := range neighbors(comp[head]) {
				if !seen[v] {
					seen[v] = true
					comp = append(comp, v)
				}
			}
		}
		comps = append(comps, comp)
	}
	return comps
}

type distItem[T comparable, W mathx.Number] struct {
	node T
	dist W
}
