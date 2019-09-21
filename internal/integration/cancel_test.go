// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/zeebo/assert"

	"storj.io/drpc/drpcctx"
)

func TestCancel(t *testing.T) {
	ctx := drpcctx.NewTracker(context.Background())
	defer ctx.Wait()
	defer ctx.Cancel()

	cli, close := createConnection(ctx)
	defer close()

	{
		ctx, cancel := context.WithCancel(ctx)
		cancel()

		out, err := cli.Method1(ctx, in(1))
		assert.Nil(t, out)
		assert.That(t, errors.Is(err, context.Canceled))
	}

	{
		out, err := cli.Method1(ctx, in(1))
		assert.NotNil(t, out)
		assert.NoError(t, err)
	}

	{
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		stream, err := cli.Method2(ctx)
		assert.NoError(t, err)

		// we can't check any errors here because the cancel notification
		// is async, and so it's possible that it never gets scheduled.
		for i := 0; i < 100; i++ {
			if i == 50 {
				cancel()
			}
			_ = stream.Send(in(1))
		}
		_, _ = stream.CloseAndRecv()
	}

	{
		out, err := cli.Method1(ctx, in(1))
		assert.NotNil(t, out)
		assert.NoError(t, err)
	}
}
