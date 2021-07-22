// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package grpccompat

import (
	"context"
	"testing"

	"github.com/zeebo/assert"
	"github.com/zeebo/errs"
)

func TestWebCompat_Simple(t *testing.T) {
	impl := &serviceImpl{
		Method1Fn: func(ctx context.Context, in *In) (*Out, error) {
			if in.In == 5 {
				return nil, errs.New("marker")
			}
			return out(in.In), nil
		},
	}

	testWebCompat(t, impl, func(t *testing.T, cli Client, ensure func(*Out, error)) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ensure(cli.Method1(ctx, in(5)))
		ensure(cli.Method1(ctx, in(4)))
		ensure(cli.Method1(ctx, in(0)))
	})
}

func TestWebCompat_Streaming(t *testing.T) {
	impl := &serviceImpl{
		Method3Fn: func(in *In, stream ServerMethod3Stream) error {
			if in.In == 1 {
				return errs.New("marker")
			}
			for i := int64(0); i < 5; i++ {
				if err := stream.Send(out(i)); err != nil {
					return err
				}
			}
			if in.In == 2 {
				return errs.New("marker")
			}
			return nil
		},
	}

	for i := int64(0); i < 3; i++ {
		testWebCompat(t, impl, func(t *testing.T, cli Client, ensure func(*Out, error)) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			stream, err := cli.Method3(ctx, in(i))
			assert.NoError(t, err)

			for i := 0; i < 6; i++ {
				ensure(stream.Recv())
			}
		})
	}

}
