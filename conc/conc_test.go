package conc

import (
	"context"
	"errors"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

var errBoom = errors.New("boom")

func TestMapOrderAndValues(t *testing.T) {
	t.Parallel()
	items := make([]int, 100)
	for i := range items {
		items[i] = i
	}
	for _, limit := range []int{-1, 0, 1, 3, 100, 1000} {
		got, err := Map(t.Context(), items, limit, func(_ context.Context, v int) (int, error) {
			runtime.Gosched()
			return v * v, nil
		})
		if err != nil {
			t.Fatalf("limit %d: err = %v", limit, err)
		}
		for i, v := range got {
			if v != i*i {
				t.Fatalf("limit %d: got[%d] = %d", limit, i, v)
			}
		}
	}
}

func TestMapEmpty(t *testing.T) {
	t.Parallel()
	got, err := Map(t.Context(), nil, 4, func(context.Context, int) (int, error) {
		t.Fatal("fn called")
		return 0, nil
	})
	if got == nil || len(got) != 0 || err != nil {
		t.Fatalf("Map(nil) = %#v, %v", got, err)
	}
	if err := ForEach(t.Context(), []int{}, 0, func(context.Context, int) error {
		t.Fatal("fn called")
		return nil
	}); err != nil {
		t.Fatalf("ForEach(empty) = %v", err)
	}
}

func TestMapRespectsLimit(t *testing.T) {
	t.Parallel()
	for _, limit := range []int{1, 2, 5} {
		synctest.Test(t, func(t *testing.T) {
			var cur, peak atomic.Int64
			items := make([]int, 20)
			_, err := Map(t.Context(), items, limit, func(_ context.Context, v int) (int, error) {
				n := cur.Add(1)
				for {
					p := peak.Load()
					if n <= p || peak.CompareAndSwap(p, n) {
						break
					}
				}
				time.Sleep(time.Millisecond) // fake clock: lets other workers start
				cur.Add(-1)
				return v, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if got := peak.Load(); got != int64(limit) {
				t.Fatalf("limit %d: peak concurrency %d", limit, got)
			}
		})
	}
}

func TestMapUnboundedRunsAllConcurrently(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		// Every call waits for all others to start; this only finishes if
		// all 50 run at once.
		const n = 50
		var started sync.WaitGroup
		started.Add(n)
		_, err := Map(t.Context(), make([]int, n), 0, func(context.Context, int) (int, error) {
			started.Done()
			started.Wait()
			return 0, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestMapFirstErrorCancels(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		var calls atomic.Int64
		var canceled atomic.Int64
		items := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
		got, err := Map(t.Context(), items, 3, func(ctx context.Context, v int) (int, error) {
			calls.Add(1)
			if v == 1 {
				time.Sleep(time.Second)
				return 0, errBoom
			}
			// Others block until cancelled.
			<-ctx.Done()
			canceled.Add(1)
			return 0, ctx.Err()
		})
		if got != nil || !errors.Is(err, errBoom) {
			t.Fatalf("Map = %v, %v; want nil, boom", got, err)
		}
		if c := calls.Load(); c != 3 {
			t.Errorf("calls = %d; want 3 (no new calls after failure)", c)
		}
		if c := canceled.Load(); c != 2 {
			t.Errorf("canceled = %d; want 2", c)
		}
	})
}

func TestMapParentCancel(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		var calls atomic.Int64
		go func() {
			time.Sleep(time.Second)
			cancel()
		}()
		err := ForEach(ctx, make([]int, 10), 2, func(ctx context.Context, _ int) error {
			calls.Add(1)
			select {
			case <-ctx.Done():
				return nil // swallow: Map must still report cancellation.
			case <-time.After(time.Hour):
				return nil
			}
		})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v; want Canceled", err)
		}
		if c := calls.Load(); c != 2 {
			t.Errorf("calls = %d; want 2", c)
		}

		// Already cancelled: fn never called.
		err = ForEach(ctx, []int{1}, 1, func(context.Context, int) error {
			t.Error("fn called")
			return nil
		})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestMapPanic(t *testing.T) {
	t.Parallel()
	_, err := Map(t.Context(), []int{1, 2, 3}, 0, func(_ context.Context, v int) (int, error) {
		if v == 2 {
			panic("two")
		}
		return v, nil
	})
	var pe *PanicError
	if !errors.As(err, &pe) || pe.Value != "two" {
		t.Fatalf("err = %v (%T)", err, err)
	}
	if err.Error() != "conc: panic: two" {
		t.Errorf("Error() = %q", err.Error())
	}
	if !strings.Contains(string(pe.Stack), "TestMapPanic") {
		t.Errorf("stack missing test frame:\n%s", pe.Stack)
	}
	if pe.Unwrap() != nil {
		t.Errorf("Unwrap() = %v", pe.Unwrap())
	}

	err = ForEach(t.Context(), []int{1}, 1, func(context.Context, int) error { panic(errBoom) })
	if !errors.Is(err, errBoom) {
		t.Fatalf("err = %v; want wrapping boom", err)
	}

	err = ForEach(t.Context(), []int{1}, 1, func(context.Context, int) error {
		runtime.Goexit()
		return nil
	})
	if !errors.Is(err, errGoexit) {
		t.Fatalf("err = %v; want errGoexit", err)
	}
}

func TestMapNoGoroutineLeak(t *testing.T) {
	t.Parallel()
	// synctest.Test fails if goroutines started in the bubble are still
	// blocked when the function returns.
	synctest.Test(t, func(t *testing.T) {
		_ = ForEach(t.Context(), []int{0, 1, 2, 3, 4, 5}, 4, func(ctx context.Context, v int) error {
			if v == 2 {
				return errBoom
			}
			<-ctx.Done()
			return nil
		})
		_, _ = Map(t.Context(), make([]int, 100), 0, func(context.Context, int) (int, error) {
			return 0, nil
		})
		synctest.Wait()
	})
}

func TestMutex(t *testing.T) {
	t.Parallel()
	var m Mutex[int] // zero value usable
	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() { m.With(func(v *int) { *v++ }) })
	}
	wg.Wait()
	if got := m.Get(); got != 100 {
		t.Fatalf("Get = %d", got)
	}
	m.Set(5)
	if old := m.Swap(6); old != 5 {
		t.Fatalf("Swap old = %d", old)
	}
	if got := m.Get(); got != 6 {
		t.Fatalf("Get = %d", got)
	}

	sm := NewMutex([]string{"a"})
	sm.With(func(v *[]string) { *v = append(*v, "b") })
	if got := sm.Get(); !slices.Equal(got, []string{"a", "b"}) {
		t.Fatalf("Get = %v", got)
	}
}

func TestRWMutex(t *testing.T) {
	t.Parallel()
	var m RWMutex[map[string]int]
	m.Set(map[string]int{})
	var wg sync.WaitGroup
	for i := range 50 {
		wg.Go(func() {
			if i%2 == 0 {
				m.With(func(v *map[string]int) { (*v)["n"]++ })
			} else {
				m.Read(func(v map[string]int) { _ = v["n"] })
			}
		})
	}
	wg.Wait()
	var n int
	m.Read(func(v map[string]int) { n = v["n"] })
	if n != 25 {
		t.Fatalf("n = %d", n)
	}

	r := NewRWMutex("x")
	if old := r.Swap("y"); old != "x" || r.Get() != "y" {
		t.Fatalf("Swap/Get = %q/%q", old, r.Get())
	}
}

func TestRWMutexConcurrentReaders(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		// Two readers must be able to hold the read lock at the same time.
		m := NewRWMutex(1)
		var both sync.WaitGroup
		both.Add(2)
		var wg sync.WaitGroup
		for range 2 {
			wg.Go(func() {
				m.Read(func(int) {
					both.Done()
					both.Wait()
				})
			})
		}
		wg.Wait()
	})
}

