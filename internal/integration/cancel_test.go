// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package integration

import (
	"context"
	"io"
	"testing"

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
