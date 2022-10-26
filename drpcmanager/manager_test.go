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
		stream, err := cman.NewClientStream(ctx)
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

				stream, err := man.NewClientStream(subctx)
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

type blockedTransport struct {
	mu *sync.Mutex
	co *sync.Cond
	ro bool
	wo bool
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

func (b *blockedTransport) wait(p int, rw *bool) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for !*rw {
		b.co.Wait()
	}
	return p, nil
}

func (b *blockedTransport) Read(p []byte) (n int, err error)  { return b.wait(len(p), &b.ro) }
func (b *blockedTransport) Write(p []byte) (n int, err error) { return b.wait(len(p), &b.wo) }
func (b *blockedTransport) Close() error                      { return nil }
