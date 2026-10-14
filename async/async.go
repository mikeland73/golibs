// Package async provides JavaScript Promise-like futures for Go: a value that
// will be produced by a goroutine, together with its error, plus combinators
// such as All, AllSettled, Race, Any and Then.
//
// Unlike lo.Async, which returns a bare channel, a Future carries an error,
// recovers panics (as *PanicError), can be awaited any number of times from
// any number of goroutines, and can be awaited with a context:
//
//	user := async.Go(func() (User, error) { return fetchUser(id) })
//	posts := async.GoCtx(ctx, func(ctx context.Context) ([]Post, error) { return fetchPosts(ctx, id) })
//	u, err := user.Await()
//	p, err := posts.AwaitCtx(ctx)
//
//	vals, err := async.All(ctx, f1, f2, f3) // results in input order
//
// Every combinator that takes a context stops waiting when the context is
// done; it never cancels or abandons the futures themselves (a Future's
// goroutine always runs fn to completion). To cancel the work, use GoCtx and
// cancel the context passed to it.
//
// For bounded parallel maps, worker pools and groups of goroutines use
// golang.org/x/sync/errgroup; for a single lazily computed value use
// sync.OnceValues.
package async

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
)

// ErrNoFutures is returned by Race and Any when called without any futures,
// since there is no value that could ever settle them.
var ErrNoFutures = errors.New("async: no futures")

// PanicError is the error a Future is rejected with when its function panics.
type PanicError struct {
	// Value is the value passed to panic.
	Value any
	// Stack is the stack trace of the panicking goroutine, as returned by
	// runtime/debug.Stack.
	Stack []byte
}

// Error implements the error interface. It reports the panic value but not
// the stack; use the Stack field for that.
func (e *PanicError) Error() string {
	return fmt.Sprintf("async: panic: %v", e.Value)
}

// Unwrap returns the panic value if it is an error, so that errors.Is and
// errors.As can see through a PanicError. Otherwise it returns nil.
func (e *PanicError) Unwrap() error {
	if err, ok := e.Value.(error); ok {
		return err
	}
	return nil
}

// errGoexit is used when fn calls runtime.Goexit (e.g. via t.FailNow) so the
// future still settles instead of blocking its waiters forever.
var errGoexit = errors.New("async: function called runtime.Goexit")

// Result is the outcome of a settled Future: either a value or an error.
type Result[T any] struct {
	Value T
	Err   error
}

// Future is the eventual result of an asynchronous computation. It is settled
// exactly once, with a value or an error, and is safe for concurrent use.
//
// Futures are created with Go, GoCtx, Resolved, Rejected or Then. The zero
// value is not usable.
type Future[T any] struct {
	done chan struct{}
	val  T
	err  error
}

func newFuture[T any]() *Future[T] {
	return &Future[T]{done: make(chan struct{})}
}

// run calls fn, recording its result (or a *PanicError) and settling f.
func (f *Future[T]) run(fn func() (T, error)) {
	normal := false
	defer func() {
		if !normal {
			if r := recover(); r != nil {
				f.err = &PanicError{Value: r, Stack: debug.Stack()}
			} else {
				f.err = errGoexit
			}
		}
		close(f.done)
	}()
	f.val, f.err = fn()
	normal = true
}

// Go runs fn in a new goroutine and returns a Future for its result. If fn
// panics the Future is rejected with a *PanicError.
func Go[T any](fn func() (T, error)) *Future[T] {
	f := newFuture[T]()
	go f.run(fn)
	return f
}

// GoCtx runs fn(ctx) in a new goroutine and returns a Future for its result.
// If ctx is already done, fn is not called and the Future is rejected with
// ctx.Err(). Otherwise the Future settles when fn returns; fn is responsible
// for honoring cancellation of ctx. If fn panics the Future is rejected with a
// *PanicError.
func GoCtx[T any](ctx context.Context, fn func(context.Context) (T, error)) *Future[T] {
	if err := ctx.Err(); err != nil {
		return Rejected[T](err)
	}
	return Go(func() (T, error) { return fn(ctx) })
}

// Resolved returns an already settled Future holding v.
func Resolved[T any](v T) *Future[T] {
	f := newFuture[T]()
	f.val = v
	close(f.done)
	return f
}

// Rejected returns an already settled Future holding err. If err is nil, the
// Future is resolved with the zero value of T.
func Rejected[T any](err error) *Future[T] {
	f := newFuture[T]()
	f.err = err
	close(f.done)
	return f
}

// Await blocks until f settles and returns its value and error. It may be
// called any number of times, from any number of goroutines; every call
// returns the same result.
func (f *Future[T]) Await() (T, error) {
	<-f.done
	return f.val, f.err
}

