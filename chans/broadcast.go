package chans

import (
	"context"
	"sync"
)

// Broadcaster is a pub/sub hub: every value published is delivered to every
// current subscriber, each through its own channel. It is safe for
// concurrent use.
//
// Delivery policy: Publish blocks until every subscriber has accepted the
// value (into its buffer or by a waiting receiver) or ctx is done, so a slow
// subscriber slows publishers down rather than losing values. TryPublish
// never blocks and drops the value for subscribers that are not ready.
//
// Values published by one goroutine are received by each subscriber in
// publish order; there is no ordering between concurrent publishers.
//
// The zero value is a usable Broadcaster with unbuffered subscriber
// channels. A Broadcaster must not be copied after first use.
type Broadcaster[T any] struct {
	mu     sync.Mutex
	subs   map[*subscriber[T]]struct{}
	buffer int
	closed bool
}

type subscriber[T any] struct {
	ch   chan T
	done chan struct{} // closed when unsubscribing; unblocks in-flight sends
	// mu is held for reading by each in-flight send on ch and for writing
	// while closing ch, so ch is never closed during a send. Publishers only
	// ever hold one subscriber's lock at a time, and a send blocked on ch is
	// released by done, so shutdown never waits on other subscribers.
	mu     sync.RWMutex
	closed bool
	once   sync.Once
}

// shutdown stops delivery to s and closes its channel. It is idempotent and
// returns promptly: a publisher blocked on s is released via s.done.
// s must already have been removed from the Broadcaster.
func (s *subscriber[T]) shutdown() {
	s.once.Do(func() {
		close(s.done)
		s.mu.Lock()
		s.closed = true
		close(s.ch)
		s.mu.Unlock()
	})
}

// send delivers v to s, blocking until s accepts it, s is shut down, or ctx
// is done.
func (s *subscriber[T]) send(ctx context.Context, v T) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return nil
	}
	select {
	case s.ch <- v:
	case <-s.done:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

// trySend delivers v to s if it can do so without blocking and reports
// whether it did.
func (s *subscriber[T]) trySend(v T) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return false
	}
	select {
	case s.ch <- v:
		return true
	default:
		return false
	}
}

// NewBroadcaster returns a Broadcaster whose subscriber channels have the
// given buffer size (buffer < 0 is treated as 0). A buffer lets subscribers
// fall behind by that many values before Publish blocks or TryPublish drops.
func NewBroadcaster[T any](buffer int) *Broadcaster[T] {
	return &Broadcaster[T]{buffer: max(buffer, 0)}
}

// Subscribe registers a new subscriber and returns its channel, which
// receives every value published from now on, and an unsubscribe function.
// Calling unsubscribe stops delivery and closes the channel (values still
// buffered can be drained); it is idempotent and safe to call concurrently
// with Publish and Close. If the Broadcaster is already closed, the returned
// channel is closed.
func (b *Broadcaster[T]) Subscribe() (ch <-chan T, unsubscribe func()) {
	s := &subscriber[T]{
		ch:   make(chan T, b.buffer),
		done: make(chan struct{}),
	}
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		s.shutdown()
		return s.ch, func() {}
	}
	if b.subs == nil {
		b.subs = make(map[*subscriber[T]]struct{})
	}
	b.subs[s] = struct{}{}
	b.mu.Unlock()

	return s.ch, func() {
		b.mu.Lock()
		delete(b.subs, s)
		b.mu.Unlock()
		s.shutdown()
	}
}

// Publish delivers v to every current subscriber, blocking until each has
// accepted it or ctx is done. The subscribers are those registered when Publish
// is called: ones that subscribe meanwhile do not receive v, and ones that
// unsubscribe meanwhile are skipped. It returns ctx.Err() if ctx is done first,
// in which case some subscribers may have received v and others not, and
// ErrClosed if the Broadcaster is closed.
func (b *Broadcaster[T]) Publish(ctx context.Context, v T) error {
	subs, err := b.snapshot()
	if err != nil {
		return err
	}
	for _, s := range subs {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := s.send(ctx, v); err != nil {
			return err
		}
	}
	return nil
}

// TryPublish delivers v to every subscriber that can accept it without
// blocking and drops it for the rest. It returns the number of subscribers
// that received v (0 if the Broadcaster is closed).
func (b *Broadcaster[T]) TryPublish(v T) int {
	subs, err := b.snapshot()
	if err != nil {
		return 0
	}
	n := 0
	for _, s := range subs {
		if s.trySend(v) {
			n++
		}
	}
	return n
}

// Close unsubscribes every subscriber, closing their channels, and makes
// subsequent Publish calls fail with ErrClosed. Publish calls blocked on a
// subscriber return promptly. Close is idempotent.
func (b *Broadcaster[T]) Close() {
	b.mu.Lock()
	b.closed = true
	subs := b.subs
	b.subs = nil
	b.mu.Unlock()
	for s := range subs {
		s.shutdown()
	}
}

// snapshot returns the current subscribers, or ErrClosed if b is closed.
func (b *Broadcaster[T]) snapshot() ([]*subscriber[T], error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil, ErrClosed
	}
	subs := make([]*subscriber[T], 0, len(b.subs))
	for s := range b.subs {
		subs = append(subs, s)
	}
	return subs, nil
}
