// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package integration

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/zeebo/assert"

	"storj.io/drpc/drpcconn"
	"storj.io/drpc/drpctest"
)

func TestTransport_Error(t *testing.T) {
	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	// create a channel to signal when the rpc has started
	started := make(chan struct{})

	// create a server that signals then waits for the context to die
	cli, close := createConnection(t, impl{
		Method1Fn: func(ctx context.Context, _ *In) (*Out, error) {
			started <- struct{}{}
			<-ctx.Done()
			return nil, nil
		},
	})
	defer close()

	// async start the client issuing the rpc
	ctx.Run(func(ctx context.Context) { _, _ = cli.Method1(ctx, in(1)) })

	// wait for it to be started
	<-started

	// kill the transport from underneath of it
	assert.NoError(t, cli.DRPCConn().(*drpcconn.Conn).Transport().Close())
}

func TestTransport_Blocked(t *testing.T) {
	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	// create a channel to hold the rpc error
	errch := make(chan error, 1)

	// create a transport that signals when reads/writes happen
	trs := new(transportSignaler)
	defer func() { assert.NoError(t, trs.Close()) }()

	// start a client issuing an rpc that we keep track of
	cli := NewDRPCServiceClient(drpcconn.New(trs))
	ctx.Run(func(ctx context.Context) {
		_, err := cli.Method1(ctx, in(1))
		errch <- err
	})

	// wait for the write to happen before canceling the context. this
	// should cause the rpc goroutine to exit.
	<-trs.write.Signal()
	ctx.Cancel()

	// we should always get a canceled error from issuing the rpc: not
	// the error returned by the transport due to a read/write.
	assert.Equal(t, <-errch, context.Canceled)
}

func TestTransport_ErrorCausesCancel(t *testing.T) {
	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	// create a channel to signal when the rpc has started
	started := make(chan struct{})
	serr := make(chan error, 1)
	cerr := make(chan error, 1)

	// create a server that signals then waits for the context to die
	cli, close := createConnection(t, impl{
		Method2Fn: func(stream DRPCService_Method2Stream) error {
			started <- struct{}{}
			serr <- stream.MsgRecv(nil, Encoding)
			return nil
		},
	})
	defer close()

	// async start the client issuing the rpc
	ctx.Run(func(ctx context.Context) {
		stream, _ := cli.Method2(ctx)
		started <- struct{}{}
		cerr <- stream.MsgRecv(nil, Encoding)
	})

	// wait for it to be started. it is important to wait for
	// both the client and server to be started, otherwise there's
	// a race due to the client performing multiple operations to
	// invoke, and the server can send on started before the client
	// returns the stream, causing the client to return <nil>, canceled.
	<-started
	<-started

	// kill the transport from underneath of it
	assert.NoError(t, cli.DRPCConn().(*drpcconn.Conn).Transport().Close())

	// the server should always be context.Canceled because it for sure sees
	// that the remote side closed the connection.
	{
		err := <-serr
		t.Log("server error:", err)
		assert.That(t, errors.Is(err, context.Canceled))
	}

	// net.Pipe has a nondeterministic select inside of the read call on the local
	// side and remote side being closed, and in some rare cases it will see the
	// remote side closed first, returning io.EOF instead of io.ErrClosedPipe, so
	// we have to check that as well.
	{
		err := <-cerr
		t.Log("client error:", err)
		assert.That(t, errors.Is(err, io.ErrClosedPipe) || errors.Is(err, context.Canceled))
	}
}
