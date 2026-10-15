// Package conc provides small, safe concurrency helpers: a bounded parallel
// Map/ForEach over a slice, values guarded by a mutex (Mutex, RWMutex) and a
// concurrency-safe map (SafeMap).
//
//	pages, err := conc.Map(ctx, urls, 8, func(ctx context.Context, url string) (string, error) {
//		return fetch(ctx, url)
//	})
//
//	var hits conc.SafeMap[string, int]
//	hits.Update("/home", func(n int, _ bool) int { return n + 1 })
//
// Panics in functions run by Map and ForEach are recovered and returned as a
// *PanicError.
//
// Map and ForEach are built on golang.org/x/sync/errgroup; use errgroup
// directly (with SetLimit / TryGo) for open-ended groups of goroutines and
// worker pools. Other related tools that are deliberately not duplicated here:
//   - golang.org/x/sync/semaphore for weighted concurrency limits;
//   - golang.org/x/sync/singleflight for de-duplicating concurrent calls;
//   - golang.org/x/time/rate for rate limiting;
//   - sync.OnceValue / sync.OnceValues for lazy one-time initialization;
//   - sync.WaitGroup.Go for fire-and-wait goroutines;
//   - sync.Map for append-mostly caches with disjoint keys.
package conc

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
)

// PanicError is the error a recovered panic is reported as, by Map and
// ForEach here and by futures in the async package.
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
	return fmt.Sprintf("panic: %v", e.Value)
}

// Unwrap returns the panic value if it is an error, so that errors.Is and
// errors.As can see through a PanicError. Otherwise it returns nil.
func (e *PanicError) Unwrap() error {
	if err, ok := e.Value.(error); ok {
		return err
	}
	return nil
}

var errGoexit = errors.New("conc: function called runtime.Goexit")

// call runs fn and returns its error, or a *PanicError if fn panics. If fn
// calls runtime.Goexit, onGoexit is called before the goroutine exits (and
// call never returns), so the failure is still observed.
func call(fn func() error, onGoexit func()) (err error) {
	normal := false
	defer func() {
		if normal {
			return
		}
		if r := recover(); r != nil {
			err = &PanicError{Value: r, Stack: debug.Stack()}
			return
		}
		onGoexit()
	}()
	err = fn()
	normal = true
	return err
}

// Map calls fn on every item concurrently, with at most limit calls in
// flight at once (limit <= 0 means no limit), and returns the results in
// input order.
//
// The first error (including a recovered panic, as *PanicError) cancels the
// context passed to the in-flight calls, stops new calls from starting, and
// is returned once all in-flight calls have finished; the result slice is nil
// in that case. If ctx is done before all items have started, Map returns
// ctx.Err(), unless an in-flight call failed first, in which case that
// call's error is returned. Map never leaves goroutines running after it
// returns. With no items it returns an empty slice and nil.
func Map[T, R any](ctx context.Context, items []T, limit int, fn func(context.Context, T) (R, error)) ([]R, error) {
	out := make([]R, len(items))
	err := run(ctx, len(items), limit, func(ctx context.Context, i int) error {
		r, err := fn(ctx, items[i])
		out[i] = r
		return err
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ForEach calls fn on every item concurrently with at most limit calls in
// flight at once (limit <= 0 means no limit). Error, cancellation and panic
// semantics are the same as for Map. With no items it returns nil.
func ForEach[T any](ctx context.Context, items []T, limit int, fn func(context.Context, T) error) error {
	return run(ctx, len(items), limit, func(ctx context.Context, i int) error {
		return fn(ctx, items[i])
	})
}

// run calls fn(ctx, i) for i in [0, n) with at most limit calls in flight,
// on top of errgroup (which cancels ctx on the first error).
func run(ctx context.Context, n, limit int, fn func(context.Context, int) error) error {
	parent := ctx
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	g, ctx := errgroup.WithContext(ctx)
	if limit > 0 {
		g.SetLimit(limit)
	}
	// errgroup never sees an error from a goroutine that called Goexit, so
	// record it here and cancel the remaining calls ourselves.
	var goexited atomic.Bool
	var ran atomic.Int64
	for i := range n {
		if ctx.Err() != nil {
			break // a call failed or the parent ctx is done
		}
		g.Go(func() error {
			// g.Go may have waited for a free slot; don't start new work
			// once a call has failed or the parent ctx is done.
			if ctx.Err() != nil {
				return nil
			}
			ran.Add(1)
			return call(func() error { return fn(ctx, i) }, func() {
				goexited.Store(true)
				cancel(errGoexit)
			})
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}
	if goexited.Load() {
		return errGoexit
	}
	if int(ran.Load()) < n {
		return parent.Err()
	}
	return nil
}

// Mutex holds a value of type T guarded by a sync.Mutex, in the spirit of
// Rust's Mutex<T>: the value can only be reached while the lock is held.
//
// The zero value is an unlocked Mutex holding the zero value of T. A Mutex
// must not be copied after first use. Functions passed to With must not call
// methods on the same Mutex (that would deadlock) and must not retain the
// pointer they are given after returning.
type Mutex[T any] struct {
	mu sync.Mutex
	v  T
}

// NewMutex returns a Mutex holding v.
func NewMutex[T any](v T) *Mutex[T] {
	return &Mutex[T]{v: v}
}

// With calls fn with a pointer to the guarded value while holding the lock,
// allowing an atomic read-modify-write of the value.
func (m *Mutex[T]) With(fn func(v *T)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	fn(&m.v)
}

// Get returns a copy of the guarded value. Note that for reference types
// (maps, slices, pointers) the copy is shallow; use With to access their
// contents safely.
func (m *Mutex[T]) Get() T {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.v
}

// Set replaces the guarded value with v.
func (m *Mutex[T]) Set(v T) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.v = v
}

// Swap replaces the guarded value with v and returns the previous value.
func (m *Mutex[T]) Swap(v T) (old T) {
	m.mu.Lock()
	defer m.mu.Unlock()
	old, m.v = m.v, v
	return old
}

// RWMutex holds a value of type T guarded by a sync.RWMutex, allowing many
// concurrent readers or one writer.
//
// The zero value is an unlocked RWMutex holding the zero value of T. An
// RWMutex must not be copied after first use. Functions passed to With and
// Read must not call methods on the same RWMutex (that would deadlock) and
// must not retain references to the value after returning.
type RWMutex[T any] struct {
	mu sync.RWMutex
	v  T
}

// NewRWMutex returns an RWMutex holding v.
func NewRWMutex[T any](v T) *RWMutex[T] {
	return &RWMutex[T]{v: v}
}

// With calls fn with a pointer to the guarded value while holding the write
// lock, allowing an atomic read-modify-write of the value.
func (m *RWMutex[T]) With(fn func(v *T)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	fn(&m.v)
}

// Read calls fn with the guarded value while holding the read lock. Other
// readers may run concurrently. fn receives a (shallow) copy; it must not
// modify the contents of reference types such as maps or slices.
func (m *RWMutex[T]) Read(fn func(v T)) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	fn(m.v)
}

// Get returns a (shallow) copy of the guarded value under the read lock.
func (m *RWMutex[T]) Get() T {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.v
}

// Set replaces the guarded value with v.
func (m *RWMutex[T]) Set(v T) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.v = v
}

// Swap replaces the guarded value with v and returns the previous value.
func (m *RWMutex[T]) Swap(v T) (old T) {
	m.mu.Lock()
	defer m.mu.Unlock()
	old, m.v = m.v, v
	return old
}
