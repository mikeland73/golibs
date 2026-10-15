package mathx_test

import (
	"fmt"

	"github.com/mikeland73/golibs/mathx"
)

func Example() {
	fmt.Println(mathx.Pow(3, 4))
	fmt.Println(mathx.ModPow(2, 100, 1_000_000_007))
	fmt.Println(-7/2, mathx.FloorDiv(-7, 2), mathx.CeilDiv(-7, 2))
	fmt.Println(-7%3, mathx.Mod(-7, 3))
	fmt.Println(mathx.GCD(12, 18), mathx.LCM(4, 6))
	fmt.Println(mathx.Isqrt(99), mathx.IsPrime(97))
	// Output:
	// 81
	// 976371285
	// -3 -4 -3
	// -1 2
	// 6 12
	// 9 true
}

func ExampleMod() {
	// Wrap an index around a circular buffer.
	n := 5
	for _, i := range []int{-1, 0, 6} {
		fmt.Print(mathx.Mod(i, n), " ")
	}
	fmt.Println()
	// Output:
	// 4 0 1
}

func ExamplePrimes() {
	fmt.Println(mathx.Primes(30))
	// Output:
	// [2 3 5 7 11 13 17 19 23 29]
}

func ExampleDivisors() {
	fmt.Println(mathx.Divisors(28))
	// Output:
	// [1 2 4 7 14 28]
}

func ExampleBinomial() {
	fmt.Println(mathx.Binomial(5, 2), mathx.Binomial(52, 5))
	// Output:
	// 10 2598960
}

func ExampleDigits() {
	d := mathx.Digits(1203)
	fmt.Println(d)
	sum := 0
	for _, x := range d {
		sum += x
	}
	fmt.Println("digit sum:", sum)
	fmt.Println(mathx.FromDigits([]int{4, 2}))
	// Output:
	// [1 2 0 3]
	// digit sum: 6
	// 42
}
