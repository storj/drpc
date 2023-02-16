// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package integration

import (
	"context"
	"errors"
	"io"
	"runtime"
	"testing"
	"time"

	"github.com/zeebo/assert"

	"storj.io/drpc"
	"storj.io/drpc/drpcconn"
	"storj.io/drpc/drpcpool"
	"storj.io/drpc/drpctest"
)

func TestCancel(t *testing.T) {
	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	cli, close := createConnection(t, standardImpl)
	defer close()

	{ // ensure that we get canceled if issuing an rpc with an already canceled context
		ctx, cancel := context.WithCancel(ctx)
		cancel()

		out, err := cli.Method1(ctx, in(1))
		assert.Nil(t, out)
		assert.Equal(t, err, context.Canceled)
	}

	{ // ensure that if we cancel after rpc is done, transport stays valid
		{
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			out, err := cli.Method1(ctx, in(1))
			assert.NotNil(t, out)
			assert.NoError(t, err)

			cancel()
		}

		out, err := cli.Method1(ctx, in(1))
		assert.NotNil(t, out)
		assert.NoError(t, err)
	}

	{ // ensure that cancel in the middle of a stream eventually errors
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		stream, err := cli.Method2(ctx)
		assert.NoError(t, err)

		for i := 0; i < 10; i++ {
			assert.NoError(t, stream.Send(in(1)))
		}

		go cancel()

		for !errors.Is(stream.Send(in(1)), io.EOF) {
			runtime.Gosched()
		}
	}
}

func TestCancellationPropagation_Unitary(t *testing.T) {
	timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	called := make(chan struct{}, 1)
	cancelled := make(chan struct{}, 1)

	sleepy := impl{
		Method1Fn: func(ctx context.Context, _ *In) (*Out, error) {
			called <- struct{}{}
			select {
			case <-timeout.Done():
			case <-ctx.Done():
				cancelled <- struct{}{}
			}
			return &Out{Out: 1}, nil
		},
	}

	cli, close := createConnection(t, sleepy)
	defer close()

	clientctx := drpctest.NewTracker(t)
	defer clientctx.Close()

	clientctx.Run(func(ctx context.Context) {
		_, _ = cli.Method1(ctx, in(1))
	})

	<-called
	clientctx.Cancel()
	clientctx.Wait()

	select {
	case <-cancelled:
	case <-timeout.Done():
		t.Fatal("did not finish in time")
	}
}

func TestCancellationPropagation_Stream(t *testing.T) {
	timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	called := make(chan struct{}, 1)
	cancelled := make(chan struct{}, 1)

	sleepy := impl{
		Method4Fn: func(stream DRPCService_Method4Stream) error {
			called <- struct{}{}
			select {
			case <-stream.Context().Done():
				cancelled <- struct{}{}
			case <-timeout.Done():
				t.Error("server did not exit")
			}
			return nil
		},
	}

	cli, close := createConnection(t, sleepy)
	defer close()

	clientctx := drpctest.NewTracker(t)
	defer clientctx.Close()

	clientctx.Run(func(ctx context.Context) {
		stream, _ := cli.Method4(ctx)

		// this is a weird case where the rpc does not send or receive or even
		// close the stream, and neither does the other side, and so we have to
		// explicitly flush the invoke.
		type (
			getStreamer interface{ GetStream() drpc.Stream }
			rawFlusher  interface{ RawFlush() error }
		)
		_ = stream.(getStreamer).GetStream().(rawFlusher).RawFlush()

		called <- struct{}{}
		select {
		case <-stream.Context().Done():
		case <-timeout.Done():
			t.Error("client did not exit")
		}
	})

	// Ensuring both the client and the server have called is important
	// before canceling, otherwise there's a race due to the client
	// performing multiple operations to invoke, and the server can
	// send on called before the client returns the stream, causing
	// the client to return <nil>, canceled.
	<-called
	<-called
	clientctx.Cancel()
	clientctx.Wait()

	select {
	case <-cancelled:
	case <-timeout.Done():
		t.Fatal("did not finish in time")
	}
}

func TestCancelWhileWriteBlocked(t *testing.T) {
	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	tr := newTransportBlocker()
	defer func() { _ = tr.Close() }()

	conn := drpcconn.New(tr)
	cli := NewDRPCServiceClient(conn)

	done := make(chan struct{}, 1)
	timer := time.NewTimer(time.Second)
	defer timer.Stop()

	invokeCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	allowCancel := make(chan struct{})

	ctx.Run(func(_ context.Context) {
		<-allowCancel
		cancel()
	})

	ctx.Run(func(_ context.Context) {
		stream, _ := cli.Method2(invokeCtx)
		tr.BlockWrites()
		go close(allowCancel)
		_ = stream.Send(in(2))
		done <- struct{}{}
	})

	select {
	case <-done:
	case <-timer.C:
		var buf [1 << 20]byte
		t.Logf("%s", buf[:runtime.Stack(buf[:], true)])
		t.Fatal("timeout")
	}
}

func TestCancelRepeatedPooled(t *testing.T) {
	tctx := drpctest.NewTracker(t)
	defer tctx.Close()
	server := impl{
		Method2Fn: func(stream DRPCService_Method2Stream) error {
			var total int64
			for {
				in, err := stream.Recv()
				if errors.Is(err, io.EOF) {
					break
				} else if err != nil {
					return err
				}
				total += in.In
			}
			return stream.SendAndClose(out(total))
		},
	}
	conns := 0
	foo := func(ctx context.Context, p *drpcpool.Pool) {
		conn := p.Get(ctx, "foo", func(ctx context.Context, key interface{}) (drpcpool.Conn, error) {
			conns++
			return createRawConnection(t, server, tctx), nil
		})
		defer func() { _ = conn.Close() }()

		stream, err := NewDRPCServiceClient(conn).Method2(ctx)
		assert.NoError(t, err)
		assert.NoError(t, stream.Send(in(1)))
		assert.NoError(t, stream.Send(in(2)))
		assert.NoError(t, stream.Send(in(3)))
		out, err := stream.CloseAndRecv()
		assert.NoError(t, err)
		assert.Equal(t, out.Out, 6)

		<-stream.Context().Done()
	}
	p := drpcpool.New(drpcpool.Options{
		Capacity: 1,
	})
	for i := 0; i < 10000; i++ {
		ctx, cancel := context.WithCancel(tctx)
		foo(ctx, p)
		cancel()
	}
	assert.Equal(t, conns, 1)
}
