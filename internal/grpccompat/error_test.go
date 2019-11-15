// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package grpccompat

import (
	"context"
	"testing"

	"github.com/zeebo/assert"
	"github.com/zeebo/errs"
)

func TestError_ErrorPassedBack(t *testing.T) {
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
	})
}
