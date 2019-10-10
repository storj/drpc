// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package integration

import (
	"context"
	"testing"

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
