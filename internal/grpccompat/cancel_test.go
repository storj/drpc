// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package grpccompat

import (
	"context"
	"testing"

	"github.com/zeebo/assert"
	"github.com/zeebo/errs"
)

func TestCancel_ErrorAfterCancel(t *testing.T) {
	impl := &serviceImpl{
		Method4Fn: func(stream ServerMethod4Stream) error {
			<-stream.Context().Done()
			return errs.New("marker")
		},
	}

	testCompat(t, impl, func(t *testing.T, cli Client, ensure func(*Out, error)) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		stream, err := cli.Method4(ctx)
		assert.NoError(t, err)

		cancel()
		ensure(stream.Recv())
		ensure(nil, stream.Send(in(0)))
	})
}

func TestCancel_CancelAfterError(t *testing.T) {
	impl := &serviceImpl{
		Method4Fn: func(stream ServerMethod4Stream) error {
			return errs.New("marker")
		},
	}

	testCompat(t, impl, func(t *testing.T, cli Client, ensure func(*Out, error)) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		stream, err := cli.Method4(ctx)
		assert.NoError(t, err)

		ensure(stream.Recv())
		ensure(nil, stream.Send(in(0)))
		cancel()
		ensure(stream.Recv())
		ensure(nil, stream.Send(in(1)))
	})
}

func TestCancel_CancelAfterSuccess(t *testing.T) {
	impl := &serviceImpl{
		Method4Fn: func(stream ServerMethod4Stream) error {
			_ = stream.Send(out(2))
			_, _ = stream.Recv()
			<-stream.Context().Done()
			return nil
		},
	}

	testCompat(t, impl, func(t *testing.T, cli Client, ensure func(*Out, error)) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		stream, err := cli.Method4(ctx)
		assert.NoError(t, err)

		ensure(stream.Recv())
		ensure(nil, stream.Send(in(0)))
		cancel()
		ensure(stream.Recv())
		ensure(nil, stream.Send(in(1)))
	})
}
