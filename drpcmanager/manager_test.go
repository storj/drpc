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
	run := func(t *testing.T, grace time.Duration) string {
		ctx := drpctest.NewTracker(t)
		defer ctx.Close()

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
		defer func() { _ = man.Close() }()
		defer tr.setReadOpen(true)
		defer tr.setWriteOpen(true)

		subctx, cancel := context.WithCancel(ctx)
		defer cancel()

		stream, err := man.NewClientStream(subctx, "rpc")
		assert.NoError(t, err)
		defer func() { _ = stream.Close() }()

		// hold the stream's write mutex the way a concurrent send would, by
		// parking an in-flight flush inside the transport.
		ctx.Run(func(context.Context) {
			_ = stream.RawWrite(drpcwire.KindMessage, []byte("message"))
			_ = stream.RawFlush()
		})

		// wait until the flush is actually parked in the transport, so the cancel
		// below reliably observes the stream as busy.
		tr.waitWriting()

		// release the send only after the manager has had a chance to see it as
		// busy. Without a grace period the manager gives up before this fires and
		// reports busy; with one, the retry picks the stream up once it lands.
		time.AfterFunc(50*time.Millisecond, func() { tr.setWriteOpen(true) })

		cancel()

		select {
		case outcome := <-outcomes:
			return outcome
		case <-time.After(10 * time.Second):
			t.Fatal("timed out waiting for a cancel outcome")
			return ""
		}
	}

	t.Run("NoGrace", func(t *testing.T) {
		assert.Equal(t, run(t, 0), CancelBusy)
	})

	t.Run("Grace", func(t *testing.T) {
		assert.Equal(t, run(t, time.Second), CancelClean)
	})
}

type blockedTransport struct {
	mu *sync.Mutex
	co *sync.Cond
	ro bool
	wo bool
	rn int // number of reads currently blocked
	wn int // number of writes currently blocked
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
	b.mu.Lock()
	defer b.mu.Unlock()

	*n++
	b.co.Broadcast()
	defer func() { *n-- }()

	for !*rw {
		b.co.Wait()
	}
	return p, nil
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
