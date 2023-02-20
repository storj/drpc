// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcstream

import (
	"bytes"
	"context"
	"errors"
	"io"
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
