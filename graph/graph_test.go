package graph

import (
	"math"
	"math/rand/v2"
	"slices"
	"sort"
	"testing"
)

const inf = math.MaxInt / 4

func randomEdges(r *rand.Rand, n, m int, weighted bool) [][]int {
	edges := make([][]int, m)
	for i := range edges {
		u, v := r.IntN(n), r.IntN(n)
		if weighted {
			edges[i] = []int{u, v, r.IntN(10)}
		} else {
			edges[i] = []int{u, v}
		}
	}
	return edges
}

// floydWarshall returns all-pairs shortest distances; weight 1 if w == nil.
func floydWarshall(n int, adj [][]Edge[int, int]) [][]int {
	d := make([][]int, n)
	for i := range d {
		d[i] = make([]int, n)
		for j := range d[i] {
			d[i][j] = inf
		}
		d[i][i] = 0
	}
	for u, es := range adj {
		for _, e := range es {
			d[u][e.To] = min(d[u][e.To], e.Weight)
		}
	}
	for k := range n {
		for i := range n {
			for j := range n {
				if d[i][k]+d[k][j] < d[i][j] {
					d[i][j] = d[i][k] + d[k][j]
				}
			}
		}
	}
	return d
}

func unitWeights(adj [][]int) [][]Edge[int, int] {
	w := make([][]Edge[int, int], len(adj))
	for u, vs := range adj {
		for _, v := range vs {
			w[u] = append(w[u], Edge[int, int]{v, 1})
		}
	}
	return w
}

func TestAdjList(t *testing.T) {
	g := AdjList([][2]string{{"a", "b"}, {"b", "c"}, {"a", "c"}}, false)
	want := map[string][]string{"a": {"b", "c"}, "b": {"a", "c"}, "c": {"b", "a"}}
	for k, v := range want {
		if !slices.Equal(g[k], v) {
			t.Errorf("AdjList[%q] = %v, want %v", k, g[k], v)
		}
	}
	if len(g) != 3 {
		t.Errorf("len = %d, want 3", len(g))
	}
	d := AdjList([][2]int{{1, 2}, {2, 3}}, true)
	if len(d) != 2 || !slices.Equal(d[1], []int{2}) || d[3] != nil {
		t.Errorf("directed AdjList = %v", d)
	}
	if got := AdjList[int](nil, false); len(got) != 0 {
		t.Errorf("empty AdjList = %v", got)
	}
}

func TestAdjListN(t *testing.T) {
	adj := AdjListN(4, [][]int{{0, 1, 99}, {1, 2}, {0, 2}}, false)
	want := [][]int{{1, 2}, {0, 2}, {1, 0}, nil}
	for i := range want {
		if !slices.Equal(adj[i], want[i]) {
			t.Errorf("adj[%d] = %v, want %v", i, adj[i], want[i])
		}
	}
	dir := AdjListN(3, [][]int{{0, 1}, {1, 2}}, true)
	if !slices.Equal(dir[0], []int{1}) || len(dir[2]) != 0 {
		t.Errorf("directed = %v", dir)
	}
	assertPanics(t, "short edge", func() { AdjListN(2, [][]int{{0}}, true) })
}

func TestWeightedAdjListN(t *testing.T) {
	adj := WeightedAdjListN(3, [][]int{{0, 1, 5}, {1, 2, 7}}, false)
	if want := []Edge[int, int]{{1, 5}}; !slices.Equal(adj[0], want) {
		t.Errorf("adj[0] = %v, want %v", adj[0], want)
	}
	if want := []Edge[int, int]{{0, 5}, {2, 7}}; !slices.Equal(adj[1], want) {
		t.Errorf("adj[1] = %v, want %v", adj[1], want)
	}
	dir := WeightedAdjListN(3, [][]int{{0, 1, 5}}, true)
	if len(dir[1]) != 0 {
		t.Errorf("directed adj[1] = %v", dir[1])
	}
	assertPanics(t, "short edge", func() { WeightedAdjListN(2, [][]int{{0, 1}}, true) })
}

