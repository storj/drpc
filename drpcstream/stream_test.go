// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcstream

import (
	"bytes"
	"context"
	"errors"
	"io"
	"runtime"
	"testing"

	"github.com/zeebo/assert"
	"github.com/zeebo/errs"

	"storj.io/drpc"
	"storj.io/drpc/drpctest"
	"storj.io/drpc/drpcwire"
)

func TestStream_StateTransitions(t *testing.T) {
	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	any := errors.New("any sentinel error")

	handlePacket := func(st *Stream, kind drpcwire.Kind) error {
		return st.HandlePacket(drpcwire.Packet{Kind: kind})
	}

	checkErrs := func(t *testing.T, exp interface{}, got error) {
		t.Helper()

		if cl, ok := exp.(*errs.Class); ok {
			assert.That(t, cl.Has(got))
		} else {
			switch exp {
			case any:
				assert.Error(t, got)
			case nil:
				assert.NoError(t, got)
			default:
				assert.Equal(t, exp, got)
			}
		}
	}

	cases := []struct {
		Op   func(st *Stream) error
		Send interface{}
		Recv error
	}{
		{ // send close
			Op:   func(st *Stream) error { return st.Close() },
			Send: any,
			Recv: any,
		},

		{ // send error
			Op:   func(st *Stream) error { return st.SendError(errors.New("test")) },
			Send: io.EOF,
			Recv: any,
		},

		{ // send closesend
			Op:   func(st *Stream) error { return st.CloseSend() },
			Send: any,
			Recv: nil,
		},

		{ // recv cancel
			Op:   func(st *Stream) error { st.Cancel(context.Canceled); return nil },
			Send: io.EOF,
			Recv: context.Canceled,
		},

		{ // recv deadline
			Op:   func(st *Stream) error { st.Cancel(context.DeadlineExceeded); return nil },
			Send: io.EOF,
			Recv: context.DeadlineExceeded,
		},

		{ // recv close
			Op:   func(st *Stream) error { return handlePacket(st, drpcwire.KindClose) },
			Send: &drpc.ClosedError,
			Recv: io.EOF,
		},

		{ // recv error
			Op:   func(st *Stream) error { return handlePacket(st, drpcwire.KindError) },
			Send: io.EOF,
			Recv: any,
		},

		{ // recv closesend
			Op:   func(st *Stream) error { return handlePacket(st, drpcwire.KindCloseSend) },
			Send: nil,
			Recv: io.EOF,
		},
	}

	for _, test := range cases {
		st := New(ctx, 0, drpcwire.NewWriter(io.Discard, 0))
		assert.NoError(t, test.Op(st))

		checkErrs(t, test.Send, st.RawWrite(drpcwire.KindMessage, nil))

		if test.Recv == nil {
			ctx.Run(func(ctx context.Context) { _ = handlePacket(st, drpcwire.KindMessage) })
		}
		_, err := st.RawRecv()
		checkErrs(t, test.Recv, err)
	}
}

func TestStream_Unblocks(t *testing.T) {
	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	handlePacket := func(st *Stream, kind drpcwire.Kind) error {
		return st.HandlePacket(drpcwire.Packet{Kind: kind})
	}

	cases := []struct {
		Op func(st *Stream) error
	}{
		{ // send close
			Op: func(st *Stream) error { return st.Close() },
		},

		{ // send error
			Op: func(st *Stream) error { return st.SendError(errors.New("test")) },
		},

		{ // recv cancel
			Op: func(st *Stream) error { st.Cancel(context.Canceled); return nil },
		},

		{ // recv deadline
			Op: func(st *Stream) error { st.Cancel(context.DeadlineExceeded); return nil },
		},

		{ // recv close
			Op: func(st *Stream) error { return handlePacket(st, drpcwire.KindClose) },
		},

		{ // recv error
			Op: func(st *Stream) error { return handlePacket(st, drpcwire.KindError) },
		},

		{ // recv closesend
			Op: func(st *Stream) error { return handlePacket(st, drpcwire.KindCloseSend) },
		},
	}

	for _, test := range cases {
		st := New(ctx, 0, drpcwire.NewWriter(io.Discard, 0))

		ctx.Run(func(ctx context.Context) { _, _ = st.RawRecv() })
		assert.NoError(t, test.Op(st))
		ctx.Wait()
	}
}