// AwaitCtx is like Await but stops waiting when ctx is done, returning the
// zero value and ctx.Err(). The Future itself keeps running. If f is already
// settled its result is returned even if ctx is done.
func (f *Future[T]) AwaitCtx(ctx context.Context) (T, error) {
	select {
	case <-f.done:
		return f.val, f.err
	default:
	}
	select {
	case <-f.done:
		return f.val, f.err
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	}
}

// Done returns a channel that is closed when f settles. It is useful in a
// select statement; after it is closed, Await returns immediately.
func (f *Future[T]) Done() <-chan struct{} {
	return f.done
}

// Then returns a Future that, once f resolves successfully, runs fn on its
// value. If f is rejected, the returned Future is rejected with the same
// error and fn is not called. A panic in fn rejects the returned Future with a
// *PanicError.
//
// The goroutine started by Then lives until f settles.
func Then[T, R any](f *Future[T], fn func(T) (R, error)) *Future[R] {
	out := newFuture[R]()
	go out.run(func() (R, error) {
		v, err := f.Await()
		if err != nil {
			var zero R
			return zero, err
		}
		return fn(v)
	})
	return out
}

// watch reports the index of each future on the returned channel once it
// settles. Futures that are already settled are reported immediately, in
// input order; the rest are watched by one goroutine each, which exits when
// its future settles or when stop is called, whichever is first. The channel
// is buffered so reporting never blocks. Callers must call stop.
func watch[T any](ctx context.Context, fs []*Future[T]) (settled <-chan int, stop context.CancelFunc) {
	ctx, stop = context.WithCancel(ctx)
	ch := make(chan int, len(fs))
	for i, f := range fs {
		select {
		case <-f.done:
			ch <- i
			continue
		default:
		}
		go func() {
			select {
			case <-f.done:
				ch <- i
			case <-ctx.Done():
			}
		}()
	}
	return ch, stop
}

// next returns the next index from settled, or ctx.Err() if ctx is done
// first. Like AwaitCtx, a future that has already settled wins over a done
// ctx.
func next(ctx context.Context, settled <-chan int) (int, error) {
	select {
	case i := <-settled:
		return i, nil
	default:
	}
	select {
	case i := <-settled:
		return i, nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

// All waits for all futures and returns their values in input order. It
// returns early with the first error it observes if any future is rejected
// (already-rejected futures are observed first, in input order), or with
// ctx.Err() if ctx is done first. With no futures it
// returns an empty slice and nil.
func All[T any](ctx context.Context, fs ...*Future[T]) ([]T, error) {
	settled, stop := watch(ctx, fs)
	defer stop()
	out := make([]T, len(fs))
	for range fs {
		i, err := next(ctx, settled)
		if err != nil {
			return nil, err
		}
		if err := fs[i].err; err != nil {
			return nil, err
		}
		out[i] = fs[i].val
	}
	return out, nil
}

// AllSettled waits for all futures and returns their results in input order,
// never short-circuiting on errors. If ctx is done before every future
// settles, the results of the futures still pending have Err set to
// ctx.Err(). With no futures it returns an empty slice.
func AllSettled[T any](ctx context.Context, fs ...*Future[T]) []Result[T] {
	out := make([]Result[T], len(fs))
	for i, f := range fs {
		v, err := f.AwaitCtx(ctx)
		out[i] = Result[T]{Value: v, Err: err}
	}
	return out
}

// Race returns the result of the first future to settle, whether it resolved
// or was rejected. It returns ctx.Err() if ctx is done first, and
// ErrNoFutures if called with no futures. If several futures are already
// settled, the first of them in input order wins.
func Race[T any](ctx context.Context, fs ...*Future[T]) (T, error) {
	var zero T
	if len(fs) == 0 {
		return zero, ErrNoFutures
	}
	settled, stop := watch(ctx, fs)
	defer stop()
	i, err := next(ctx, settled)
	if err != nil {
		return zero, err
	}
	return fs[i].val, fs[i].err
}

// Any returns the value of the first future to resolve successfully. If every
// future is rejected it returns errors.Join of all their errors in input
// order. It returns ctx.Err() if ctx is done first, and ErrNoFutures if
// called with no futures.
func Any[T any](ctx context.Context, fs ...*Future[T]) (T, error) {
	var zero T
	if len(fs) == 0 {
		return zero, ErrNoFutures
	}
	settled, stop := watch(ctx, fs)
	defer stop()
	for range fs {
		i, err := next(ctx, settled)
		if err != nil {
			return zero, err
		}
		if fs[i].err == nil {
			return fs[i].val, nil
		}
	}
	errs := make([]error, len(fs))
	for i, f := range fs {
		errs[i] = f.err
	}
	return zero, errors.Join(errs...)
}
