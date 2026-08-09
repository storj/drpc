// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcmanager

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/zeebo/assert"

	"storj.io/drpc/drpctest"
	"storj.io/drpc/drpcwire"
	"storj.io/drpc/internal/drpcopts"
)

func closed(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

func TestTimeout(t *testing.T) {
	tr := make(blockingTransport)
	man := NewWithOptions(tr, Options{
		InactivityTimeout: time.Millisecond,
	})
	defer func() { _ = man.Close() }()

	_, _, err := man.NewServerStream(context.Background())
	assert.That(t, errors.Is(err, context.DeadlineExceeded))
}

type blockingTransport chan struct{}

func (b blockingTransport) Read(p []byte) (n int, err error)  { <-b; return 0, io.EOF }
func (b blockingTransport) Write(p []byte) (n int, err error) { <-b; return 0, io.EOF }
func (b blockingTransport) Close() error                      { close(b); return nil }

func TestUnblocked_NoCancel(t *testing.T) {
	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	cconn, sconn := net.Pipe()
	defer func() { _ = cconn.Close() }()
	defer func() { _ = sconn.Close() }()

	cman := New(cconn)
	defer func() { _ = cman.Close() }()

	sman := New(sconn)
	defer func() { _ = sman.Close() }()

	ctx.Run(func(ctx context.Context) {
		stream, err := cman.NewClientStream(ctx, "rpc")
		assert.NoError(t, err)
		defer func() { _ = stream.Close() }()

		assert.NoError(t, stream.RawWrite(drpcwire.KindInvoke, []byte("invoke")))
		assert.NoError(t, stream.RawWrite(drpcwire.KindMessage, []byte("message")))
		assert.NoError(t, stream.RawFlush())
		assert.That(t, !closed(cman.Unblocked()))

		assert.NoError(t, stream.Close())
		assert.That(t, closed(cman.Unblocked()))
	})

	ctx.Run(func(ctx context.Context) {
		stream, _, err := sman.NewServerStream(ctx)
		assert.NoError(t, err)
		defer func() { _ = stream.Close() }()

		_, err = stream.RawRecv()
		assert.NoError(t, err)

		_, err = stream.RawRecv()
		assert.That(t, errors.Is(err, io.EOF))
	})

	ctx.Wait()
}

func TestUnblocked_SoftCancel(t *testing.T) {
	run := func(t *testing.T, softCancel bool) {
		ctx := drpctest.NewTracker(t)
		defer ctx.Close()

		tr := newBlockedTransport()
		man := NewWithOptions(tr, Options{SoftCancel: softCancel})
		defer func() { _ = man.Close() }()
		defer tr.setReadOpen(true)
		defer tr.setWriteOpen(true)

		for i := 0; i < 10; i++ {
			func() {
				subctx, cancel := context.WithCancel(ctx)
				defer cancel()

				stream, err := man.NewClientStream(subctx, "rpc")
				if softCancel {
					assert.NoError(t, err)
				} else if i > 0 {
					assert.Error(t, err)
					return
				}
				defer func() { _ = stream.Close() }()

				assert.That(t, !closed(man.Unblocked()))
				cancel()

				// temporary unblock writing to allow the stream to finish soft cancel
				tr.setWriteOpen(true)
				<-man.Unblocked()
				tr.setWriteOpen(false)
			}()
		}
	}

	t.Run("Enabled", func(t *testing.T) { run(t, true) })
	t.Run("Disabled", func(t *testing.T) { run(t, false) })
}

// TestSoftCancel_Grace covers the case that a stream is sending when its context
// is canceled. Without a grace period the transport is destroyed; with one, the
// send is given a bounded chance to finish so the cancel can be delivered and the
// transport kept.
func TestSoftCancel_Grace(t *testing.T) {
	// run parks a send in the transport, cancels the stream, calls release
	// once the cancel has observed the stream as busy, and returns the
	// reported outcome and the manager.
	run := func(t *testing.T, ctx *drpctest.Tracker, grace time.Duration, release func(*blockedTransport)) (string, *Manager) {
		outcomes := make(chan string, 1)

		tr := newBlockedTransport()
		opts := Options{SoftCancel: true, SoftCancelGrace: grace}
		drpcopts.SetManagerCancelCB(&opts.Internal, func(o string) {
			select {
			case outcomes <- o:
			default:
			}
		})

		man := NewWithOptions(tr, opts)
		t.Cleanup(func() { _ = man.Close() })
		t.Cleanup(func() { tr.setReadOpen(true) })

		subctx, cancel := context.WithCancel(ctx)
		t.Cleanup(cancel)

		stream, err := man.NewClientStream(subctx, "rpc")
		assert.NoError(t, err)
		t.Cleanup(func() { _ = stream.Close() })

		// hold the stream's write mutex the way a concurrent send would, by
		// parking an in-flight flush inside the transport. it is not tracked,
		// because release may leave it parked until cleanup reopens writes.
		go func() {
			_ = stream.RawWrite(drpcwire.KindMessage, []byte("message"))
			_ = stream.RawFlush()
		}()

		// wait until the flush is actually parked in the transport, so the cancel
		// below reliably observes the stream as busy.
		tr.waitWriting()

		// cleanups run last first, so this unparks the send before the stream
		// Close cleanup waits on it.
		t.Cleanup(func() { tr.setWriteOpen(true) })

		// give the manager a chance to see the send as busy before releasing
		// it. the outcome must not depend on which happens first.
		time.AfterFunc(50*time.Millisecond, func() { release(tr) })

		cancel()

		select {
		case outcome := <-outcomes:
			return outcome, man
		case <-time.After(10 * time.Second):
			t.Fatal("timed out waiting for a cancel outcome")
			return "", nil
		}
	}

	openWrites := func(tr *blockedTransport) { tr.setWriteOpen(true) }

	t.Run("NoGrace", func(t *testing.T) {
		ctx := drpctest.NewTracker(t)
		defer ctx.Close()

		// the send stays parked until cleanup, so the cancel is always busy.
		outcome, man := run(t, ctx, 0, func(*blockedTransport) {})
		assert.Equal(t, outcome, cancelBusy)

		select {
		case <-man.Closed():
		case <-time.After(5 * time.Second):
			t.Fatal("transport not closed after busy cancel")
		}
	})

	t.Run("Grace", func(t *testing.T) {
		ctx := drpctest.NewTracker(t)
		defer ctx.Close()

		outcome, man := run(t, ctx, time.Second, openWrites)
		assert.Equal(t, outcome, cancelClean)
		assert.That(t, !closed(man.Closed()))
	})

	// the original send finishes within the grace period, but the cancel
	// write that follows stalls. the grace period must still bound it.
	t.Run("StalledCancelWrite", func(t *testing.T) {
		ctx := drpctest.NewTracker(t)
		defer ctx.Close()

		start := time.Now()
		outcome, man := run(t, ctx, 200*time.Millisecond, func(tr *blockedTransport) { tr.allowWrites(1) })
		assert.Equal(t, outcome, cancelBusy)
		assert.That(t, time.Since(start) < 5*time.Second)

		select {
		case <-man.Closed():
		case <-time.After(5 * time.Second):
			t.Fatal("transport not closed after grace elapsed")
		}
	})
}

// TestSoftCancel_GraceStalledFirstCancelWrite checks that the grace period
// bounds the cancel write even when the stream is not busy, so the first
// attempt goes straight to a stalled transport.
func TestSoftCancel_GraceStalledFirstCancelWrite(t *testing.T) {
	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	tr := newBlockedTransport()
	man := NewWithOptions(tr, Options{SoftCancel: true, SoftCancelGrace: 100 * time.Millisecond})
	defer func() { _ = man.Close() }()
	defer tr.setReadOpen(true)
	defer tr.setWriteOpen(true)

	subctx, cancel := context.WithCancel(ctx)
	defer cancel()

	stream, err := man.NewClientStream(subctx, "rpc")
	assert.NoError(t, err)
	assert.NoError(t, stream.RawWrite(drpcwire.KindInvoke, []byte("rpc")))
	cancel()

	select {
	case <-man.Closed():
	case <-time.After(5 * time.Second):
		t.Fatal("stalled cancel write outlived the grace period")
	}
}

// TestSoftCancel_GraceTerminated checks that a manager closed during the grace
// period stops waiting for it.
func TestSoftCancel_GraceTerminated(t *testing.T) {
	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	outcomes := make(chan string, 1)

	tr := newBlockedTransport()
	opts := Options{SoftCancel: true, SoftCancelGrace: time.Hour}
	drpcopts.SetManagerCancelCB(&opts.Internal, func(o string) { outcomes <- o })

	man := NewWithOptions(tr, opts)
	defer func() { _ = man.Close() }()
	defer tr.setReadOpen(true)
	defer tr.setWriteOpen(true)

	subctx, cancel := context.WithCancel(ctx)
	defer cancel()

	stream, err := man.NewClientStream(subctx, "rpc")
	assert.NoError(t, err)
	assert.NoError(t, stream.RawWrite(drpcwire.KindInvoke, []byte("rpc")))

	// park a send so that the cancel is busy and waits out the grace period.
	ctx.Run(func(context.Context) { _ = stream.RawFlush() })
	tr.waitWriting()
	cancel()

	// the semaphore is released just before the grace wait starts.
	for man.sem.Full() {
		time.Sleep(time.Millisecond)
	}

	// Close blocks until the parked send is released by the deferred cleanup.
	go func() { _ = man.Close() }()

	select {
	case o := <-outcomes:
		assert.Equal(t, o, cancelBusy)
	case <-time.After(5 * time.Second):
		t.Fatal("grace wait outlived the manager")
	}
}

// TestSoftCancel_GraceBackToBackSends checks that a caller that keeps sending
// after the cancel cannot keep the stream busy for the whole grace period: new
// sends fail so only the one in flight has to finish.
func TestSoftCancel_GraceBackToBackSends(t *testing.T) {
	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	outcomes := make(chan string, 1)

	// each send holds the write lock for a while, as a real one would, so
	// there is little room between back to back sends.
	tr := newBlockedTransport()
	tr.wd = time.Millisecond
	opts := Options{SoftCancel: true, SoftCancelGrace: 5 * time.Second}
	drpcopts.SetManagerCancelCB(&opts.Internal, func(o string) { outcomes <- o })

	man := NewWithOptions(tr, opts)
	defer func() { _ = man.Close() }()
	defer tr.setReadOpen(true)

	subctx, cancel := context.WithCancel(ctx)
	defer cancel()

	stream, err := man.NewClientStream(subctx, "rpc")
	assert.NoError(t, err)

	sendErr := make(chan error, 1)
	ctx.Run(func(context.Context) {
		for {
			if err := stream.RawWrite(drpcwire.KindMessage, []byte("message")); err != nil {
				sendErr <- err
				return
			}
			if err := stream.RawFlush(); err != nil {
				sendErr <- err
				return
			}
		}
	})

	tr.waitWriting()
	cancel()
	time.AfterFunc(50*time.Millisecond, func() { tr.setWriteOpen(true) })

	start := time.Now()
	assert.Equal(t, <-outcomes, cancelClean)
	assert.That(t, time.Since(start) < time.Second)
	assert.That(t, !closed(man.Closed()))
	assert.Error(t, <-sendErr)
}

type blockedTransport struct {
	mu *sync.Mutex
	co *sync.Cond
	ro bool
	wo bool
	rn int // number of reads currently blocked
	wn int // number of writes currently blocked
	wa int // number of writes allowed through while writes are closed

	wd time.Duration // how long each write takes once let through
}

func newBlockedTransport() *blockedTransport {
	mu := new(sync.Mutex)
	co := sync.NewCond(mu)
	return &blockedTransport{
		mu: mu,
		co: co,
	}
}

func (b *blockedTransport) setWriteOpen(open bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.wo = open
	b.co.Broadcast()
}

func (b *blockedTransport) setReadOpen(open bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.ro = open
	b.co.Broadcast()
}

func (b *blockedTransport) wait(p int, rw *bool, n *int) (int, error) {
	if n == &b.wn && b.wd > 0 {
		defer time.Sleep(b.wd) // runs after the mutex is released
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	*n++
	b.co.Broadcast()
	defer func() { *n-- }()

	for !*rw {
		if n == &b.wn && b.wa > 0 {
			b.wa--
			break
		}
		b.co.Wait()
	}
	return p, nil
}

// allowWrites lets n blocked or future writes through while writes are closed.
func (b *blockedTransport) allowWrites(n int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.wa += n
	b.co.Broadcast()
}

// waitWriting blocks until at least one Write is parked in the transport.
func (b *blockedTransport) waitWriting() {
	b.mu.Lock()
	defer b.mu.Unlock()

	for b.wn == 0 {
		b.co.Wait()
	}
}

func (b *blockedTransport) Read(p []byte) (n int, err error)  { return b.wait(len(p), &b.ro, &b.rn) }
func (b *blockedTransport) Write(p []byte) (n int, err error) { return b.wait(len(p), &b.wo, &b.wn) }
func (b *blockedTransport) Close() error                      { return nil }