func TestStream_ContextCancel(t *testing.T) {
	ctx := context.Background()
	st := New(ctx, 0, drpcwire.NewWriter(io.Discard, 0))

	child, cancel := context.WithCancel(st.Context())
	defer cancel()

	assert.NoError(t, st.Close())
	<-st.Context().Done()
	<-child.Done()
}

func TestStream_ConcurrentCloseCancel(t *testing.T) {
	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	pr, pw := io.Pipe()
	defer func() { _ = pr.Close() }()
	defer func() { _ = pw.Close() }()

	st := New(ctx, 0, drpcwire.NewWriter(pw, 0))

	// start the Close call
	errch := make(chan error, 1)
	go func() { errch <- st.Close() }()

	// wait for the close to begin writing
	_, err := pr.Read(make([]byte, 1))
	assert.NoError(t, err)

	// cancel the context and close the transport
	st.Cancel(context.Canceled)
	assert.NoError(t, pw.Close())

	// we should always receive the canceled error
	assert.That(t, errors.Is(<-errch, context.Canceled))
}

func TestStream_Control(t *testing.T) {
	st := New(context.Background(), 0, drpcwire.NewWriter(io.Discard, 0))

	// N.B. the stream will return nil on any HandlePacket calls after the
	// stream has been terminated for any reason, including if an invalid
	// packet has been sent. the order of these two assertions is important!

	// an invalid packet is not an error if the control bit is set
	assert.NoError(t, st.HandlePacket(drpcwire.Packet{Control: true}))

	// an invalid packet is an error if the control bit it not set
	assert.That(t, drpc.InternalError.Has(st.HandlePacket(drpcwire.Packet{})))
}

func TestStream_CorkUntilFirstRead(t *testing.T) {
	run := func() {
		ctx := drpctest.NewTracker(t)
		defer ctx.Close()

		var buf bytes.Buffer
		st := New(ctx, 0, drpcwire.NewWriter(&buf, 50))

		// concurrently read and write at the same time.
		// we should always see the write happen.

		errch := make(chan error, 3)
		ctx.Run(func(ctx context.Context) {
			errch <- st.MsgSend([]byte("write"), byteEncoding{})
		})
		ctx.Run(func(ctx context.Context) {
			_, err := st.RawRecv()
			errch <- err
		})
		ctx.Run(func(ctx context.Context) {
			errch <- st.HandlePacket(drpcwire.Packet{
				Data: []byte("read"),
				ID:   drpcwire.ID{Message: 1},
				Kind: drpcwire.KindMessage,
			})
		})

		assert.NoError(t, <-errch)
		assert.NoError(t, <-errch)
		assert.NoError(t, <-errch)

		assert.Equal(t, buf.String(), "\x05\x00\x01\x05write")
	}
	for i := 0; i < 100; i++ {
		run()
	}
}

type byteEncoding struct{}

func (byteEncoding) Marshal(msg drpc.Message) ([]byte, error) { return msg.([]byte), nil }
func (byteEncoding) Unmarshal(buf []byte, msg drpc.Message) error {
	*msg.(*[]byte) = append(*msg.(*[]byte), buf...)
	return nil
}

func TestStream_PacketBufferReuse(t *testing.T) {
	run := func() {
		ctx := drpctest.NewTracker(t)
		defer ctx.Close()
		defer ctx.Wait()

		buf := make([]byte, 20)
		st := New(ctx, 0, drpcwire.NewWriter(io.Discard, 0))

		ctx.Run(func(ctx context.Context) {
			for !st.IsTerminated() {
				err := st.HandlePacket(drpcwire.Packet{
					Data: buf,
					Kind: drpcwire.KindMessage,
				})
				if err != nil {
					return
				}
				for i := range buf {
					buf[i]++
				}
			}
		})

		ctx.Run(func(ctx context.Context) {
			for !st.IsTerminated() {
				_, err := st.RawRecv()
				if err != nil {
					return
				}
			}
		})

		ctx.Run(func(ctx context.Context) {
			st.Cancel(context.Canceled)
		})
	}

	for i := 0; i < 100; i++ {
		run()
	}
}

