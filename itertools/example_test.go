package itertools_test

import (
	"fmt"
	"slices"

	"github.com/mikeland73/golibs/itertools"
)

func ExamplePermutations() {
	for p := range itertools.Permutations([]string{"a", "b", "c"}, 2) {
		fmt.Print(p, " ")
	}
	fmt.Println()
	// Output:
	// [a b] [a c] [b a] [b c] [c a] [c b]
}

func ExampleCombinations() {
	for c := range itertools.Combinations([]int{1, 2, 3, 4}, 2) {
		fmt.Print(c, " ")
	}
	fmt.Println()
	// Output:
	// [1 2] [1 3] [1 4] [2 3] [2 4] [3 4]
}

func ExampleCombinationsWithReplacement() {
	for c := range itertools.CombinationsWithReplacement([]string{"x", "y"}, 2) {
		fmt.Print(c, " ")
	}
	fmt.Println()
	// Output:
	// [x x] [x y] [y y]
}

func ExampleProduct() {
	for p := range itertools.Product([]int{0, 1}, 2) {
		fmt.Print(p, " ")
	}
	fmt.Println()
	// Output:
	// [0 0] [0 1] [1 0] [1 1]
}

func ExamplePowerSet() {
	// Find all subsets that sum to 5.
	for sub := range itertools.PowerSet([]int{1, 2, 3, 4}) {
		sum := 0
		for _, x := range sub {
			sum += x
		}
		if sum == 5 {
			fmt.Println(sub)
		}
	}
	// Output:
	// [1 4]
	// [2 3]
}

func ExampleNextPermutation() {
	s := []byte("aab")
	slices.Sort(s)
	for ok := true; ok; ok = itertools.NextPermutation(s) {
		fmt.Println(string(s))
	}
	// Output:
	// aab
	// aba
	// baa
}

func ExampleAccumulate() {
	fmt.Println(itertools.Accumulate([]int{3, 1, 4, 1, 5}, func(a, b int) int { return max(a, b) }))
	// Output:
	// [3 3 4 4 5]
}

func ExamplePrefixSums() {
	s := []int{3, 1, 4, 1, 5}
	p := itertools.PrefixSums(s)
	fmt.Println(p)
	fmt.Println("sum(s[1:4]) =", p[4]-p[1])
	// Output:
	// [0 3 4 8 9 14]
	// sum(s[1:4]) = 6
}

func ExamplePairwise() {
	for a, b := range itertools.Pairwise([]int{1, 4, 9, 16}) {
		fmt.Print(b-a, " ")
	}
	fmt.Println()
	// Output:
	// 3 5 7
}

func ExampleWindows() {
	for w := range itertools.Windows([]int{1, 2, 3, 4, 5}, 3) {
		fmt.Print(w, " ")
	}
	fmt.Println()
	// Output:
	// [1 2 3] [2 3 4] [3 4 5]
}

func ExampleRunLength() {
	for ch, n := range itertools.RunLength([]rune("aaabccdddd")) {
		fmt.Printf("%c%d", ch, n)
	}
	fmt.Println()
	// Output:
	// a3b1c2d4
}
