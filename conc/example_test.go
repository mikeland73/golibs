package conc_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/mikeland73/golibs/conc"
)

func ExampleMap() {
	words := []string{"go", "is", "fun"}
	upper, err := conc.Map(context.Background(), words, 2, func(_ context.Context, w string) (string, error) {
		return strings.ToUpper(w), nil
	})
	fmt.Println(upper, err)
	// Output: [GO IS FUN] <nil>
}

func ExampleForEach() {
	err := conc.ForEach(context.Background(), []int{1, 2, 3}, 0, func(_ context.Context, n int) error {
		if n == 2 {
			return errors.New("two is not allowed")
		}
		return nil
	})
	fmt.Println(err)
	// Output: two is not allowed
}

func ExampleMutex() {
	var counter conc.Mutex[int]
	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() { counter.With(func(n *int) { *n++ }) })
	}
	wg.Wait()
	fmt.Println(counter.Get())
	// Output: 10
}

func ExampleRWMutex() {
	cfg := conc.NewRWMutex(map[string]string{"env": "dev"})
	cfg.With(func(m *map[string]string) { (*m)["env"] = "prod" })
	cfg.Read(func(m map[string]string) { fmt.Println(m["env"]) })
	// Output: prod
}

func ExampleSafeMap() {
	var hits conc.SafeMap[string, int]
	var wg sync.WaitGroup
	for _, page := range []string{"/", "/about", "/", "/"} {
		wg.Go(func() {
			hits.Update(page, func(n int, _ bool) int { return n + 1 })
		})
	}
	wg.Wait()
	fmt.Println(hits.String())
	v, _ := hits.Get("/")
	fmt.Println(v, hits.Len())
	// Output:
	// SafeMap[/:3 /about:1]
	// 3 2
}
