package async

import (
	"context"
	"errors"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

var (
	errA = errors.New("a")
	errB = errors.New("b")
)

// gate returns a future that settles with (v, err) once release is closed.
func gate[T any](release <-chan struct{}, v T, err error) *Future[T] {
	return Go(func() (T, error) {
		<-release
		return v, err
	})
}

func TestGoResolveReject(t *testing.T) {
	t.Parallel()
	v, err := Go(func() (int, error) { return 42, nil }).Await()
	if v != 42 || err != nil {
		t.Fatalf("Await = %v, %v; want 42, nil", v, err)
	}
	v, err = Go(func() (int, error) { return 7, errA }).Await()
	if v != 7 || !errors.Is(err, errA) {
		t.Fatalf("Await = %v, %v; want 7, a", v, err)
	}
}

func TestResolvedRejected(t *testing.T) {
	t.Parallel()
	f := Resolved("x")
	select {
	case <-f.Done():
	default:
		t.Fatal("Resolved future not done")
	}
	if v, err := f.Await(); v != "x" || err != nil {
		t.Fatalf("Resolved.Await = %q, %v", v, err)
	}
	if v, err := Rejected[string](errA).Await(); v != "" || !errors.Is(err, errA) {
		t.Fatalf("Rejected.Await = %q, %v", v, err)
	}
	if v, err := Rejected[int](nil).Await(); v != 0 || err != nil {
		t.Fatalf("Rejected(nil).Await = %v, %v", v, err)
	}
}

func TestAwaitManyGoroutines(t *testing.T) {
	t.Parallel()
	release := make(chan struct{})
	f := gate(release, 5, nil)
	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			if v, err := f.Await(); v != 5 || err != nil {
				t.Errorf("Await = %v, %v", v, err)
			}
		})
	}
	close(release)
	wg.Wait()
	// Repeated awaits return the same result.
	for range 3 {
		if v, _ := f.Await(); v != 5 {
			t.Fatalf("Await = %v", v)
		}
	}
}

func TestPanicRecovered(t *testing.T) {
	t.Parallel()
	_, err := Go(func() (int, error) { panic("boom") }).Await()
	var pe *PanicError
	if !errors.As(err, &pe) {
		t.Fatalf("err = %v (%T); want *PanicError", err, err)
	}
	if pe.Value != "boom" {
		t.Errorf("Value = %v", pe.Value)
	}
	if !strings.Contains(string(pe.Stack), "TestPanicRecovered") {
		t.Errorf("stack does not mention test function:\n%s", pe.Stack)
	}
	if got := err.Error(); got != "async: panic: boom" {
		t.Errorf("Error() = %q", got)
	}
	if pe.Unwrap() != nil {
		t.Errorf("Unwrap of non-error panic = %v", pe.Unwrap())
	}

	// Panicking with an error is unwrappable.
	_, err = Go(func() (int, error) { panic(errA) }).Await()
	if !errors.Is(err, errA) {
		t.Errorf("errors.Is(panic(errA), errA) = false; err = %v", err)
	}
}

func TestGoexitSettles(t *testing.T) {
	t.Parallel()
	_, err := Go(func() (int, error) {
		runtime.Goexit()
		return 0, nil
	}).Await()
	if !errors.Is(err, errGoexit) {
		t.Fatalf("err = %v; want errGoexit", err)
	}
}

func TestGoCtx(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		f := GoCtx(ctx, func(ctx context.Context) (int, error) {
			<-ctx.Done()
			return 0, ctx.Err()
		})
		synctest.Wait()
		select {
		case <-f.Done():
			t.Fatal("settled before cancel")
		default:
		}
		cancel()
		if _, err := f.Await(); !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v", err)
		}

		// Already-cancelled ctx: fn is never called.
		called := false
		_, err := GoCtx(ctx, func(context.Context) (int, error) {
			called = true
			return 1, nil
		}).Await()
		if called || !errors.Is(err, context.Canceled) {
			t.Fatalf("called=%v err=%v", called, err)
		}
	})
}

func TestAwaitCtx(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		f := gate(release, 1, nil)
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		if _, err := f.AwaitCtx(ctx); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("err = %v", err)
		}
		close(release)
		synctest.Wait()
		// Settled futures win even with a done ctx.
		if v, err := f.AwaitCtx(ctx); v != 1 || err != nil {
			t.Fatalf("AwaitCtx = %v, %v", v, err)
		}
	})
}

func TestThen(t *testing.T) {
	t.Parallel()
	f := Then(Resolved(20), func(v int) (string, error) {
		return strings.Repeat("a", v/10), nil
	})
	if v, err := f.Await(); v != "aa" || err != nil {
		t.Fatalf("Then = %q, %v", v, err)
	}

	called := false
	_, err := Then(Rejected[int](errA), func(int) (int, error) {
		called = true
		return 0, nil
	}).Await()
	if called || !errors.Is(err, errA) {
		t.Fatalf("called=%v err=%v", called, err)
	}

	_, err = Then(Resolved(1), func(int) (int, error) { panic("then") }).Await()
	var pe *PanicError
	if !errors.As(err, &pe) || pe.Value != "then" {
		t.Fatalf("err = %v", err)
	}

	_, err = Then(Resolved(1), func(int) (int, error) { return 0, errB }).Await()
	if !errors.Is(err, errB) {
		t.Fatalf("err = %v", err)
	}
}

