package mathx

import (
	"math"
	"math/big"
	"slices"
	"testing"
)

func expectPanic(t *testing.T, name string, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s: expected panic", name)
		}
	}()
	f()
}

func TestAbsSign(t *testing.T) {
	if Abs(-5) != 5 || Abs(5) != 5 || Abs(0) != 0 {
		t.Error("Abs int")
	}
	if Abs(-2.5) != 2.5 || Abs(int8(-128)) != -128 {
		t.Error("Abs float / overflow")
	}
	if Sign(-3) != -1 || Sign(0) != 0 || Sign(7) != 1 {
		t.Error("Sign int")
	}
	if Sign(uint(3)) != 1 || Sign(-0.1) != -1 || Sign(math.NaN()) != 0 {
		t.Error("Sign misc")
	}
}

func TestPow(t *testing.T) {
	tests := []struct {
		base int64
		exp  int
		want int64
	}{
		{2, 0, 1},
		{0, 0, 1},
		{0, 5, 0},
		{2, 10, 1024},
		{-2, 3, -8},
		{-2, 4, 16},
		{3, 39, 4052555153018976267},
		{1, 1000000, 1},
		{-1, 1000001, -1},
	}
	for _, tt := range tests {
		if got := Pow(tt.base, tt.exp); got != tt.want {
			t.Errorf("Pow(%d, %d) = %d want %d", tt.base, tt.exp, got, tt.want)
		}
	}
	if Pow(uint8(2), 8) != 0 { // wraps
		t.Error("Pow should wrap")
	}
	expectPanic(t, "Pow negative", func() { Pow(2, -1) })
}

func TestModPow(t *testing.T) {
	check := func(base, exp, mod int64) {
		t.Helper()
		want := new(big.Int).Exp(big.NewInt(base), big.NewInt(exp), big.NewInt(mod))
		if want.Sign() < 0 {
			want.Add(want, big.NewInt(mod))
		}
		if got := ModPow(base, exp, mod); got != want.Int64() {
			t.Errorf("ModPow(%d, %d, %d) = %d want %s", base, exp, mod, got, want)
		}
	}
	check(2, 100, 1_000_000_007)
	check(-2, 3, 5)
	check(-7, 5, 13)
	check(10, 0, 7)
	check(0, 0, 7)
	check(123456789, 987654321, 1)
	check(3_000_000_000, 1_000_000_000_000, 3_000_000_019)
	check(math.MaxInt64-1, math.MaxInt64, math.MaxInt64)
	check(-math.MaxInt64, 12345, math.MaxInt64-24)
	if got := ModPow(2, 100, 1_000_000_007); got != 976371285 {
		t.Errorf("got %d", got)
	}
	// uint64 near max.
	m := uint64(math.MaxUint64 - 58) // largest 64-bit prime: 2^64-59
	if got := ModPow(uint64(2), m-1, m); got != 1 {
		t.Errorf("Fermat uint64: %d", got)
	}
	if got := ModPow(int32(-3), int32(3), int32(7)); got != 1 { // -27 mod 7 = 1
		t.Errorf("int32: %d", got)
	}
	expectPanic(t, "mod 0", func() { ModPow(2, 3, 0) })
	expectPanic(t, "mod negative", func() { ModPow(2, 3, -5) })
	expectPanic(t, "exp negative", func() { ModPow(2, -3, 5) })
}

func TestGCDLCM(t *testing.T) {
	tests := []struct{ a, b, gcd, lcm int }{
		{12, 18, 6, 36},
		{0, 0, 0, 0},
		{0, 5, 5, 0},
		{5, 0, 5, 0},
		{-4, 6, 2, 12},
		{4, -6, 2, 12},
		{-4, -6, 2, 12},
		{7, 13, 1, 91},
		{1, 1, 1, 1},
	}
	for _, tt := range tests {
		if got := GCD(tt.a, tt.b); got != tt.gcd {
			t.Errorf("GCD(%d, %d) = %d want %d", tt.a, tt.b, got, tt.gcd)
		}
		if got := LCM(tt.a, tt.b); got != tt.lcm {
			t.Errorf("LCM(%d, %d) = %d want %d", tt.a, tt.b, got, tt.lcm)
		}
	}
	if GCD(uint(48), uint(180)) != 12 || LCM(uint8(4), uint8(6)) != 12 {
		t.Error("unsigned")
	}
}

