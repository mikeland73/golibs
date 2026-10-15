package async_test

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/mikeland73/golibs/async"
	"github.com/mikeland73/golibs/conc"
)

func ExampleGo() {
	f := async.Go(func() (int, error) { return 6 * 7, nil })
	v, err := f.Await()
	fmt.Println(v, err)
	// Output: 42 <nil>
}

func ExampleGo_panic() {
	f := async.Go(func() (int, error) { panic("boom") })
	_, err := f.Await()
	var pe *conc.PanicError
	fmt.Println(errors.As(err, &pe), pe.Value)
	// Output: true boom
}

func ExampleAll() {
	ctx := context.Background()
	var fs []*async.Future[int]
	for i := range 4 {
		fs = append(fs, async.Go(func() (int, error) { return i * i, nil }))
	}
	vals, err := async.All(ctx, fs...)
	fmt.Println(vals, err)
	// Output: [0 1 4 9] <nil>
}

func ExampleAllSettled() {
	ctx := context.Background()
	results := async.AllSettled(ctx,
		async.Resolved(1),
		async.Rejected[int](errors.New("nope")),
	)
	for _, r := range results {
		fmt.Println(r.Value, r.Err)
	}
	// Output:
	// 1 <nil>
	// 0 nope
}

func ExampleAny() {
	ctx := context.Background()
	v, err := async.Any(ctx,
		async.Rejected[string](errors.New("mirror 1 down")),
		async.Resolved("from mirror 2"),
	)
	fmt.Println(v, err)

	_, err = async.Any(ctx,
		async.Rejected[string](errors.New("mirror 1 down")),
		async.Rejected[string](errors.New("mirror 2 down")),
	)
	fmt.Println(err)
	// Output:
	// from mirror 2 <nil>
	// mirror 1 down
	// mirror 2 down
}

func ExampleThen() {
	n := async.Resolved("123")
	doubled := async.Then(n, func(s string) (int, error) {
		v, err := strconv.Atoi(s)
		return v * 2, err
	})
	fmt.Println(doubled.Await())
	// Output: 246 <nil>
}
