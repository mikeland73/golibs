package tree_test

import (
	"fmt"
	"slices"

	"github.com/mikeland73/golibs/tree"
)

func Example() {
	root := tree.Parse("[4,2,7,1,3,6,9]")
	fmt.Println(slices.Collect(tree.InOrder(root)))
	fmt.Println(tree.Height(root), tree.Size(root))
	fmt.Println(root)
	// Output:
	// [1 2 3 4 6 7 9]
	// 3 7
	//   _4_
	//  /   \
	//  2   7
	// / \ / \
	// 1 3 6 9
}

func ExampleFromLevelOrder() {
	root := tree.FromLevelOrder[int](1, nil, 2, 3)
	fmt.Println(tree.Format(root))
	fmt.Println(slices.Collect(tree.PreOrder(root)))
	// Output:
	// [1,null,2,3]
	// [1 2 3]
}

func ExampleLevelOrder() {
	root := tree.Parse("[3,9,20,null,null,15,7]")
	for level := range tree.LevelOrder(root) {
		fmt.Println(level)
	}
	// Output:
	// [3]
	// [9 20]
	// [15 7]
}

func ExampleFormat() {
	// Invert a binary tree.
	root := tree.Parse("[4,2,7,1,3]")
	var invert func(n *tree.Node[int]) *tree.Node[int]
	invert = func(n *tree.Node[int]) *tree.Node[int] {
		if n != nil {
			n.Left, n.Right = invert(n.Right), invert(n.Left)
		}
		return n
	}
	fmt.Println(tree.Format(invert(root)))
	// Output: [4,7,2,null,null,3,1]
}
