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

// source returns a closed-at-end channel yielding vals.
func source[T any](vals ...T) <-chan T {
	ch := make(chan T)
	go func() {
		defer close(ch)
		for _, v := range vals {
			ch <- v
		}
	}()
	return ch
}

// buffered returns a closed channel holding vals, with no goroutine behind it.
func buffered[T any](vals ...T) <-chan T {
	ch := make(chan T, len(vals))
	for _, v := range vals {
		ch <- v
	}
	close(ch)
	return ch
}

// endless returns a channel yielding 0, 1, 2, ... until ctx is done.
func endless(ctx context.Context) <-chan int {
	ch := make(chan int)
	go func() {
		defer close(ch)
		for i := 0; ; i++ {
			if Send(ctx, ch, i) != nil {
				return
			}
		}
	}()
	return ch
}

func collect[T any](ch <-chan T) []T {
	var out []T
	for v := range ch {
		out = append(out, v)
	}
	return out
}

func TestSend(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		ch := make(chan int, 1)
		if err := Send(t.Context(), ch, 1); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		// Buffer full: blocks until timeout.
		if err := Send(ctx, ch, 2); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("err = %v", err)
		}
		<-ch
		// Ctx already done: never sends even though there is room.
		if err := Send(ctx, ch, 3); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("err = %v", err)
		}
		if len(ch) != 0 {
			t.Fatal("Send with done ctx sent a value")
		}
	})
}

func TestRecv(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		ch := make(chan int, 1)
		ch <- 7
		if v, err := Recv(t.Context(), ch); v != 7 || err != nil {
			t.Fatalf("Recv = %v, %v", v, err)
		}
		ctx, cancel := context.WithCancel(t.Context())
		go func() {
			time.Sleep(time.Second)
			cancel()
		}()
		if _, err := Recv(ctx, ch); !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v", err)
		}
		ch <- 8
		if _, err := Recv(ctx, ch); !errors.Is(err, context.Canceled) {
			t.Fatalf("Recv with done ctx: err = %v", err)
		}
		if len(ch) != 1 {
			t.Fatal("Recv with done ctx consumed a value")
		}
		<-ch
		close(ch)
		if v, err := Recv(t.Context(), ch); v != 0 || !errors.Is(err, ErrClosed) {
			t.Fatalf("Recv closed = %v, %v", v, err)
		}
	})
}

func TestTrySendTryRecv(t *testing.T) {
	t.Parallel()
	ch := make(chan string, 1)
	if v, ok := TryRecv(ch); ok || v != "" {
		t.Fatalf("TryRecv empty = %q, %v", v, ok)
	}
	if !TrySend(ch, "a") {
		t.Fatal("TrySend to empty buffer failed")
	}
	if TrySend(ch, "b") {
		t.Fatal("TrySend to full buffer succeeded")
	}
	if v, ok := TryRecv(ch); !ok || v != "a" {
		t.Fatalf("TryRecv = %q, %v", v, ok)
	}
	close(ch)
	if _, ok := TryRecv(ch); ok {
		t.Fatal("TryRecv closed ok = true")
	}
	if TrySend[int](nil, 1) {
		t.Fatal("TrySend to nil channel succeeded")
	}
}

func TestRecvTimeout(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		ch := make(chan int)
		start := time.Now()
		if _, ok := RecvTimeout(ch, 5*time.Second); ok {
			t.Fatal("ok = true on timeout")
		}
		if d := time.Since(start); d != 5*time.Second {
			t.Fatalf("waited %v", d)
		}
		go func() {
			time.Sleep(time.Second)
			ch <- 3
		}()
		if v, ok := RecvTimeout(ch, 5*time.Second); !ok || v != 3 {
			t.Fatalf("RecvTimeout = %v, %v", v, ok)
		}
		// Ready value with zero timeout.
		buf := make(chan int, 1)
		buf <- 4
		if v, ok := RecvTimeout(buf, 0); !ok || v != 4 {
			t.Fatalf("RecvTimeout(0) = %v, %v", v, ok)
		}
		close(buf)
		if _, ok := RecvTimeout(buf, time.Hour); ok {
			t.Fatal("closed ok = true")
		}
	})
}

func TestOrDone(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		got := collect(OrDone(t.Context(), source(1, 2, 3)))
		if !slices.Equal(got, []int{1, 2, 3}) {
			t.Fatalf("got %v", got)
		}

		// A never-closed input: cancelling ctx closes the output.
		ctx, cancel := context.WithCancel(t.Context())
		never := make(chan int)
		out := OrDone(ctx, never)
		cancel()
		if _, ok := <-out; ok {
			t.Fatal("received value after cancel")
		}
	})
}

