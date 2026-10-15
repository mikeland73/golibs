package counter_test

import (
	"fmt"
	"strings"

	"github.com/mikeland73/golibs/counter"
)

func Example() {
	words := strings.Fields("the cat and the hat and the bat")
	c := counter.New(words...)
	fmt.Println(c["the"], c["dog"])
	fmt.Println(c.MostCommon(2))
	c["dog"]++
	fmt.Println(c.Total())
	fmt.Println(c)
	// Output:
	// 3 0
	// [{the 3} {and 2}]
	// 9
	// Counter{the: 3, and: 2, bat: 1, cat: 1, dog: 1, hat: 1}
}

func ExampleCounter_Equal() {
	isAnagram := func(a, b string) bool {
		return counter.New([]rune(a)...).Equal(counter.New([]rune(b)...))
	}
	fmt.Println(isAnagram("listen", "silent"), isAnagram("rat", "car"))
	// Output: true false
}
