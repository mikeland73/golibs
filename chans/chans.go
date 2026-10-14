// Package chans provides context-aware channel operations, pipeline stages
// and a pub/sub Broadcaster.
//
//	if err := chans.Send(ctx, jobs, job); err != nil {
//		return err // ctx was cancelled before anyone took the job
//	}
//
//	nums := chans.OrDone(ctx, source)
//	evens := chans.Filter(ctx, nums, func(n int) bool { return n%2 == 0 })
//	squares := chans.Map(ctx, evens, 4, func(n int) int { return n * n })
//	for v := range chans.Take(ctx, squares, 10) { ... }
//
// # Pipeline contract
//
// Every stage (OrDone, Map, Filter, Take, Tee) starts goroutines that close
// the stage's output(s) and exit when either the input is closed and all
// values have been delivered, or ctx is done. A consumer that stops reading
// before the output is closed must cancel ctx, otherwise the stage blocks
// forever trying to deliver its next value. Stages never close their input.
// Values in flight when ctx is cancelled are dropped. Panics in user
// functions are not recovered.
//
// For FanIn, FanOut, SliceToChannel, ChannelToSlice, Generator, Buffer and
// Batch see github.com/samber/lo; for iterating a channel as an iter.Seq see
// lo/it.ChannelToSeq.
package chans

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"time"
)

// ErrClosed is returned by Recv when the channel is closed and drained, and
// by Broadcaster.Publish after the Broadcaster is closed.
var ErrClosed = errors.New("chans: channel closed")

// Send sends v on ch, blocking until the value is received (or buffered) or
// ctx is done. It returns ctx.Err() if ctx is done before the send happens;
// if ctx is already done, Send never sends. Like a plain send, Send panics if
// ch is closed and blocks forever (until ctx is done) if ch is nil.
func Send[T any](ctx context.Context, ch chan<- T, v T) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case ch <- v:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Recv receives a value from ch, blocking until one is available, ch is
// closed, or ctx is done. It returns ErrClosed once ch is closed and drained,
// and ctx.Err() if ctx is done first; if ctx is already done, Recv never
// receives.
//
// Recv returns a single error rather than Go's usual (v, ok) pair so a
// caller can handle "no value" with one check and use errors.Is to tell
// closure from cancellation.
func Recv[T any](ctx context.Context, ch <-chan T) (T, error) {
	var zero T
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	select {
	case v, ok := <-ch:
		if !ok {
			return zero, ErrClosed
		}
		return v, nil
	case <-ctx.Done():
		return zero, ctx.Err()
	}
}

// TrySend sends v on ch only if that can happen without blocking (a receiver
// is waiting or the buffer has room) and reports whether it did. Like a plain
// send, it panics if ch is closed.
func TrySend[T any](ch chan<- T, v T) bool {
	select {
	case ch <- v:
		return true
	default:
		return false
	}
}

// TryRecv receives a value from ch only if one is ready without blocking.
// ok is false if no value is ready or ch is closed and drained; use Recv or
// a plain receive if the two cases must be told apart.
func TryRecv[T any](ch <-chan T) (v T, ok bool) {
	select {
	case v, ok = <-ch:
		return v, ok
	default:
		return v, false
	}
}

// RecvTimeout receives a value from ch, waiting at most d. ok is false if
// the timeout expires first or ch is closed and drained. A value that is
// ready immediately is returned even if d <= 0.
func RecvTimeout[T any](ch <-chan T, d time.Duration) (v T, ok bool) {
	select {
	case v, ok = <-ch:
		return v, ok
	default:
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case v, ok = <-ch:
		return v, ok
	case <-t.C:
		return v, false
	}
}

// OrDone returns a channel that yields the values of in until in is closed
// or ctx is done, whichever is first. It is the basic building block for
// ranging over a channel that may never be closed:
//
//	for v := range chans.OrDone(ctx, in) { ... }
func OrDone[T any](ctx context.Context, in <-chan T) <-chan T {
	return Filter(ctx, in, nil)
}

// Map returns a channel of fn applied to every value from in, using workers
// goroutines (workers < 1 is treated as 1). With one worker the output order
// matches the input order; with more workers the output is unordered. The
// output is closed once all workers have finished.
func Map[T, R any](ctx context.Context, in <-chan T, workers int, fn func(T) R) <-chan R {
	workers = max(workers, 1)
	out := make(chan R)
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for {
				v, err := Recv(ctx, in)
				if err != nil {
					return
				}
				if Send(ctx, out, fn(v)) != nil {
					return
				}
			}
		})
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

// Filter returns a channel of the values from in for which pred returns
// true, in input order. A nil pred keeps every value.
func Filter[T any](ctx context.Context, in <-chan T, pred func(T) bool) <-chan T {
	out := make(chan T)
	go func() {
		defer close(out)
		for {
			v, err := Recv(ctx, in)
			if err != nil {
				return
			}
			if pred != nil && !pred(v) {
				continue
			}
			if Send(ctx, out, v) != nil {
				return
			}
		}
	}()
	return out
}

// Take returns a channel of at most the first n values from in. The output
// is closed after n values, when in is closed, or when ctx is done. Take
// stops reading from in after n values without draining it, so a producer
// feeding in should observe the same ctx (and the caller should cancel it)
// to avoid blocking forever. If n <= 0 the returned channel is already closed
// and in is not read.
func Take[T any](ctx context.Context, in <-chan T, n int) <-chan T {
	out := make(chan T)
	if n <= 0 {
		close(out)
		return out
	}
	go func() {
		defer close(out)
		for range n {
			v, err := Recv(ctx, in)
			if err != nil {
				return
			}
			if Send(ctx, out, v) != nil {
				return
			}
		}
	}()
	return out
}

// Tee returns n channels that each receive every value from in, in input
// order. Outputs are unbuffered and advance in lockstep: the next value is
// not read from in until every output has received the current one, so the
// slowest consumer sets the pace, but the outputs may be read in any order
// (or from different goroutines). Each consumer must keep reading until its
// output is closed or ctx is cancelled. If n <= 0 Tee returns nil and does
// not read in.
func Tee[T any](ctx context.Context, in <-chan T, n int) []<-chan T {
	if n <= 0 {
		return nil
	}
	outs := make([]chan T, n)
	res := make([]<-chan T, n)
	for i := range outs {
		outs[i] = make(chan T)
		res[i] = outs[i]
	}
	go func() {
		defer func() {
			for _, o := range outs {
				close(o)
			}
		}()
		// cases[0] is ctx.Done(); cases[1:] are the sends to each output.
		cases := make([]reflect.SelectCase, n+1)
		cases[0] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ctx.Done())}
		for {
			v, err := Recv(ctx, in)
			if err != nil {
				return
			}
			rv := reflect.ValueOf(&v).Elem()
			for i, o := range outs {
				cases[i+1] = reflect.SelectCase{Dir: reflect.SelectSend, Chan: reflect.ValueOf(o), Send: rv}
			}
			for range n {
				chosen, _, _ := reflect.Select(cases)
				if chosen == 0 {
					return
				}
				// Disable the case: a zero Chan is ignored by reflect.Select.
				cases[chosen].Chan = reflect.Value{}
			}
		}
	}()
	return res
}