func TestSafeMap(t *testing.T) {
	t.Parallel()
	var s SafeMap[string, int] // zero value usable
	if _, ok := s.Get("a"); ok || !s.IsEmpty() || s.Len() != 0 || len(s.Keys()) != 0 {
		t.Fatal("zero SafeMap not empty")
	}
	s.Delete("missing") // no-op on nil map
	s.Clear()
	if got := s.String(); got != "SafeMap[]" {
		t.Fatalf("String = %q", got)
	}

	s.Set("a", 1)
	s.Set("b", 2)
	if v, ok := s.Get("a"); !ok || v != 1 {
		t.Fatalf("Get(a) = %v, %v", v, ok)
	}
	if v, loaded := s.GetOrSet("a", 10); !loaded || v != 1 {
		t.Fatalf("GetOrSet(a) = %v, %v", v, loaded)
	}
	if v, loaded := s.GetOrSet("c", 3); loaded || v != 3 {
		t.Fatalf("GetOrSet(c) = %v, %v", v, loaded)
	}
	if got := s.Update("c", func(old int, ok bool) int {
		if !ok {
			t.Error("Update: ok = false for present key")
		}
		return old * 10
	}); got != 30 {
		t.Fatalf("Update = %d", got)
	}
	if got := s.Update("d", func(old int, ok bool) int {
		if ok || old != 0 {
			t.Errorf("Update absent: old=%d ok=%v", old, ok)
		}
		return 4
	}); got != 4 {
		t.Fatalf("Update = %d", got)
	}
	if got := s.String(); got != "SafeMap[a:1 b:2 c:30 d:4]" {
		t.Fatalf("String = %q", got)
	}
	keys := s.Keys()
	slices.Sort(keys)
	if !slices.Equal(keys, []string{"a", "b", "c", "d"}) {
		t.Fatalf("Keys = %v", keys)
	}
	s.Delete("b")
	if _, ok := s.Get("b"); ok || s.Len() != 3 {
		t.Fatalf("after Delete: Len = %d", s.Len())
	}
	s.Clear()
	if !s.IsEmpty() {
		t.Fatal("not empty after Clear")
	}

	var u SafeMap[int, int]
	if got := u.Update(1, func(int, bool) int { return 7 }); got != 7 {
		t.Fatalf("Update on zero map = %d", got)
	}
	var g SafeMap[int, int]
	if v, loaded := g.GetOrSet(1, 2); loaded || v != 2 {
		t.Fatalf("GetOrSet on zero map = %v, %v", v, loaded)
	}
}

