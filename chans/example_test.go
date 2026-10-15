package chans_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mikeland73/golibs/chans"
)

// naturals yields 1, 2, 3, ... until ctx is cancelled.
func naturals(ctx context.Context) <-chan int {
	ch := make(chan int)
	go func() {
		defer close(ch)
		for i := 1; chans.Send(ctx, ch, i) == nil; i++ {
		}
	}()
	return ch
}

func Example_pipeline() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // stops every stage and the producer

	odds := chans.Filter(ctx, naturals(ctx), func(n int) bool { return n%2 == 1 })
	squares := chans.Map(ctx, odds, 1, func(n int) int { return n * n })
	for v := range chans.Take(ctx, squares, 5) {
		fmt.Print(v, " ")
	}
	fmt.Println()
	// Output: 1 9 25 49 81
}

func ExampleSend() {
	ch := make(chan int) // nobody is receiving
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	err := chans.Send(ctx, ch, 1)
	fmt.Println(errors.Is(err, context.DeadlineExceeded))
	// Output: true
}

func ExampleRecv() {
	ch := make(chan string, 1)
	ch <- "hello"
	close(ch)
	ctx := context.Background()
	v, err := chans.Recv(ctx, ch)
	fmt.Printf("%q %v\n", v, err)
	v, err = chans.Recv(ctx, ch)
	fmt.Printf("%q %v\n", v, err)
	// Output:
	// "hello" <nil>
	// "" chans: channel closed
}

func ExampleTryRecv() {
	ch := make(chan int, 1)
	fmt.Println(chans.TrySend(ch, 1), chans.TrySend(ch, 2))
	fmt.Println(chans.TryRecv(ch))
	fmt.Println(chans.TryRecv(ch))
	// Output:
	// true false
	// 1 true
	// 0 false
}

func ExampleTee() {
	ctx := context.Background()
	in := make(chan string, 2)
	in <- "a"
	in <- "b"
	close(in)
	outs := chans.Tee(ctx, in, 2)
	for range 2 {
		fmt.Println(<-outs[0], <-outs[1])
	}
	// Output:
	// a a
	// b b
}

func ExampleBroadcaster() {
	ctx := context.Background()
	b := chans.NewBroadcaster[string](1)
	alice, _ := b.Subscribe()
	bob, unsubscribeBob := b.Subscribe()

	_ = b.Publish(ctx, "hello")
	fmt.Println(<-alice, <-bob)

	unsubscribeBob()
	_ = b.Publish(ctx, "bye")
	fmt.Println(<-alice)

	b.Close()
	_, ok := <-alice
	fmt.Println(ok)
	// Output:
	// hello hello
	// bye
	// false
}