func TestMap(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		in := []int{1, 2, 3, 4, 5, 6, 7, 8}
		for _, workers := range []int{-1, 0, 1} {
			got := collect(Map(t.Context(), source(in...), workers, func(v int) int { return v * 10 }))
			if !slices.Equal(got, []int{10, 20, 30, 40, 50, 60, 70, 80}) {
				t.Fatalf("workers=%d: got %v (want ordered)", workers, got)
			}
		}
		got := collect(Map(t.Context(), source(in...), 4, func(v int) int { return v * 10 }))
		slices.Sort(got)
		if !slices.Equal(got, []int{10, 20, 30, 40, 50, 60, 70, 80}) {
			t.Fatalf("workers=4: got %v", got)
		}
		if got := collect(Map(t.Context(), source[int](), 3, func(v int) int { return v })); len(got) != 0 {
			t.Fatalf("empty: got %v", got)
		}
	})
}

func TestMapRunsInParallel(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		// Every fn call waits for 3 to be in flight; this only completes if
		// the 3 workers run concurrently.
		var started sync.WaitGroup
		started.Add(3)
		out := Map(t.Context(), source(1, 2, 3), 3, func(v int) int {
			started.Done()
			started.Wait()
			return v
		})
		if got := collect(out); len(got) != 3 {
			t.Fatalf("got %v", got)
		}
	})
}

func TestFilter(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		got := collect(Filter(t.Context(), source(1, 2, 3, 4, 5, 6), func(v int) bool { return v%2 == 0 }))
		if !slices.Equal(got, []int{2, 4, 6}) {
			t.Fatalf("got %v", got)
		}
	})
}

func TestTake(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		tests := []struct {
			n    int
			in   []int
			want []int
		}{
			{n: 2, in: []int{1, 2, 3}, want: []int{1, 2}},
			{n: 5, in: []int{1, 2, 3}, want: []int{1, 2, 3}},
			{n: 3, in: []int{1, 2, 3}, want: []int{1, 2, 3}},
			{n: 0, in: nil, want: nil},
			{n: -1, in: nil, want: nil},
		}
		for _, tt := range tests {
			got := collect(Take(t.Context(), buffered(tt.in...), tt.n))
			if !slices.Equal(got, tt.want) {
				t.Errorf("Take(%v, %d) = %v; want %v", tt.in, tt.n, got, tt.want)
			}
		}

		// Take from an endless producer, then cancel to stop the producer.
		ctx, cancel := context.WithCancel(t.Context())
		got := collect(Take(ctx, endless(ctx), 4))
		cancel()
		if !slices.Equal(got, []int{0, 1, 2, 3}) {
			t.Fatalf("got %v", got)
		}
	})
}

func TestTee(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		if outs := Tee(t.Context(), source[int](), 0); outs != nil {
			t.Fatalf("Tee(0) = %v", outs)
		}

		outs := Tee(t.Context(), source(1, 2, 3, 4), 3)
		if len(outs) != 3 {
			t.Fatalf("len = %d", len(outs))
		}
		results := make([][]int, 3)
		var wg sync.WaitGroup
		for i, o := range outs {
			wg.Go(func() { results[i] = collect(o) })
		}
		wg.Wait()
		for i, r := range results {
			if !slices.Equal(r, []int{1, 2, 3, 4}) {
				t.Errorf("out %d = %v", i, r)
			}
		}
	})
}

func TestTeeAnyReadOrder(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		// A single goroutine reading the second output before the first must
		// not deadlock.
		outs := Tee(t.Context(), source("a", "b"), 2)
		for range 2 {
			if v := <-outs[1]; v == "" {
				t.Fatal("unexpected close")
			}
			if v := <-outs[0]; v == "" {
				t.Fatal("unexpected close")
			}
		}
		for _, o := range outs {
			if _, ok := <-o; ok {
				t.Fatal("output not closed")
			}
		}
	})
}

// TestPipelineCancelNoLeak builds a full pipeline over a never-ending source,
// abandons it midway and cancels ctx. synctest.Test fails if any stage
// goroutine is still blocked when the test function returns.
func TestPipelineCancelNoLeak(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		src := endless(ctx)
		stage := OrDone(ctx, src)
		stage = Filter(ctx, stage, func(v int) bool { return v%2 == 0 })
		mapped := Map(ctx, stage, 4, func(v int) int { return v * v })
		tees := Tee(ctx, mapped, 3)
		taken := Take(ctx, tees[0], 1000)

		// Drain two tee outputs; read only a few values from the third
		// (via Take) and then abandon it.
		for _, o := range tees[1:] {
			go func() {
				for range o {
				}
			}()
		}
		for range 5 {
			<-taken
		}
		cancel()
		synctest.Wait()
	})
}

func TestStagesStopOnClosedInputNoLeak(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		in := make(chan int)
		close(in)
		ctx := t.Context()
		for _, out := range []<-chan int{
			OrDone(ctx, in),
			Filter(ctx, in, func(int) bool { return true }),
			Map(ctx, in, 5, func(v int) int { return v }),
			Take(ctx, in, 3),
		} {
			if _, ok := <-out; ok {
				t.Fatal("value from closed input")
			}
		}
		for _, out := range Tee(ctx, in, 2) {
			if _, ok := <-out; ok {
				t.Fatal("value from closed input")
			}
		}
		synctest.Wait()
	})
}