func TestSafeMapAllSnapshot(t *testing.T) {
	t.Parallel()
	s := NewSafeMap[int, int]()
	for i := range 10 {
		s.Set(i, i*i)
	}
	seen := 0
	for k, v := range s.All() {
		if v != k*k {
			t.Fatalf("All yielded %d:%d", k, v)
		}
		// Mutating during iteration must not deadlock nor affect the snapshot.
		s.Set(k+100, 0)
		s.Delete(k)
		seen++
	}
	if seen != 10 {
		t.Fatalf("seen = %d", seen)
	}
	if s.Len() != 10 {
		t.Fatalf("Len = %d", s.Len())
	}
	// Early break.
	n := 0
	for range s.All() {
		n++
		break
	}
	if n != 1 {
		t.Fatalf("n = %d", n)
	}
}

func TestSafeMapConcurrent(t *testing.T) {
	t.Parallel()
	var s SafeMap[int, int]
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Go(func() {
			for j := range 1000 {
				s.Update(j%10, func(old int, _ bool) int { return old + 1 })
				s.GetOrSet(100+i, j)
				s.Get(j % 10)
				_ = s.Len()
				if j%100 == 0 {
					_ = s.Keys()
					for range s.All() {
						break
					}
				}
			}
		})
	}
	wg.Wait()
	for k := range 10 {
		if v, _ := s.Get(k); v != 800 {
			t.Fatalf("counter %d = %d; want 800", k, v)
		}
	}
	for i := range 8 {
		if v, ok := s.Get(100 + i); !ok || v != 0 {
			t.Fatalf("GetOrSet key %d = %d, %v; want first value 0", 100+i, v, ok)
		}
	}
}