func TestBFSRandomVsFloyd(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 1))
	for range 300 {
		n := 1 + r.IntN(12)
		directed := r.IntN(2) == 0
		adj := AdjListN(n, randomEdges(r, n, r.IntN(3*n), false), directed)
		nb := func(u int) []int { return adj[u] }
		fw := floydWarshall(n, unitWeights(adj))
		src := r.IntN(n)

		dist := Distances(src, nb)
		prevD := 0
		count := 0
		for u, d := range BFS(src, nb) {
			count++
			if d < prevD {
				t.Fatalf("BFS distances not non-decreasing")
			}
			prevD = d
			if fw[src][u] != d {
				t.Fatalf("BFS dist %d->%d = %d, want %d", src, u, d, fw[src][u])
			}
		}
		for v := range n {
			d, ok := dist[v]
			if ok != (fw[src][v] < inf) || (ok && d != fw[src][v]) {
				t.Fatalf("Distances %d->%d = %d,%v want %d", src, v, d, ok, fw[src][v])
			}
		}
		if count != len(dist) {
			t.Fatalf("BFS yielded %d nodes, Distances has %d", count, len(dist))
		}

		goal := r.IntN(n)
		path, ok := ShortestPath(src, goal, nb)
		if ok != (fw[src][goal] < inf) {
			t.Fatalf("ShortestPath reachability mismatch")
		}
		if ok {
			if len(path)-1 != fw[src][goal] || path[0] != src || path[len(path)-1] != goal {
				t.Fatalf("ShortestPath %d->%d = %v, want length %d", src, goal, path, fw[src][goal])
			}
			for i := 1; i < len(path); i++ {
				if !slices.Contains(adj[path[i-1]], path[i]) {
					t.Fatalf("ShortestPath uses non-edge %d->%d", path[i-1], path[i])
				}
			}
		} else if path != nil {
			t.Fatalf("ShortestPath returned non-nil path for unreachable goal")
		}

		// Multi-source vs min over sources.
		starts := []int{r.IntN(n), r.IntN(n), r.IntN(n)}
		for u, d := range BFSMulti(starts, nb) {
			want := inf
			for _, s := range starts {
				want = min(want, fw[s][u])
			}
			if d != want {
				t.Fatalf("BFSMulti dist to %d = %d, want %d", u, d, want)
			}
		}
	}
}

func TestBFSEarlyBreakAndInfinite(t *testing.T) {
	// Implicit infinite graph: integers with edges x -> x+1, x -> 2x.
	nb := func(x int) []int { return []int{x + 1, 2 * x} }
	var got []int
	for u, d := range BFS(1, nb) {
		if d > 2 {
			break
		}
		got = append(got, u)
	}
	if want := []int{1, 2, 3, 4}; !slices.Equal(got, want) {
		t.Errorf("BFS = %v, want %v", got, want)
	}
	n := 0
	for range DFS(0, func(x int) []int { return []int{x + 1} }) {
		n++
		if n == 1000 {
			break
		}
	}
	if n != 1000 {
		t.Errorf("DFS early break count = %d", n)
	}
}

func TestBFSMultiDuplicatesAndEmpty(t *testing.T) {
	adj := AdjListN(3, [][]int{{0, 1}, {1, 2}}, false)
	nb := func(u int) []int { return adj[u] }
	var nodes, dists []int
	for u, d := range BFSMulti([]int{0, 0, 2}, nb) {
		nodes = append(nodes, u)
		dists = append(dists, d)
	}
	if !slices.Equal(nodes, []int{0, 2, 1}) || !slices.Equal(dists, []int{0, 0, 1}) {
		t.Errorf("BFSMulti = %v %v", nodes, dists)
	}
	for range BFSMulti(nil, nb) {
		t.Fatal("BFSMulti(nil) yielded")
	}
}

func TestShortestPathSelfAndGrid(t *testing.T) {
	type cell struct{ r, c int }
	grid := []string{
		"..#.",
		".##.",
		"....",
	}
	nb := func(p cell) []cell {
		var out []cell
		for _, d := range []cell{{-1, 0}, {0, 1}, {1, 0}, {0, -1}} {
			q := cell{p.r + d.r, p.c + d.c}
			if q.r >= 0 && q.r < len(grid) && q.c >= 0 && q.c < len(grid[0]) && grid[q.r][q.c] == '.' {
				out = append(out, q)
			}
		}
		return out
	}
	path, ok := ShortestPath(cell{0, 0}, cell{0, 3}, nb)
	if !ok || len(path) != 8 {
		t.Errorf("grid path = %v, %v; want length 8", path, ok)
	}
	path, ok = ShortestPath(cell{1, 0}, cell{1, 0}, nb)
	if !ok || len(path) != 1 {
		t.Errorf("self path = %v, %v", path, ok)
	}
	_, ok = ShortestPath(cell{0, 0}, cell{0, 2}, nb) // wall
	if ok {
		t.Errorf("path to wall should not exist")
	}
}

func recursiveDFS(u int, adj [][]int, seen []bool, out *[]int) {
	seen[u] = true
	*out = append(*out, u)
	for _, v := range adj[u] {
		if !seen[v] {
			recursiveDFS(v, adj, seen, out)
		}
	}
}