func TestDivMod(t *testing.T) {
	for a := -20; a <= 20; a++ {
		for b := -7; b <= 7; b++ {
			if b == 0 {
				continue
			}
			fa, fb := float64(a), float64(b)
			if got, want := FloorDiv(a, b), int(math.Floor(fa/fb)); got != want {
				t.Errorf("FloorDiv(%d, %d) = %d want %d", a, b, got, want)
			}
			if got, want := CeilDiv(a, b), int(math.Ceil(fa/fb)); got != want {
				t.Errorf("CeilDiv(%d, %d) = %d want %d", a, b, got, want)
			}
			m := Mod(a, b)
			if FloorDiv(a, b)*b+m != a {
				t.Errorf("identity broken for %d, %d", a, b)
			}
			if m != 0 && (m < 0) != (b < 0) {
				t.Errorf("Mod(%d, %d) = %d has wrong sign", a, b, m)
			}
		}
	}
	if Mod(-1, 5) != 4 || Mod(7, -3) != -2 || FloorDiv(-7, 2) != -4 || CeilDiv(7, 2) != 4 {
		t.Error("spot checks")
	}
	if FloorDiv(uint(7), uint(2)) != 3 || CeilDiv(uint(7), uint(2)) != 4 || Mod(uint(7), uint(3)) != 1 {
		t.Error("unsigned")
	}
	expectPanic(t, "FloorDiv by zero", func() { FloorDiv(1, 0) })
}

func TestIsqrt(t *testing.T) {
	for n := range 10000 {
		r := Isqrt(n)
		if r*r > n || (r+1)*(r+1) <= n {
			t.Fatalf("Isqrt(%d) = %d", n, r)
		}
	}
	cases := []int64{
		math.MaxInt64, 1 << 62, (1 << 62) - 1, 3037000499 * 3037000499, 3037000499*3037000499 - 1,
		999999999999999999, 4503599761588224, 4503599761588223,
	}
	for _, n := range cases {
		r := Isqrt(n)
		want := new(big.Int).Sqrt(new(big.Int).SetInt64(n)).Int64()
		if r != want {
			t.Errorf("Isqrt(%d) = %d want %d", n, r, want)
		}
	}
	if Isqrt(uint64(math.MaxUint64)) != math.MaxUint32 {
		t.Error("MaxUint64")
	}
	if Isqrt(uint64(math.MaxUint32)*math.MaxUint32) != math.MaxUint32 {
		t.Error("square of MaxUint32")
	}
	if Isqrt(int8(127)) != 11 {
		t.Error("int8")
	}
	expectPanic(t, "negative", func() { Isqrt(-1) })
}

func TestIsPrime(t *testing.T) {
	sieve := Primes(100000)
	set := map[int]bool{}
	for _, p := range sieve {
		set[p] = true
	}
	for n := -5; n <= 100000; n++ {
		if IsPrime(n) != set[n] {
			t.Fatalf("IsPrime(%d) = %v", n, IsPrime(n))
		}
	}
	tests := []struct {
		n    uint64
		want bool
	}{
		{1_000_000_007, true},
		{998244353, true},
		{math.MaxUint64 - 58, true},        // 2^64 - 59
		{(1 << 61) - 1, true},              // Mersenne prime
		{3215031751, false},                // strong pseudoprime to bases 2,3,5,7
		{3825123056546413051, false},       // strong pseudoprime to bases 2..23
		{1_000_000_007 * 998244353, false}, // semiprime
		{4294967291 * 4294967279, false},   // product of two large 32-bit primes
		{math.MaxUint64, false},            // 3 * 5 * 17 * ...
		{(1 << 31) - 1, true},              // 2^31-1
		{561, false},                       // Carmichael
		{41041, false},                     // Carmichael
		{1369, false},                      // 37^2
		{1373, true},                       // just above 37^2
		{1_000_000_000_000_000_003, true},  // 10^18+3
		{1_000_000_000_000_000_001, false},
	}
	for _, tt := range tests {
		if got := IsPrime(tt.n); got != tt.want {
			t.Errorf("IsPrime(%d) = %v want %v", tt.n, got, tt.want)
		}
		if bp := new(big.Int).SetUint64(tt.n).ProbablyPrime(20); bp != tt.want {
			t.Errorf("test table wrong for %d (big says %v)", tt.n, bp)
		}
	}
}

