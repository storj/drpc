// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package integration

import (
	"bytes"
	"errors"
	"io"
	"math/rand"
	"testing"

	"github.com/zeebo/assert"
	"github.com/zeebo/errs"

	"storj.io/drpc/drpctest"
)

func TestLarge(t *testing.T) {
	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	//
	// define some test helpers
	//

	type any = interface{}

	receiver := func(recv func() (any, error), valid func(any) bool) error {
		var got []any
		for {
			in, err := recv()
			if errors.Is(err, io.EOF) {
				break
			} else if err != nil {
				return err
			}
			got = append(got, in)
		}
		for _, v := range got {
			if !valid(v) {
				return errs.New("invalid data")
			}
		}
		return nil
	}

	sender := func(send func(n int64) error) error {
		for i := 0; i < 100; i++ {
			if err := send(int64(rand.Intn(4 << 15))); err != nil {
				return err
			}
		}
		return nil
	}

	run := func(send func(n int64) error, recv func() (any, error), valid func(any) bool, close func() error) error {
		ch := make(chan error, 2)
		go func() { ch <- receiver(recv, valid) }()
		go func() { ch <- errs.Combine(sender(send), close()) }()
		return errs.Combine(<-ch, <-ch)
	}

	//
	// execute the actual test
	//

	cli, close := createConnection(t, &impl{
		Method4Fn: func(stream DRPCService_Method4Stream) error {
			return run(
				func(n int64) error { return stream.Send(&Out{Out: n, Data: data(n)}) },
				func() (any, error) { return stream.Recv() },
				func(in any) bool { return bytes.Equal(in.(*In).Data, data(in.(*In).In)) },
				stream.CloseSend,
			)
		},
	})
	defer close()

	stream, err := cli.Method4(ctx)
	assert.NoError(t, err)
	defer func() { assert.NoError(t, stream.Close()) }()

	assert.NoError(t, run(
		func(n int64) error { return stream.Send(&In{In: n, Data: data(n)}) },
		func() (any, error) { return stream.Recv() },
		func(out any) bool { return bytes.Equal(out.(*Out).Data, data(out.(*Out).Out)) },
		stream.CloseSend,
	))
}
