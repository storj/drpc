// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package integration

import (
	"context"
	"strings"
	"testing"

	"github.com/zeebo/assert"

	"storj.io/drpc/drpcctx"
	"storj.io/drpc/drpcerr"
)

func TestError(t *testing.T) {
	ctx := drpcctx.NewTracker(context.Background())
	defer ctx.Wait()
	defer ctx.Cancel()

	cli, close := createConnection(standardImpl)
	defer close()

	for i := int64(2); i < 20; i++ {
		out, err := cli.Method1(ctx, in(i))
		assert.Nil(t, out)
		assert.Error(t, err)
		assert.Equal(t, drpcerr.Code(err), i)
	}
}

func TestError_Context(t *testing.T) {
	ctx := drpcctx.NewTracker(context.Background())
	defer ctx.Wait()
	defer ctx.Cancel()

	cli, close := createConnection(impl{
		Method1Fn: func(ctx context.Context, in *In) (*Out, error) {
			return nil, [...]error{
				context.Canceled,
				context.DeadlineExceeded,
			}[in.In%2]
		},
	})
	defer close()

	for i := int64(2); i < 20; i++ {
		out, err := cli.Method1(ctx, in(i))
		assert.Nil(t, out)
		assert.Error(t, err)
		assert.That(t, strings.Contains(err.Error(), "context"))
	}
}
