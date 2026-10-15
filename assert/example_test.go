package assert_test

import (
	"fmt"
	"os"
	"slices"

	"github.com/mikeland73/golibs/assert"
)

func reverse(s []int) []int {
	out := slices.Clone(s)
	slices.Reverse(out)
	return out
}

func Example() {
	// In Log mode failures are printed and execution continues.
	defer assert.SetMode(assert.SetMode(assert.Log))
	assert.Output = os.Stdout
	defer func() { assert.Output = os.Stderr }()
	before := assert.Failures()

	assert.Equal(reverse([]int{1, 2, 3}), []int{3, 2, 1})
	assert.Equal(reverse(nil), []int{}) // nil and empty slices are equal
	assert.ElementsMatch([]string{"b", "a"}, []string{"a", "b"})
	assert.Equal(reverse([]int{1, 2}), []int{1, 2}, "reverse of %v", []int{1, 2})

	fmt.Println("failures:", assert.Failures()-before)
	// Output:
	// assert.Equal failed at example_test.go:27: reverse of [1 2]
	//   got:  [2, 1]
	//   want: [1, 2]
	// failures: 1
}

func ExamplePanics() {
	ok := assert.Panics(func() {
		var s []int
		_ = s[len(s)]
	})
	fmt.Println(ok)
	// Output:
	// true
}

func ExampleSetMode() {
	// The default mode panics, like Python's assert statement.
	defer func() { fmt.Println("recovered:", recover()) }()
	assert.True(1+1 == 3, "math is broken")
	// Output:
	// recovered: assert.True failed at example_test.go:50: math is broken
}
