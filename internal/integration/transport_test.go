// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package integration

import (
	"context"
	"testing"

	"github.com/zeebo/assert"
	"storj.io/drpc/drpcconn"
	"storj.io/drpc/drpcctx"
)

func TestTransport_Error(t *testing.T) {
	// ensure that everything we launch eventually exits
	ctx := drpcctx.NewTracker(context.Background())
	defer ctx.Wait()
	defer ctx.Cancel()

	// create a channel to signal when the rpc has started
	started := make(chan struct{})

	// create a server that signals then waits for the context to die
	cli, close := createConnection(impl{
		Method1Fn: func(ctx context.Context, in *In) (*Out, error) {
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
	cli.DRPCConn().Transport().Close()
}

func TestTransport_Blocked(t *testing.T) {
	// ensure that everything we launch eventually exits
	ctx := drpcctx.NewTracker(context.Background())
	defer ctx.Wait()
	defer ctx.Cancel()

	// create a channel to hold the rpc error
	errch := make(chan error, 1)

	// create a transport that signals when reads/writes happen
	trs := new(transportSignaler)
	defer trs.Close()

	// start a client issuing an rpc that we keep track of
	cli := NewDRPCServiceClient(drpcconn.New(trs))
	ctx.Run(func(ctx context.Context) {
		_, err := cli.Method1(ctx, in(1))
		errch <- err
	})

	// wait for the write to happen before cancelling the context. this
	// should cause the rpc goroutine to exit.
	<-trs.write.Signal()
	ctx.Cancel()

	// we should always get a canceled error from issuing the rpc: not
	// the error returned by the transport due to a read/write.
	assert.Equal(t, <-errch, context.Canceled)
}
