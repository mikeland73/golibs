package trie_test

import (
	"fmt"
	"slices"

	"github.com/mikeland73/golibs/trie"
)

func Example() {
	t := trie.New("app", "apple", "apply", "bat")
	t.Insert("app") // words are counted

	fmt.Println(t.Contains("app"), t.Count("app"), t.Contains("ap"))
	fmt.Println(t.HasPrefix("ap"), t.CountPrefix("app"))
	fmt.Println(t.WithPrefix("appl"))
	t.Delete("apple")
	fmt.Println(t, t.Len())
	// Output:
	// true 2 false
	// true 4
	// [apple apply]
	// Trie[app apply bat] 4
}

// Word Search II: find all dictionary words in a letter grid by walking the
// trie nodes during a DFS.
func ExampleTrie_Root() {
	board := []string{
		"oaan",
		"etae",
		"ihkr",
		"iflv",
	}
	dict := trie.New("oath", "pea", "eat", "rain")

	found := map[string]bool{}
	seen := make([][]bool, len(board))
	for i := range seen {
		seen[i] = make([]bool, len(board[0]))
	}
	var dfs func(r, c int, n *trie.Node, path []byte)
	dfs = func(r, c int, n *trie.Node, path []byte) {
		if r < 0 || c < 0 || r >= len(board) || c >= len(board[0]) || seen[r][c] {
			return
		}
		ch := board[r][c]
		next := n.Child(rune(ch))
		if next == nil {
			return // no word continues this way: prune
		}
		path = append(path, ch)
		if next.IsWord() {
			found[string(path)] = true
		}
		seen[r][c] = true
		dfs(r+1, c, next, path)
		dfs(r-1, c, next, path)
		dfs(r, c+1, next, path)
		dfs(r, c-1, next, path)
		seen[r][c] = false
	}
	for r := range board {
		for c := range board[r] {
			dfs(r, c, dict.Root(), nil)
		}
	}
	var words []string
	for w := range found {
		words = append(words, w)
	}
	slices.Sort(words)
	fmt.Println(words)
	// Output:
	// [eat oath]
}