func TestStream_SendCancelBusyDuringBlockedClose(t *testing.T) {
	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	pr, pw := io.Pipe()
	defer func() { _ = pr.Close() }()
	defer func() { _ = pw.Close() }()

	st := New(ctx, 0, drpcwire.NewWriter(pw, 0))

	// launch a goroutine to close the stream
	ctx.Run(func(ctx context.Context) { _ = st.Close() })

	// read just 1 byte from the pipe to ensure that the Close has started
	_, err := pr.Read(make([]byte, 1))
	assert.NoError(t, err)
	assert.That(t, st.IsTerminated())

	// even though the stream is terminated, soft cancel should report that
	// the stream is still busy because the close is being sent.
	busy, err := st.SendCancel(context.Canceled)
	assert.NoError(t, err)
	assert.That(t, busy)
}

func TestStream_SendCancelWhileConcurrentWriteAndCloseBlocked(t *testing.T) {
	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	pr, pw := io.Pipe()
	defer func() { _ = pr.Close() }()
	defer func() { _ = pw.Close() }()

	st := New(ctx, 0, drpcwire.NewWriter(pw, 0))

	// launch a goroutine to send a message
	ctx.Run(func(ctx context.Context) { _ = st.MsgSend([]byte("write"), byteEncoding{}) })

	// read just 1 byte from the pipe to ensure that the send has started
	_, err := pr.Read(make([]byte, 1))
	assert.NoError(t, err)

	// launch a goroutine to Close the stream
	ctx.Run(func(ctx context.Context) { _ = st.Close() })

	// we need to wait a little bit for Close to have the mutex acquired
	for st.mu.Unlocked() {
	}

	// SendCancel should report busy and not block forever
	busy, err := st.SendCancel(context.Canceled)
	assert.NoError(t, err)
	assert.That(t, busy)
}

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) { return f(p) }

func TestStream_StopWrites(t *testing.T) {
	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	stop := errors.New("stop")

	t.Run("Write", func(t *testing.T) {
		var buf bytes.Buffer
		st := New(ctx, 0, drpcwire.NewWriter(&buf, 0))

		st.StopWrites(stop)
		assert.Equal(t, st.RawWrite(drpcwire.KindMessage, []byte("data")), stop)
		assert.Equal(t, buf.Len(), 0)
	})

	t.Run("Flush", func(t *testing.T) {
		var buf bytes.Buffer
		st := New(ctx, 0, drpcwire.NewWriter(&buf, 1024))

		assert.NoError(t, st.RawWrite(drpcwire.KindMessage, []byte("data")))
		st.StopWrites(stop)
		assert.Equal(t, st.RawFlush(), stop)
		assert.Equal(t, buf.Len(), 0)
	})

	// a write in progress stops before its next frame rather than finishing.
	t.Run("FrameBoundary", func(t *testing.T) {
		var st *Stream
		var writes int
		wr := drpcwire.NewWriter(writerFunc(func(p []byte) (int, error) {
			writes++
			st.StopWrites(stop)
			return len(p), nil
		}), 1)
		st = NewWithOptions(ctx, 0, wr, Options{SplitSize: 1})

		assert.Equal(t, st.RawWrite(drpcwire.KindMessage, []byte("data")), stop)
		assert.Equal(t, writes, 1)
	})

	// terminal messages must not tell the remote that a stream that was cut
	// short finished cleanly, and must leave the stream for a soft cancel.
	t.Run("Terminal", func(t *testing.T) {
		for name, op := range map[string]func(*Stream) error{
			"Close":     func(st *Stream) error { return st.Close() },
			"CloseSend": func(st *Stream) error { return st.CloseSend() },
			"SendError": func(st *Stream) error { return st.SendError(errors.New("test")) },
		} {
			t.Run(name, func(t *testing.T) {
				var buf bytes.Buffer
				st := New(ctx, 0, drpcwire.NewWriter(&buf, 0))

				// Close is cleanup, so it does not report the stop again.
				want := stop
				if name == "Close" {
					want = nil
				}

				st.StopWrites(stop)
				assert.Equal(t, op(st), want)
				assert.Equal(t, buf.Len(), 0)
				assert.That(t, !st.IsTerminated())

				busy, err := st.SendCancel(context.Canceled)
				assert.NoError(t, err)
				assert.That(t, !busy)
				assert.That(t, st.IsTerminated())
				assert.That(t, buf.Len() > 0)
			})
		}
	})

	// a nil error must not make stopped writes and closes report success.
	t.Run("Nil", func(t *testing.T) {
		var buf bytes.Buffer
		st := New(ctx, 0, drpcwire.NewWriter(&buf, 0))

		st.StopWrites(nil)
		assert.That(t, errors.Is(st.RawWrite(drpcwire.KindMessage, []byte("data")), context.Canceled))
		assert.That(t, errors.Is(st.CloseSend(), context.Canceled))
		assert.Equal(t, buf.Len(), 0)
	})

	// once the stream is canceled, sends fail like they would without a stop.
	t.Run("AfterCancel", func(t *testing.T) {
		st := New(ctx, 0, drpcwire.NewWriter(io.Discard, 0))

		st.StopWrites(stop)
		st.Cancel(context.Canceled)
		assert.Equal(t, st.RawWrite(drpcwire.KindMessage, []byte("data")), io.EOF)
	})
}

