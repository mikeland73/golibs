package stack_test

import (
	"fmt"

	"github.com/mikeland73/golibs/stack"
)

func Example() {
	var s stack.Stack[int] // zero value is ready to use
	s.Push(1, 2, 3)
	fmt.Println(s.String())
	fmt.Println(s.Peek())
	fmt.Println(s.Pop(), s.Pop())
	fmt.Println(s.Len())
	// Output:
	// Stack[1 2 3]
	// 3
	// 3 2
	// 1
}

func Example_balancedParens() {
	balanced := func(str string) bool {
		pairs := map[rune]rune{')': '(', ']': '[', '}': '{'}
		var s stack.Stack[rune]
		for _, r := range str {
			switch r {
			case '(', '[', '{':
				s.Push(r)
			case ')', ']', '}':
				if top, ok := s.TryPop(); !ok || top != pairs[r] {
					return false
				}
			}
		}
		return s.IsEmpty()
	}
	fmt.Println(balanced("{[()()]}"), balanced("([)]"))
	// Output: true false
}
