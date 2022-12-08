// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

package integration

import (
	"context"
	"testing"

	"github.com/zeebo/assert"
	"github.com/zeebo/errs"

	"storj.io/drpc/drpccache"
	"storj.io/drpc/drpctest"
)

func TestCache(t *testing.T) {
	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	// create a server that signals then waits for the context to die
	cli, close := createConnection(t, impl{
		Method1Fn: func(ctx context.Context, _ *In) (*Out, error) {
			cache := drpccache.FromContext(ctx)
			if cache == nil {
				return nil, errs.New("no cache associated with context")
			}
			cache.LoadOrCreate("value", func() interface{} { return 42 })
			return &Out{Out: 123}, nil
		},
		Method2Fn: func(stream DRPCService_Method2Stream) error {
			cache := drpccache.FromContext(stream.Context())
			if cache == nil {
				return errs.New("no cache associated with context")
			}
			value, _ := cache.Load("value").(int)
			return stream.SendAndClose(&Out{Out: int64(value)})
		},
	})
	defer close()

	{ // value not yet cached
		stream, err := cli.Method2(ctx)
		assert.NoError(t, err)
		out, err := stream.CloseAndRecv()
		assert.NoError(t, err)
		assert.True(t, Equal(out, &Out{Out: 0}))
	}

	{ // store value in the cache
		out, err := cli.Method1(ctx, in(1))
		assert.NoError(t, err)
		assert.True(t, Equal(out, &Out{Out: 123}))
	}

	{ // expected value in the cache
		stream, err := cli.Method2(ctx)
		assert.NoError(t, err)
		out, err := stream.CloseAndRecv()
		assert.NoError(t, err)
		assert.True(t, Equal(out, &Out{Out: 42}))
	}
}