func TestAll(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		ctx := t.Context()

		got, err := All[int](ctx)
		if got == nil || len(got) != 0 || err != nil {
			t.Fatalf("All() = %#v, %v; want [], nil", got, err)
		}

		// Results are in input order regardless of completion order.
		r1, r2, r3 := make(chan struct{}), make(chan struct{}), make(chan struct{})
		fs := []*Future[int]{gate(r1, 1, nil), gate(r2, 2, nil), gate(r3, 3, nil)}
		close(r3)
		synctest.Wait()
		close(r1)
		synctest.Wait()
		close(r2)
		got, err = All(ctx, fs...)
		if err != nil || len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 3 {
			t.Fatalf("All = %v, %v", got, err)
		}

		// Returns as soon as one fails, without waiting for the rest.
		never := make(chan struct{})
		got, err = All(ctx, gate(never, 1, nil), Rejected[int](errA))
		if got != nil || !errors.Is(err, errA) {
			t.Fatalf("All = %v, %v; want nil, a", got, err)
		}

		// Returns on ctx cancellation.
		cctx, cancel := context.WithCancel(ctx)
		go func() {
			time.Sleep(time.Second)
			cancel()
		}()
		_, err = All(cctx, gate(never, 1, nil))
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v", err)
		}
		close(never) // let gate goroutines finish so the bubble can exit.
	})
}

func TestAllSettled(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		if got := AllSettled[int](t.Context()); got == nil || len(got) != 0 {
			t.Fatalf("AllSettled() = %#v", got)
		}
		res := AllSettled(t.Context(), Resolved(1), Rejected[int](errA), Go(func() (int, error) { panic("p") }))
		if len(res) != 3 || res[0] != (Result[int]{Value: 1}) || !errors.Is(res[1].Err, errA) {
			t.Fatalf("res = %+v", res)
		}
		var pe *PanicError
		if !errors.As(res[2].Err, &pe) {
			t.Fatalf("res[2] = %+v", res[2])
		}

		never := make(chan struct{})
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		res = AllSettled(ctx, Resolved(1), gate(never, 2, nil))
		if res[0] != (Result[int]{Value: 1}) || !errors.Is(res[1].Err, context.DeadlineExceeded) {
			t.Fatalf("res = %+v", res)
		}
		close(never)
	})
}

func TestRace(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		ctx := t.Context()
		if _, err := Race[int](ctx); !errors.Is(err, ErrNoFutures) {
			t.Fatalf("Race() err = %v", err)
		}

		slow, fast := make(chan struct{}), make(chan struct{})
		fs := []*Future[int]{gate(slow, 1, nil), gate(fast, 2, errB)}
		go func() {
			time.Sleep(time.Second)
			close(fast)
			time.Sleep(time.Second)
			close(slow)
		}()
		// First settled wins, even if it is a rejection.
		if v, err := Race(ctx, fs...); v != 2 || !errors.Is(err, errB) {
			t.Fatalf("Race = %v, %v", v, err)
		}

		never := make(chan struct{})
		cctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		if _, err := Race(cctx, gate(never, 1, nil)); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("err = %v", err)
		}
		close(never)
	})
}

func TestAny(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		ctx := t.Context()
		if _, err := Any[int](ctx); !errors.Is(err, ErrNoFutures) {
			t.Fatalf("Any() err = %v", err)
		}

		// First success wins; earlier and later failures are ignored.
		r := make(chan struct{})
		go func() {
			time.Sleep(time.Second)
			close(r)
		}()
		v, err := Any(ctx, Rejected[int](errA), gate(r, 5, nil), Rejected[int](errB))
		if v != 5 || err != nil {
			t.Fatalf("Any = %v, %v", v, err)
		}

		// All fail: joined errors in input order.
		_, err = Any(ctx, Rejected[int](errA), Rejected[int](errB))
		if !errors.Is(err, errA) || !errors.Is(err, errB) || err.Error() != "a\nb" {
			t.Fatalf("err = %q", err)
		}

		never := make(chan struct{})
		cctx, cancel := context.WithCancel(ctx)
		cancel()
		if _, err := Any(cctx, gate(never, 1, nil)); !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v", err)
		}
		close(never)
	})
}

// TestCombinatorsNoLeak verifies that combinators returning early (due to an
// error or ctx) do not leave watcher goroutines behind: synctest.Test fails
// if any goroutine in the bubble is still blocked when the test ends.
func TestCombinatorsNoLeak(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		never := make(chan struct{})
		pending := gate(never, 0, nil)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		_, _ = All(t.Context(), pending, Rejected[int](errA))
		_, _ = Race(t.Context(), pending, Resolved(1))
		_, _ = Any(t.Context(), pending, Resolved(1))
		_, _ = All(ctx, pending)
		_, _ = Race(ctx, pending)
		_, _ = Any(ctx, pending)
		_ = AllSettled(ctx, pending)
		synctest.Wait()
		// Only the pending future's own goroutine may remain; release it.
		close(never)
	})
}

func TestCombinatorsPreferSettledOverDoneCtx(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	boom := errors.New("boom")
	for range 100 {
		if v, err := Race(ctx, Resolved(1), Resolved(2)); v != 1 || err != nil {
			t.Fatalf("Race = %v, %v; want 1, nil", v, err)
		}
		if v, err := Any(ctx, Rejected[int](boom), Resolved(2)); v != 2 || err != nil {
			t.Fatalf("Any = %v, %v; want 2, nil", v, err)
		}
		if v, err := All(ctx, Resolved(1), Resolved(2)); !slices.Equal(v, []int{1, 2}) || err != nil {
			t.Fatalf("All = %v, %v; want [1 2], nil", v, err)
		}
		if _, err := All(ctx, Resolved(1), Rejected[int](boom)); !errors.Is(err, boom) {
			t.Fatalf("All err = %v; want boom", err)
		}
	}
}
