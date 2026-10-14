package chans

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

func TestBroadcasterDeliversToAll(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		b := NewBroadcaster[int](0)
		const subs = 3
		results := make([][]int, subs)
		var wg sync.WaitGroup
		for i := range subs {
			ch, _ := b.Subscribe()
			wg.Go(func() { results[i] = collect(ch) })
		}
		for v := range 5 {
			if err := b.Publish(t.Context(), v); err != nil {
				t.Fatal(err)
			}
		}
		b.Close()
		wg.Wait()
		for i, r := range results {
			if !slices.Equal(r, []int{0, 1, 2, 3, 4}) {
				t.Errorf("sub %d got %v", i, r)
			}
		}
	})
}

func TestBroadcasterZeroValueAndNoSubscribers(t *testing.T) {
	t.Parallel()
	var b Broadcaster[string]
	if err := b.Publish(t.Context(), "nobody"); err != nil {
		t.Fatal(err)
	}
	if n := b.TryPublish("nobody"); n != 0 {
		t.Fatalf("TryPublish = %d", n)
	}
	ch, unsub := b.Subscribe()
	go func() { _ = b.Publish(t.Context(), "hi") }()
	if v := <-ch; v != "hi" {
		t.Fatalf("got %q", v)
	}
	unsub()
}

func TestBroadcasterOnlyFutureValues(t *testing.T) {
	t.Parallel()
	b := NewBroadcaster[int](4)
	_ = b.Publish(t.Context(), 1)
	ch, unsub := b.Subscribe()
	_ = b.Publish(t.Context(), 2)
	unsub()
	if got := collect(ch); !slices.Equal(got, []int{2}) {
		t.Fatalf("got %v; want [2]", got)
	}
}

func TestBroadcasterPublishBlocksOnSlowSubscriber(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		b := NewBroadcaster[int](1)
		ch, unsub := b.Subscribe()
		defer unsub()
		if err := b.Publish(t.Context(), 1); err != nil { // fills buffer
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		if err := b.Publish(ctx, 2); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("err = %v; want DeadlineExceeded", err)
		}
		if err := b.Publish(ctx, 3); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("done ctx: err = %v", err)
		}
		if v := <-ch; v != 1 {
			t.Fatalf("got %d", v)
		}
		if _, ok := TryRecv(ch); ok {
			t.Fatal("value published after deadline was delivered")
		}
	})
}

func TestBroadcasterTryPublishDrops(t *testing.T) {
	t.Parallel()
	b := NewBroadcaster[int](1)
	fast, unsubFast := b.Subscribe()
	slow, unsubSlow := b.Subscribe()
	if n := b.TryPublish(1); n != 2 {
		t.Fatalf("TryPublish = %d; want 2", n)
	}
	<-fast
	if n := b.TryPublish(2); n != 1 { // slow's buffer is full
		t.Fatalf("TryPublish = %d; want 1", n)
	}
	unsubFast()
	unsubSlow()
	if got := collect(fast); !slices.Equal(got, []int{2}) {
		t.Fatalf("fast got %v", got)
	}
	if got := collect(slow); !slices.Equal(got, []int{1}) {
		t.Fatalf("slow got %v", got)
	}
}

func TestBroadcasterUnsubscribeUnblocksPublish(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		b := NewBroadcaster[int](0)
		_, unsub := b.Subscribe() // never reads
		errc := make(chan error, 1)
		go func() { errc <- b.Publish(t.Context(), 1) }()
		synctest.Wait()
		select {
		case <-errc:
			t.Fatal("Publish returned while subscriber was not reading")
		default:
		}
		unsub()
		if err := <-errc; err != nil {
			t.Fatalf("Publish err = %v", err)
		}
		unsub() // idempotent
		unsub()
	})
}

func TestBroadcasterClose(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		b := NewBroadcaster[int](0)
		ch, unsub := b.Subscribe()
		errc := make(chan error, 1)
		go func() { errc <- b.Publish(t.Context(), 1) }() // blocks: nobody reads
		synctest.Wait()
		b.Close()
		if err := <-errc; err != nil {
			t.Fatalf("in-flight Publish err = %v", err)
		}
		if _, ok := <-ch; ok {
			t.Fatal("channel not closed by Close")
		}
		unsub() // after Close: no-op
		b.Close()

		if err := b.Publish(t.Context(), 2); !errors.Is(err, ErrClosed) {
			t.Fatalf("Publish after Close err = %v", err)
		}
		if n := b.TryPublish(2); n != 0 {
			t.Fatalf("TryPublish after Close = %d", n)
		}
		ch2, unsub2 := b.Subscribe()
		if _, ok := <-ch2; ok {
			t.Fatal("Subscribe after Close returned open channel")
		}
		unsub2()
	})
}

func TestBroadcasterConcurrent(t *testing.T) {
	t.Parallel()
	b := NewBroadcaster[int](2)
	var pubs, subs sync.WaitGroup
	for range 4 {
		pubs.Go(func() {
			for j := range 200 {
				if j%2 == 0 {
					_ = b.Publish(t.Context(), j)
				} else {
					b.TryPublish(j)
				}
			}
		})
	}
	for range 4 {
		subs.Go(func() {
			for range 20 {
				ch, unsub := b.Subscribe()
				for range 3 {
					if _, ok := <-ch; !ok {
						return
					}
				}
				unsub()
				for range ch { // drain buffered values; must terminate
				}
			}
		})
	}
	pubs.Wait()
	b.Close() // releases subscribers still waiting for values
	subs.Wait()
}

func TestBroadcasterOrderPerPublisher(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		b := NewBroadcaster[int](0)
		ch, _ := b.Subscribe()
		done := make(chan []int)
		go func() { done <- collect(ch) }()
		for v := range 100 {
			_ = b.Publish(t.Context(), v)
		}
		b.Close()
		got := <-done
		if len(got) != 100 || !slices.IsSorted(got) {
			t.Fatalf("got %v", got)
		}
	})
}

// Close must not wait on a Publish that is blocked on a different, idle
// subscriber.
func TestBroadcasterCloseWithPublishBlockedOnIdleSubscribers(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		var b Broadcaster[int]
		b.Subscribe() // never read
		b.Subscribe() // never read
		published := make(chan error, 1)
		go func() { published <- b.Publish(t.Context(), 1) }()
		synctest.Wait() // Publish is now blocked on one of the subscribers
		b.Close()
		if err := <-published; err != nil {
			t.Fatalf("Publish = %v, want nil", err)
		}
	})
}

// Unsubscribing one subscriber must not wait on a Publish that is blocked on
// another subscriber.
func TestBroadcasterUnsubscribeIndependentOfOtherSubscribers(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		var b Broadcaster[int]
		_, unsubA := b.Subscribe() // never read
		chB, unsubB := b.Subscribe()
		go func() { _ = b.Publish(t.Context(), 1) }()
		go func() {
			for range chB {
			}
		}()
		synctest.Wait() // Publish is now blocked on A
		unsubB()        // must return without A making progress
		unsubA()
	})
}
