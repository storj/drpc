// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

package integration

import (
	"context"
	"testing"

	"github.com/zeebo/assert"
	"github.com/zeebo/errs"

	"storj.io/drpc/drpccache"
	"storj.io/drpc/drpcctx"
)

func TestCache(t *testing.T) {
	// ensure that everything we launch eventually exits
	ctx := drpcctx.NewTracker(context.Background())
	defer ctx.Wait()
	defer ctx.Cancel()

	// create a server that signals then waits for the context to die
	cli, close := createConnection(impl{
		Method1Fn: func(ctx context.Context, in *In) (*Out, error) {
			cache := drpccache.FromContext(ctx)
			if cache == nil {
				return nil, errs.New("cache is missing")
			}
			cache.LoadOrCreate("value", func() interface{} {
				return 42
			})
			return &Out{Out: 123}, nil
		},
		Method2Fn: func(stream DRPCService_Method2Stream) error {
			cache := drpccache.FromContext(stream.Context())
			if cache == nil {
				return errs.New("no cache associated with stream")
			}

			value := cache.Load("value")
			if value == nil {
				return errs.New("expected value to be cached")
			}
			return nil
		},
	})
	defer close()

	_, err := cli.Method1(ctx, in(1))
	assert.NoError(t, err)

	stream, err := cli.Method2(ctx)
	assert.NoError(t, err)
	assert.NoError(t, stream.Close())

	// kill the transport from underneath of it
	assert.NoError(t, cli.DRPCConn().Transport().Close())
}
