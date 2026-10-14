// Package mathx provides integer math that the standard library's
// float64-only math package lacks: exact integer powers and roots, Python-style
// floor division and modulo, modular exponentiation, GCD/LCM, primes, divisors,
// binomial coefficients and digit manipulation.
//
//	mathx.Pow(3, 4)              // 81
//	mathx.ModPow(2, 100, 1e9+7)  // 976371285
//	mathx.FloorDiv(-7, 2)        // -4 (Go's -7/2 is -3)
//	mathx.Mod(-7, 3)             // 2  (Go's -7%3 is -1)
//	mathx.Isqrt(1<<62)           // 2147483648, exact
//	mathx.Primes(20)             // [2 3 5 7 11 13 17 19]
//	mathx.Binomial(52, 5)        // 2598960
//
// Integer functions wrap on overflow exactly like Go's built-in arithmetic
// unless documented otherwise.
//
// Not included because they already exist: min/max (builtins), lo.Sum,
// lo.Mean, lo.Clamp, lo.Max/lo.Min, math/bits (bit counting, 128-bit
// multiply), math/big (arbitrary precision).
package mathx

import (
	"math"
	"math/bits"
)

// Signed is the set of signed integer types.
type Signed interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

// Unsigned is the set of unsigned integer types.
type Unsigned interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

// Integer is the set of all integer types.
type Integer interface {
	Signed | Unsigned
}

// Float is the set of floating-point types.
type Float interface {
	~float32 | ~float64
}

// Number is the set of all integer and floating-point types.
type Number interface {
	Integer | Float
}

// Abs returns the absolute value of x. For signed integers, the absolute
// value of the most negative value overflows and is returned unchanged
// (e.g. Abs(math.MinInt64) == math.MinInt64), as in two's complement
// arithmetic. For floats prefer math.Abs if you need -0 handling.
func Abs[T Signed | Float](x T) T {
	if x < 0 {
		return -x
	}
	return x
}

// Sign returns -1 if x < 0, 0 if x == 0 (or NaN), and +1 if x > 0.
func Sign[T Number](x T) int {
	switch {
	case x < 0:
		return -1
	case x > 0:
		return 1
	default:
		return 0
	}
}

// Pow returns base**exp using exponentiation by squaring in O(log exp)
// multiplications. Pow(x, 0) == 1 for every x (including 0). The result
// wraps on overflow. It panics if exp < 0.
func Pow[T Integer](base T, exp int) T {
	if exp < 0 {
		panic("mathx: Pow called with negative exponent")
	}
	result := T(1)
	for exp > 0 {
		if exp&1 == 1 {
			result *= base
		}
		base *= base
		exp >>= 1
	}
	return result
}

// mulMod returns a*b mod m without overflow for any a, b < m.
func mulMod(a, b, m uint64) uint64 {
	hi, lo := bits.Mul64(a, b)
	_, rem := bits.Div64(hi, lo, m) // hi < m because a, b < m
	return rem
}

// toUnsignedMod converts x to an equivalent value in [0, mod) as a uint64.
func toUnsignedMod[T Integer](x, mod T) uint64 {
	return uint64(Mod(x, mod))
}

// ModPow returns base**exp mod mod, in the range [0, mod), using
// exponentiation by squaring in O(log exp) time. Negative bases are reduced
// Python-style first (ModPow(-2, 3, 5) == 2). Intermediate products use
// 128-bit arithmetic (math/bits), so the result is exact for every modulus
// representable by T, including moduli near math.MaxInt64 / math.MaxUint64.
//
// It panics if mod <= 0 or exp < 0.
func ModPow[T Integer](base, exp, mod T) T {
	if mod <= 0 {
		panic("mathx: ModPow called with non-positive modulus")
	}
	if exp < 0 {
		panic("mathx: ModPow called with negative exponent")
	}
	m := uint64(mod)
	b := toUnsignedMod(base, mod)
	e := uint64(exp)
	result := uint64(1) % m
	for e > 0 {
		if e&1 == 1 {
			result = mulMod(result, b, m)
		}
		b = mulMod(b, b, m)
		e >>= 1
	}
	return T(result)
}