func TestStream_Invoked(t *testing.T) {
	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	// every frame is flushed on its own, so the writer sees each one.
	var st *Stream
	var during []bool
	wr := drpcwire.NewWriter(writerFunc(func(p []byte) (int, error) {
		during = append(during, st.Invoked())
		return len(p), nil
	}), 1)
	st = NewWithOptions(ctx, 0, wr, Options{SplitSize: 1})

	assert.NoError(t, st.RawWrite(drpcwire.KindInvokeMetadata, []byte("meta")))
	assert.That(t, !st.Invoked())

	assert.NoError(t, st.RawWrite(drpcwire.KindInvoke, []byte("rpc")))
	assert.That(t, st.Invoked())
	assert.That(t, len(during) > 2)
	for _, invoked := range during {
		assert.That(t, !invoked)
	}
}

func TestStream_TryCancel(t *testing.T) {
	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	t.Run("Free", func(t *testing.T) {
		st := New(ctx, 0, drpcwire.NewWriter(io.Discard, 0))

		finished, ok := st.TryCancel(context.Canceled)
		assert.That(t, ok)
		assert.That(t, !finished)
		assert.That(t, st.IsFinished())
	})

	// a Close waiting for a write blocked in the transport holds the mutex.
	// TryCancel must not wait for it, and the Close must report the cancel
	// once the write fails.
	t.Run("Held", func(t *testing.T) {
		pr, pw := io.Pipe()
		st := New(ctx, 0, drpcwire.NewWriter(pw, 0))

		ctx.Run(func(context.Context) { _ = st.MsgSend([]byte("write"), byteEncoding{}) })
		_, err := pr.Read(make([]byte, 1))
		assert.NoError(t, err)

		closeErr := make(chan error, 1)
		ctx.Run(func(context.Context) { closeErr <- st.Close() })
		for st.mu.Unlocked() {
			runtime.Gosched() // a bare spin can starve the Close on one cpu
		}

		finished, ok := st.TryCancel(context.Canceled)
		assert.That(t, !ok)
		assert.That(t, !finished)

		_ = pr.Close()
		assert.That(t, errors.Is(<-closeErr, context.Canceled))
		assert.That(t, st.Cancel(context.Canceled))
	})

	// a nil error must not let operations that fail after the cancel report
	// success.
	t.Run("Nil", func(t *testing.T) {
		pr, pw := io.Pipe()
		st := New(ctx, 0, drpcwire.NewWriter(pw, 0))

		ctx.Run(func(context.Context) { _ = st.MsgSend([]byte("write"), byteEncoding{}) })
		_, err := pr.Read(make([]byte, 1))
		assert.NoError(t, err)

		closeErr := make(chan error, 1)
		ctx.Run(func(context.Context) { closeErr <- st.Close() })
		for st.mu.Unlocked() {
			runtime.Gosched() // a bare spin can starve the Close on one cpu
		}

		_, ok := st.TryCancel(nil)
		assert.That(t, !ok)

		_ = pr.Close()
		assert.That(t, errors.Is(<-closeErr, context.Canceled))
	})
}