func TestPrimes(t *testing.T) {
	if got := Primes(20); !slices.Equal(got, []int{2, 3, 5, 7, 11, 13, 17, 19}) {
		t.Errorf("Primes(20) = %v", got)
	}
	for _, n := range []int{-1, 0, 1} {
		if got := Primes(n); got == nil || len(got) != 0 {
			t.Errorf("Primes(%d) = %v", n, got)
		}
	}
	if got := Primes(2); !slices.Equal(got, []int{2}) {
		t.Errorf("Primes(2) = %v", got)
	}
	if got := len(Primes(1_000_000)); got != 78498 {
		t.Errorf("pi(1e6) = %d", got)
	}
}

func TestDivisors(t *testing.T) {
	tests := []struct {
		n    int
		want []int
	}{
		{1, []int{1}},
		{12, []int{1, 2, 3, 4, 6, 12}},
		{16, []int{1, 2, 4, 8, 16}},
		{13, []int{1, 13}},
		{-6, []int{1, 2, 3, 6}},
		{36, []int{1, 2, 3, 4, 6, 9, 12, 18, 36}},
	}
	for _, tt := range tests {
		if got := Divisors(tt.n); !slices.Equal(got, tt.want) {
			t.Errorf("Divisors(%d) = %v want %v", tt.n, got, tt.want)
		}
	}
	for n := 1; n < 500; n++ {
		var want []int
		for d := 1; d <= n; d++ {
			if n%d == 0 {
				want = append(want, d)
			}
		}
		if got := Divisors(n); !slices.Equal(got, want) {
			t.Fatalf("Divisors(%d) = %v", n, got)
		}
	}
	expectPanic(t, "zero", func() { Divisors(0) })
}

func TestBinomial(t *testing.T) {
	for n := range 62 {
		for k := -1; k <= n+1; k++ {
			want := new(big.Int).Binomial(int64(n), int64(k)).Int64()
			if k < 0 || k > n {
				want = 0
			}
			if got := Binomial(n, k); int64(got) != want {
				t.Fatalf("Binomial(%d, %d) = %d want %d", n, k, got, want)
			}
		}
	}
	if Binomial(52, 5) != 2598960 || Binomial(0, 0) != 1 || Binomial(-3, 1) != 0 {
		t.Error("spot checks")
	}
	// Near the int64 limit: C(66, 33) = 7219428434016265740 fits.
	if Binomial(66, 33) != 7219428434016265740 {
		t.Errorf("C(66,33) = %d", Binomial(66, 33))
	}
	// Large n, small k exercises intermediate-overflow avoidance.
	if got, want := Binomial(1_000_000_000, 2), 499999999500000000; got != want {
		t.Errorf("C(1e9, 2) = %d", got)
	}
	expectPanic(t, "overflow", func() { Binomial(68, 34) })
}

func TestDigits(t *testing.T) {
	tests := []struct {
		n    int
		want []int
	}{
		{0, []int{0}},
		{7, []int{7}},
		{1203, []int{1, 2, 0, 3}},
		{-42, []int{4, 2}},
		{math.MaxInt64, []int{9, 2, 2, 3, 3, 7, 2, 0, 3, 6, 8, 5, 4, 7, 7, 5, 8, 0, 7}},
		{math.MinInt64, []int{9, 2, 2, 3, 3, 7, 2, 0, 3, 6, 8, 5, 4, 7, 7, 5, 8, 0, 8}},
	}
	for _, tt := range tests {
		if got := Digits(tt.n); !slices.Equal(got, tt.want) {
			t.Errorf("Digits(%d) = %v want %v", tt.n, got, tt.want)
		}
	}
	for _, n := range []int{0, 1, 10, 99, 12345, math.MaxInt64} {
		if FromDigits(Digits(n)) != n {
			t.Errorf("round trip %d", n)
		}
	}
	if FromDigits(nil) != 0 || FromDigits([]int{0, 0, 7}) != 7 {
		t.Error("FromDigits edge")
	}
}

func TestDivisorsMinIntPanics(t *testing.T) {
	expectPanic(t, "Divisors(math.MinInt)", func() { Divisors(math.MinInt) })
}