func TestDFSMatchesRecursive(t *testing.T) {
	r := rand.New(rand.NewPCG(2, 2))
	for range 500 {
		n := 1 + r.IntN(15)
		adj := AdjListN(n, randomEdges(r, n, r.IntN(3*n), false), r.IntN(2) == 0)
		src := r.IntN(n)
		var want []int
		recursiveDFS(src, adj, make([]bool, n), &want)
		got := slices.Collect(DFS(src, func(u int) []int { return adj[u] }))
		if !slices.Equal(got, want) {
			t.Fatalf("DFS = %v, want %v (adj %v)", got, want, adj)
		}
	}
}

func TestDFSDeepChain(t *testing.T) {
	const n = 200000
	count := 0
	for range DFS(0, func(u int) []int {
		if u+1 < n {
			return []int{u + 1}
		}
		return nil
	}) {
		count++
	}
	if count != n {
		t.Errorf("DFS chain visited %d, want %d", count, n)
	}
}

func hasCycleNaive(n int, adj [][]int) bool {
	color := make([]int, n)
	var visit func(u int) bool
	visit = func(u int) bool {
		color[u] = 1
		for _, v := range adj[u] {
			if color[v] == 1 || (color[v] == 0 && visit(v)) {
				return true
			}
		}
		color[u] = 2
		return false
	}
	for u := range n {
		if color[u] == 0 && visit(u) {
			return true
		}
	}
	return false
}

func TestTopoSortRandom(t *testing.T) {
	r := rand.New(rand.NewPCG(3, 3))
	for range 1000 {
		n := 1 + r.IntN(10)
		var edges [][]int
		dag := r.IntN(2) == 0
		for range r.IntN(2 * n) {
			u, v := r.IntN(n), r.IntN(n)
			if dag {
				if u == v {
					continue
				}
				u, v = min(u, v), max(u, v)
			}
			edges = append(edges, []int{u, v})
		}
		adj := AdjListN(n, edges, true)
		nodes := r.Perm(n)
		order, ok := TopoSort(nodes, func(u int) []int { return adj[u] })
		cyc := hasCycleNaive(n, adj)
		if ok == cyc {
			t.Fatalf("TopoSort ok=%v but cycle=%v, adj=%v", ok, cyc, adj)
		}
		if !ok {
			if order != nil {
				t.Fatalf("TopoSort returned non-nil order on cycle")
			}
			continue
		}
		if len(order) != n {
			t.Fatalf("TopoSort len %d, want %d", len(order), n)
		}
		pos := make([]int, n)
		for i, u := range order {
			pos[u] = i
		}
		for u := range n {
			for _, v := range adj[u] {
				if pos[u] >= pos[v] {
					t.Fatalf("TopoSort order %v violates edge %d->%d", order, u, v)
				}
			}
		}
	}
}

func TestTopoSortDeterministicAndDiscovered(t *testing.T) {
	g := map[string][]string{
		"shirt": {"tie", "belt"},
		"tie":   {"jacket"},
		"pants": {"shoes", "belt"},
		"belt":  {"jacket"},
	}
	nb := func(u string) []string { return g[u] }
	// "jacket" and "shoes" are not listed but are discovered.
	order, ok := TopoSort([]string{"shirt", "pants", "shirt"}, nb)
	want := []string{"shirt", "pants", "tie", "shoes", "belt", "jacket"}
	if !ok || !slices.Equal(order, want) {
		t.Errorf("TopoSort = %v, %v; want %v", order, ok, want)
	}
	order, ok = TopoSort([]string{}, nb)
	if !ok || len(order) != 0 {
		t.Errorf("empty TopoSort = %v, %v", order, ok)
	}
	_, ok = TopoSort([]int{1}, func(int) []int { return []int{1} })
	if ok {
		t.Errorf("self loop should be a cycle")
	}
}

func TestDijkstraRandomVsFloyd(t *testing.T) {
	r := rand.New(rand.NewPCG(4, 4))
	for range 300 {
		n := 1 + r.IntN(12)
		adj := WeightedAdjListN(n, randomEdges(r, n, r.IntN(4*n), true), r.IntN(2) == 0)
		fw := floydWarshall(n, adj)
		src := r.IntN(n)
		dist, prev := Dijkstra(src, func(u int) []Edge[int, int] { return adj[u] })
		for v := range n {
			d, ok := dist[v]
			if ok != (fw[src][v] < inf) || (ok && d != fw[src][v]) {
				t.Fatalf("Dijkstra %d->%d = %d,%v want %d", src, v, d, ok, fw[src][v])
			}
			if !ok {
				if PathTo(prev, src, v) != nil {
					t.Fatalf("PathTo unreachable should be nil")
				}
				continue
			}
			path := PathTo(prev, src, v)
			if path[0] != src || path[len(path)-1] != v {
				t.Fatalf("PathTo endpoints %v", path)
			}
			total := 0
			for i := 1; i < len(path); i++ {
				best := inf
				for _, e := range adj[path[i-1]] {
					if e.To == path[i] {
						best = min(best, e.Weight)
					}
				}
				if best == inf {
					t.Fatalf("PathTo uses non-edge")
				}
				total += best
			}
			if total != d {
				t.Fatalf("PathTo weight %d, want %d (path %v)", total, d, path)
			}
		}
		if _, ok := prev[src]; ok && dist[src] == 0 {
			// prev[src] may only be set if src was improved, which can't happen.
			t.Fatalf("prev contains start")
		}
	}
}

