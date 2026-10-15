# golibs

A useful collection of Go libraries: generic data structures, algorithm helpers,
concurrency utilities, and printing/assertion helpers that fill gaps in the
Go standard library.

- **Go 1.26+.**
- **No dependencies.** Every package uses only the standard library and none
  imports another package from this repo, so each one stands alone. You can
  `go get` the module or copy a single package's source file into your project.
- **Does not overlap** with the standard library, `golang.org/x/...`, or
  [`samber/lo`](https://github.com/samber/lo). Where one of those already covers
  something, this README points to it instead (see [Not included](#not-included-and-where-to-find-it)).

```sh
go get github.com/mikeland73/golibs
```

```go
import (
	"github.com/mikeland73/golibs/pq"
	"github.com/mikeland73/golibs/pp"
)

q := pq.NewMin[int]()
q.Push(5, 1, 3)
q.Peek() // 1
q.Pop()  // 1

pp.Debug(q, q.Len()) // [main.go:12] q = PriorityQueue[3 5], q.Len() = 2
```

## Packages

### Data structures

| Package | What it is |
| --- | --- |
| [`pq`](pq) | Generic binary-heap priority queue with `Peek`, `PushPop`, `Replace`, plus `NLargest` / `NSmallest` (Python `heapq`). |
| [`stack`](stack) | LIFO stack. Zero value ready. |
| [`queue`](queue) | FIFO queue on a ring buffer (amortized O(1), shrinks when drained). |
| [`deque`](deque) | Double-ended queue with indexing and `Rotate` (Python `collections.deque`). |
| [`set`](set) | `Set[T]` as a map type with union / intersection / difference / subset ops. |
| [`counter`](counter) | `Counter[T]` multiset as a map type with `MostCommon`, `Total`, `Equal` (Python `collections.Counter`). |
| [`sorted`](sorted) | Balanced ordered `Map`, `Set` and `List` (multiset) with `Floor`/`Ceiling`/`Lower`/`Higher`, `Rank`, `At(i)`, range iteration (Java `TreeMap`, C++ `std::map`/`multiset`, Python `sortedcontainers`). |
| [`lru`](lru) | O(1) LRU cache. |
| [`unionfind`](unionfind) | Disjoint-set union over any comparable type, with path compression and union by size. |
| [`trie`](trie) | Unicode-safe prefix tree with prefix counts/listing and an exposed `Node` for custom DFS. |
| [`fenwick`](fenwick) | Fenwick tree (binary indexed tree) for prefix/range sums with point updates. |
| [`segtree`](segtree) | Generic segment tree with any associative combine function (sum/min/max built in). |
| [`linkedlist`](linkedlist) | LeetCode-style singly linked `Node[T]`: build from values, print (cycle-safe), convert to a slice. |
| [`tree`](tree) | LeetCode-style binary `Node[T]`: parse/format `"[1,2,null,3]"`, traversal iterators, ASCII pretty-print. |

### Algorithms

| Package | What it is |
| --- | --- |
| [`graph`](graph) | BFS (single and multi-source), DFS, shortest path, topological sort, Dijkstra, connected components, and adjacency-list builders. Works on explicit or implicit graphs through a `neighbors` function. |
| [`grid`](grid) | 2D grid helpers: allocate/fill/clone, transpose/rotate, `Dirs4`/`Dirs8`, in-bounds neighbor iterators, pretty-printing. |
| [`bisect`](bisect) | Python `bisect` (`Left`, `Right`, `Insort`) and predicate searches over a range (`FirstTrue`, `LastTrue`, `Float`). |
| [`itertools`](itertools) | Permutations, combinations, product, power set, `NextPermutation`, prefix sums, sliding windows, run-length grouping. |
| [`mathx`](mathx) | Integer math: `Abs`, `Pow`, `ModPow`, `GCD`/`LCM`, Python-style `FloorDiv`/`Mod`, `Isqrt`, primes, divisors, binomials, digits. |
| [`sortx`](sortx) | Sort by key (Python `sorted(key=...)`), comparator builders (`By`, `Desc`, `Chain`), `ArgSort`. |
| [`memo`](memo) | Memoization (Python `@functools.cache`), including memoized recursion for DP. |

### Concurrency

| Package | What it is |
| --- | --- |
| [`async`](async) | Futures/promises: `Go`, `Await`, `All`, `AllSettled`, `Race`, `Any`, `Then`. Panics become errors. |
| [`conc`](conc) | Bounded-parallelism `Map`/`ForEach` that keeps input order and cancels on the first error, plus `Mutex[T]`, `RWMutex[T]` and `SafeMap[K, V]`. |
| [`chans`](chans) | Context-aware channel helpers: `Send`/`Recv`/`TrySend`/`TryRecv`/`RecvTimeout`, pipeline stages (`OrDone`, `Map`, `Filter`, `Take`, `Tee`), and a pub/sub `Broadcaster`. |

### Printing and assertions

| Package | What it is |
| --- | --- |
| [`pp`](pp) | Pretty printing: Python-like `Print`/`Sprint`, Rust-like `Debug` (prints `file:line` and the source of each argument), `Grid`, `JSON`. |
| [`assert`](assert) | Runtime assertions for programs (not `_test.go` files): `Equal`, `True`, `NoError`, `Panics`, `ElementsMatch`, and more. Can panic, log and continue, or exit. |

Package docs include usage snippets, and data-structure docs give the Big-O
cost of each operation. Runnable examples are in each package's `example_test.go`.

## Not included (and where to find it)

| Need | Use |
| --- | --- |
| Map / filter / reduce, uniq, group by, chunk, flatten, zip, partition, keys/values | [`samber/lo`](https://github.com/samber/lo) (`lo.Map`, `lo.Filter`, `lo.Reduce`, `lo.Uniq`, `lo.GroupBy`, `lo.Chunk`, `lo.Zip2`, ...) |
| Sum / min / max / clamp / range | builtin `min`/`max`, `slices.Min`/`slices.Max`, `lo.Sum`, `lo.Clamp`, `lo.Range` |
| Tuples / pairs (including as map keys) | `lo.T2(a, b)` (`lo.Tuple2` is comparable), or an anonymous struct |
| Cartesian product of different slices | `lo.CrossJoin2` ... `lo.CrossJoin9` |
| Pointer to a value | Go 1.26 `new(expr)`, or `lo.ToPtr` |
| Sorting, reversing, binary search on a sorted slice | `slices.Sort`, `slices.SortFunc`, `slices.SortStableFunc`, `slices.Reverse`, `slices.BinarySearch(Func)`, `sort.Search`, `sort.Find` |
| Multi-key comparison | `cmp.Or(cmp.Compare(a.x, b.x), cmp.Compare(a.y, b.y))` |
| Hash map, `defaultdict` | Builtin `map`. Missing keys return the zero value, so `m[k]++` and `m[k] = append(m[k], v)` work without initialization. |
| Doubly linked list, circular list | [`container/list`](https://pkg.go.dev/container/list), [`container/ring`](https://pkg.go.dev/container/ring) |
| Heap interface | [`container/heap`](https://pkg.go.dev/container/heap) (or [`pq`](pq) for a generic version) |
| Bit tricks (popcount, leading/trailing zeros) | [`math/bits`](https://pkg.go.dev/math/bits) |
| Big integers / rationals | [`math/big`](https://pkg.go.dev/math/big) |
| String building, splitting, iterating lines/fields | `strings.Builder`, `strings.Fields`, `strings.SplitSeq`, `strings.Lines` |
| Iterator plumbing | [`iter`](https://pkg.go.dev/iter), `slices.Collect`, `slices.Sorted`, `maps.Keys`, `maps.Values` |
| Wait for goroutines | `sync.WaitGroup.Go` (Go 1.25+) |
| Goroutine group with errors / concurrency limit | [`golang.org/x/sync/errgroup`](https://pkg.go.dev/golang.org/x/sync/errgroup) (`SetLimit`, `TryGo`) |
| Weighted semaphore | [`golang.org/x/sync/semaphore`](https://pkg.go.dev/golang.org/x/sync/semaphore), or a buffered channel |
| Deduplicate concurrent calls | [`golang.org/x/sync/singleflight`](https://pkg.go.dev/golang.org/x/sync/singleflight) |
| Rate limiting | [`golang.org/x/time/rate`](https://pkg.go.dev/golang.org/x/time/rate) |
| Lazy init / run once | `sync.OnceFunc`, `sync.OnceValue`, `sync.OnceValues` |
| Atomic counters | [`sync/atomic`](https://pkg.go.dev/sync/atomic) (`atomic.Int64`, ...) |
| Retry, debounce, throttle | `lo.Attempt*`, `lo.NewDebounce`, `lo.NewThrottle` |
| Fan-in / fan-out, slice to channel and back, batching | `lo.FanIn`, `lo.FanOut`, `lo.SliceToChannel`, `lo.ChannelToSlice`, `lo.Buffer`, `lo.Generator` |
| Fire a function and receive its result on a channel | `lo.Async` (or [`async`](async) for errors, cancellation and combinators) |
| Assertions in `_test.go` files | `testing`, or [`testify`](https://github.com/stretchr/testify) |
| Deterministic tests of concurrent code | [`testing/synctest`](https://pkg.go.dev/testing/synctest) (Go 1.25+) |

## Development

Tooling is managed with [devbox](https://www.jetify.com/devbox), which pins Go
and [golangci-lint](https://golangci-lint.run). golangci-lint also runs the
`gofumpt` and `goimports` formatters.

```sh
devbox run test       # go test -race ./...
devbox run lint       # golangci-lint run ./...
devbox run fmt        # format with gofumpt + goimports
devbox run fmt-check  # fail if anything is unformatted
devbox run check      # fmt-check + lint + test (what CI runs)
```

CI (`.github/workflows/ci.yml`) runs the formatting check, the linter and the
unit tests on every pull request and on every push to `main`.

## License

[Apache 2.0](LICENSE)
