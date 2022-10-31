// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcstream

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/zeebo/assert"
	"github.com/zeebo/errs"

	"storj.io/drpc"
	"storj.io/drpc/drpcctx"
	"storj.io/drpc/drpcwire"
)

func TestStream_StateTransitions(t *testing.T) {
	ctx := drpcctx.NewTracker(context.Background())
	defer ctx.Wait()
	defer ctx.Cancel()

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
	ctx := drpcctx.NewTracker(context.Background())
	defer ctx.Wait()
	defer ctx.Cancel()

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
