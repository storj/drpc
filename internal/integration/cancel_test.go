// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package integration

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/zeebo/assert"

	"storj.io/drpc/drpcctx"
)

func TestCancel(t *testing.T) {
	ctx := drpcctx.NewTracker(context.Background())
	defer ctx.Wait()
	defer ctx.Cancel()

	cli, close := createConnection(standardImpl)
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

		for stream.Send(in(1)) != io.EOF {
		}
	}
}

func TestCancellationPropagation_Unitary(t *testing.T) {
	timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ctx := drpcctx.NewTracker(context.Background())
	defer ctx.Wait()
	defer ctx.Cancel()

	called := make(chan struct{}, 1)
	cancelled := make(chan struct{}, 1)

	sleepy := impl{
		Method1Fn: func(ctx context.Context, in *In) (*Out, error) {
			called <- struct{}{}
			select {
			case <-timeout.Done():
			case <-ctx.Done():
				cancelled <- struct{}{}
			}
			return &Out{Out: 1}, nil
		},
	}

	cli, close := createConnection(sleepy)
	defer close()

	clientctx := drpcctx.NewTracker(context.Background())
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

	ctx := drpcctx.NewTracker(context.Background())
	defer ctx.Wait()
	defer ctx.Cancel()

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

	cli, close := createConnection(sleepy)
	defer close()

	clientctx := drpcctx.NewTracker(context.Background())
	clientctx.Run(func(ctx context.Context) {
		stream, _ := cli.Method4(ctx)
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
