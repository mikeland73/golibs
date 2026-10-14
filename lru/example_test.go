package lru_test

import (
	"fmt"

	"github.com/mikeland73/golibs/lru"
)

func Example() {
	c := lru.New[string, int](2)
	c.Put("a", 1)
	c.Put("b", 2)
	v, ok := c.Get("a") // "a" becomes most recently used
	fmt.Println(v, ok)
	k, _, evicted := c.Put("c", 3) // evicts least recently used "b"
	fmt.Println(k, evicted)
	_, ok = c.Get("b")
	fmt.Println(ok)
	fmt.Println(c.Keys())
	fmt.Println(c)
	// Output:
	// 1 true
	// b true
	// false
	// [c a]
	// LRU[c:3 a:1]
}