func TestDijkstraFloatAndImplicit(t *testing.T) {
	type E = Edge[string, float64]
	g := map[string][]E{
		"a": {{"b", 0.5}, {"c", 2.0}},
		"b": {{"c", 0.25}},
	}
	dist, prev := Dijkstra("a", func(u string) []E { return g[u] })
	if dist["c"] != 0.75 {
		t.Errorf("dist[c] = %v, want 0.75", dist["c"])
	}
	if got := PathTo(prev, "a", "c"); !slices.Equal(got, []string{"a", "b", "c"}) {
		t.Errorf("PathTo = %v", got)
	}
	if got := PathTo(prev, "a", "a"); !slices.Equal(got, []string{"a"}) {
		t.Errorf("PathTo self = %v", got)
	}
	if got := PathTo(prev, "a", "zzz"); got != nil {
		t.Errorf("PathTo missing = %v", got)
	}
}

func TestPathToBrokenChain(t *testing.T) {
	prev := map[int]int{3: 2, 2: 1}
	if got := PathTo(prev, 0, 3); got != nil {
		t.Errorf("PathTo broken chain = %v, want nil", got)
	}
}

type unionFind []int

func (u unionFind) find(x int) int {
	for u[x] != x {
		x = u[x]
	}
	return x
}

func TestConnectedComponentsRandom(t *testing.T) {
	r := rand.New(rand.NewPCG(5, 5))
	for range 500 {
		n := 1 + r.IntN(15)
		edges := randomEdges(r, n, r.IntN(2*n), false)
		adj := AdjListN(n, edges, false)
		uf := make(unionFind, n)
		for i := range uf {
			uf[i] = i
		}
		for _, e := range edges {
			uf[uf.find(e[0])] = uf.find(e[1])
		}
		nodes := make([]int, n)
		for i := range nodes {
			nodes[i] = i
		}
		comps := ConnectedComponents(nodes, func(u int) []int { return adj[u] })
		seen := make(map[int]bool)
		roots := make(map[int]bool)
		for _, c := range comps {
			root := uf.find(c[0])
			if roots[root] {
				t.Fatalf("component split: %v", comps)
			}
			roots[root] = true
			for _, u := range c {
				if seen[u] {
					t.Fatalf("node %d in two components", u)
				}
				seen[u] = true
				if uf.find(u) != root {
					t.Fatalf("component %v mixes roots", c)
				}
			}
		}
		if len(seen) != n {
			t.Fatalf("components cover %d nodes, want %d", len(seen), n)
		}
		firsts := make([]int, len(comps))
		for i, c := range comps {
			firsts[i] = c[0]
		}
		if !sort.IntsAreSorted(firsts) {
			t.Fatalf("components not ordered by first node: %v", firsts)
		}
	}
}

func TestConnectedComponentsEmpty(t *testing.T) {
	if got := ConnectedComponents(nil, func(int) []int { return nil }); got != nil {
		t.Errorf("empty = %v", got)
	}
}

func assertPanics(t *testing.T, name string, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s: expected panic", name)
		}
	}()
	f()
}

func TestPathToLoopingPrev(t *testing.T) {
	// ShortestPath-style maps contain prev[start] = start; asking for a path
	// from a different start must terminate.
	prev := map[int]int{0: 0, 1: 0, 2: 1}
	if got := PathTo(prev, 5, 2); got != nil {
		t.Fatalf("PathTo = %v, want nil", got)
	}
	cyc := map[int]int{1: 2, 2: 1}
	if got := PathTo(cyc, 0, 1); got != nil {
		t.Fatalf("PathTo on cyclic prev = %v, want nil", got)
	}
	if got := PathTo(prev, 0, 2); !slices.Equal(got, []int{0, 1, 2}) {
		t.Fatalf("PathTo = %v, want [0 1 2]", got)
	}
}
