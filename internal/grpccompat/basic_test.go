// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package grpccompat

import (
	"context"
	"testing"

	"github.com/zeebo/assert"
	"google.golang.org/protobuf/proto"
)

func TestBasic(t *testing.T) {
	impl := &serviceImpl{
		Method1Fn: func(ctx context.Context, in *In) (*Out, error) {
			return asOut(in), nil
		},
	}

	testCompat(t, impl, func(t *testing.T, cli Client, ensure func(*Out, error)) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		in := &In{
			In:  5,
			Buf: []byte("foo"),
			Opt: proto.Int64(8),
		}

		out, err := cli.Method1(ctx, in)
		assert.NoError(t, err)
		assert.That(t, proto.Equal(out, asOut(in)))

		ensure(out, err)
	})
}