// GCD returns the greatest common divisor of a and b, always non-negative
// (like Python's math.gcd). GCD(0, 0) == 0 and GCD(x, 0) == |x|. It runs in
// O(log min(a, b)) time. Like Abs, it overflows (returns a negative value)
// when the result would be -MinInt, e.g. GCD(math.MinInt64, 0).
func GCD[T Integer](a, b T) T {
	for b != 0 {
		a, b = b, a%b
	}
	if a < 0 {
		a = -a
	}
	return a
}

// LCM returns the least common multiple of a and b, always non-negative
// (like Python's math.lcm). LCM(x, 0) == 0. The result wraps on overflow.
func LCM[T Integer](a, b T) T {
	if a == 0 || b == 0 {
		return 0
	}
	l := a / GCD(a, b) * b
	if l < 0 {
		l = -l
	}
	return l
}

// FloorDiv returns a/b rounded toward negative infinity, like Python's a // b.
// Go's a / b truncates toward zero instead: -7/2 == -3 but FloorDiv(-7, 2) == -4.
// It panics if b == 0.
func FloorDiv[T Integer](a, b T) T {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

// CeilDiv returns a/b rounded toward positive infinity: CeilDiv(7, 2) == 4,
// CeilDiv(-7, 2) == -3. It panics if b == 0.
func CeilDiv[T Integer](a, b T) T {
	q := a / b
	if a%b != 0 && (a < 0) == (b < 0) {
		q++
	}
	return q
}

// Mod returns a modulo b with the sign of b, like Python's a % b, so that
// FloorDiv(a, b)*b + Mod(a, b) == a. For b > 0 the result is always in
// [0, b), which makes it the right choice for wrapping indices:
// Mod(-1, n) == n-1 whereas Go's -1 % n == -1. It panics if b == 0.
func Mod[T Integer](a, b T) T {
	r := a % b
	if r != 0 && (r < 0) != (b < 0) {
		r += b
	}
	return r
}

// Isqrt returns the floor of the square root of n, exactly, for every
// integer type including values near math.MaxInt64 and math.MaxUint64 where
// float64 rounding would give the wrong answer. Like Python's math.isqrt.
// It panics if n < 0.
func Isqrt[T Integer](n T) T {
	if n < 0 {
		panic("mathx: Isqrt called with negative number")
	}
	return T(isqrt64(uint64(n)))
}

func isqrt64(n uint64) uint64 {
	r := uint64(math.Sqrt(float64(n)))
	const maxRoot = math.MaxUint32 // floor(sqrt(MaxUint64))
	r = min(r, maxRoot)
	for r*r > n {
		r--
	}
	for r < maxRoot && (r+1)*(r+1) <= n {
		r++
	}
	return r
}

// IsPrime reports whether n is a prime number. It is exact for all 64-bit
// values, using trial division by small primes followed by a deterministic
// Miller-Rabin test, so it runs in O(log^3 n) time rather than O(sqrt n).
// Numbers less than 2 are not prime.
func IsPrime[T Integer](n T) bool {
	if n < 2 {
		return false
	}
	return isPrime64(uint64(n))
}

var smallPrimes = [...]uint64{2, 3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37}

func isPrime64(n uint64) bool {
	for _, p := range smallPrimes {
		if n%p == 0 {
			return n == p
		}
	}
	if n < 37*37 {
		return true
	}
	// n-1 = d * 2^s with d odd.
	d := n - 1
	s := bits.TrailingZeros64(d)
	d >>= s
	// These bases are sufficient for every n < 2^64.
	for _, a := range smallPrimes {
		x := modPow64(a, d, n)
		if x == 1 || x == n-1 {
			continue
		}
		composite := true
		for range s - 1 {
			x = mulMod(x, x, n)
			if x == n-1 {
				composite = false
				break
			}
		}
		if composite {
			return false
		}
	}
	return true
}

func modPow64(b, e, m uint64) uint64 {
	result := uint64(1)
	b %= m
	for e > 0 {
		if e&1 == 1 {
			result = mulMod(result, b, m)
		}
		b = mulMod(b, b, m)
		e >>= 1
	}
	return result
}

// Primes returns all primes <= n in increasing order using the sieve of
// Eratosthenes in O(n log log n) time and O(n) memory. It returns an empty
// slice if n < 2.
func Primes(n int) []int {
	if n < 2 {
		return []int{}
	}
	composite := make([]bool, n+1)
	var primes []int
	for i := 2; i <= n; i++ {
		if composite[i] {
			continue
		}
		primes = append(primes, i)
		for j := i * i; j <= n; j += i {
			composite[j] = true
		}
	}
	return primes
}

// Divisors returns all positive divisors of n in increasing order, in
// O(sqrt n) time. Divisors(12) == [1 2 3 4 6 12]. A negative n yields the
// divisors of |n|. It panics if n == 0 or n == math.MinInt.
func Divisors(n int) []int {
	if n == 0 {
		panic("mathx: Divisors called with 0")
	}
	if n == math.MinInt {
		panic("mathx: Divisors called with math.MinInt")
	}
	if n < 0 {
		n = -n
	}
	var small, large []int
	for i := 1; i <= n/i; i++ {
		if n%i == 0 {
			small = append(small, i)
			if i != n/i {
				large = append(large, n/i)
			}
		}
	}
	for i := len(large) - 1; i >= 0; i-- {
		small = append(small, large[i])
	}
	return small
}

// Binomial returns the binomial coefficient C(n, k) — the number of ways to
// choose k items from n — computed exactly with integer arithmetic in O(k)
// time (like Python's math.comb). It returns 0 if k < 0 or k > n.
//
// Intermediate values never exceed the final result, and every
// product is checked: it panics if the result does not fit in an int (for
// example C(68, 34)). For larger values use math/big.Int.Binomial.
func Binomial(n, k int) int {
	if k < 0 || k > n {
		return 0
	}
	k = min(k, n-k)
	result := 1
	for i := 1; i <= k; i++ {
		// result*(n-k+i)/i is exact. Divide out common factors first to keep
		// intermediates small: result/g is coprime with i/g, so i/g | (n-k+i).
		g := GCD(result, i)
		hi, lo := bits.Mul64(uint64(result/g), uint64((n-k+i)/(i/g)))
		if hi != 0 || lo > math.MaxInt {
			panic("mathx: Binomial result overflows int")
		}
		result = int(lo)
	}
	return result
}

// Digits returns the base-10 digits of n, most significant first.
// Digits(1203) == [1 2 0 3], Digits(0) == [0], and a negative n yields the
// digits of |n| (Digits(-42) == [4 2]).
func Digits(n int) []int {
	if n == 0 {
		return []int{0}
	}
	var d []int
	for n != 0 {
		r := n % 10
		if r < 0 {
			r = -r // avoids overflowing on math.MinInt
		}
		d = append(d, r)
		n /= 10
	}
	for i, j := 0, len(d)-1; i < j; i, j = i+1, j-1 {
		d[i], d[j] = d[j], d[i]
	}
	return d
}

// FromDigits is the inverse of Digits: it builds an int from base-10 digits,
// most significant first. FromDigits([]int{1, 2, 3}) == 123 and
// FromDigits(nil) == 0. Digits are not validated, so values outside 0..9 are
// simply weighted by their position. The result wraps on overflow.
func FromDigits(d []int) int {
	n := 0
	for _, x := range d {
		n = n*10 + x
	}
	return n
}
