// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package integration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"testing"

	"github.com/zeebo/assert"

	"storj.io/drpc/drpcconn"
	"storj.io/drpc/drpcerr"
	"storj.io/drpc/drpcmux"
	"storj.io/drpc/drpcserver"
	"storj.io/drpc/drpcstats"
	"storj.io/drpc/drpctest"
)

func TestSimple(t *testing.T) {
	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	cli, close := createConnection(t, standardImpl)
	defer close()

	{
		out, err := cli.Method1(ctx, &In{In: 1})
		assert.NoError(t, err)
		assert.True(t, Equal(out, &Out{Out: 1}))
	}

	{
		stream, err := cli.Method2(ctx)
		assert.NoError(t, err)
		assert.NoError(t, stream.Send(&In{In: 2}))
		assert.NoError(t, stream.Send(&In{In: 2}))
		out, err := stream.CloseAndRecv()
		assert.NoError(t, err)
		assert.True(t, Equal(out, &Out{Out: 2}))
	}

	{
		stream, err := cli.Method3(ctx, &In{In: 3})
		assert.NoError(t, err)
		for {
			out, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			assert.NoError(t, err)
			assert.True(t, Equal(out, &Out{Out: 3}))
		}
	}

	{
		stream, err := cli.Method4(ctx)
		assert.NoError(t, err)
		assert.NoError(t, stream.Send(&In{In: 4}))
		assert.NoError(t, stream.Send(&In{In: 4}))
		assert.NoError(t, stream.Send(&In{In: 4}))
		assert.NoError(t, stream.Send(&In{In: 4}))
		assert.NoError(t, stream.CloseSend())
		for {
			out, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			assert.NoError(t, err)
			assert.True(t, Equal(out, &Out{Out: 4}))
		}
	}

	{
		_, err := cli.Method1(ctx, &In{In: 5})
		assert.Error(t, err)
		assert.Equal(t, drpcerr.Code(err), 5)
	}
}

func TestConcurrent(t *testing.T) {
	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	cli, close := createConnection(t, standardImpl)
	defer close()

	const N = 1000
	errs := make(chan error)
	for i := 0; i < N; i++ {
		ctx.Run(func(ctx context.Context) {
			select {
			case <-ctx.Done():
			case errs <- func() error {
				out, err := cli.Method1(ctx, &In{In: 1})
				if err != nil {
					return err
				} else if out.Out != 1 {
					return fmt.Errorf("wrong result %d", out.Out)
				} else {
					return nil
				}
			}():
			}
		})
	}
	for i := 0; i < N; i++ {
		assert.NoError(t, <-errs)
	}
}

func TestServerStats(t *testing.T) {
	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	c1, c2 := net.Pipe()
	mux := drpcmux.New()
	_ = DRPCRegisterService(mux, standardImpl)

	srv := drpcserver.NewWithOptions(mux, drpcserver.Options{
		CollectStats: true,
	})
	ctx.Run(func(ctx context.Context) { _ = srv.ServeOne(ctx, c1) })

	conn := drpcconn.NewWithOptions(c2, drpcconn.Options{})
	defer func() { _ = conn.Close() }()
	cli := NewDRPCServiceClient(conn)

	assert.Equal(t, srv.Stats(), map[string]drpcstats.Stats{})

	_, err := cli.Method1(ctx, in(5))
	assert.Error(t, err)

	assert.Equal(t, srv.Stats(), map[string]drpcstats.Stats{
		"/service.Service/Method1": {Read: 2, Written: 12},
	})

	_, err = cli.Method1(ctx, in(1))
	assert.NoError(t, err)

	assert.Equal(t, srv.Stats(), map[string]drpcstats.Stats{
		"/service.Service/Method1": {Read: 2 + 2, Written: 12 + 2},
	})

	stream, err := cli.Method3(ctx, in(3))
	assert.NoError(t, err)
	for i := 0; i < 3; i++ {
		_, err := stream.Recv()
		assert.NoError(t, err)
	}
	_, err = stream.Recv()
	assert.That(t, errors.Is(err, io.EOF))
	assert.NoError(t, stream.Close())

	assert.Equal(t, srv.Stats(), map[string]drpcstats.Stats{
		"/service.Service/Method1": {Read: 2 + 2, Written: 12 + 2},
		"/service.Service/Method3": {Read: 2, Written: 6},
	})
}

func TestClientStats(t *testing.T) {
	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	c1, c2 := net.Pipe()
	mux := drpcmux.New()
	_ = DRPCRegisterService(mux, standardImpl)

	srv := drpcserver.New(mux)
	ctx.Run(func(ctx context.Context) { _ = srv.ServeOne(ctx, c1) })

	conn := drpcconn.NewWithOptions(c2, drpcconn.Options{
		CollectStats: true,
	})
	defer func() { _ = conn.Close() }()
	cli := NewDRPCServiceClient(conn)

	assert.Equal(t, srv.Stats(), map[string]drpcstats.Stats{})

	_, err := cli.Method1(ctx, in(5))
	assert.Error(t, err)

	assert.Equal(t, conn.Stats(), map[string]drpcstats.Stats{
		"/service.Service/Method1": {Read: 12, Written: 26},
	})

	_, err = cli.Method1(ctx, in(1))
	assert.NoError(t, err)

	assert.Equal(t, conn.Stats(), map[string]drpcstats.Stats{
		"/service.Service/Method1": {Read: 12 + 2, Written: 26 + 26},
	})

	stream, err := cli.Method3(ctx, in(3))
	assert.NoError(t, err)
	for i := 0; i < 3; i++ {
		_, err := stream.Recv()
		assert.NoError(t, err)
	}
	_, err = stream.Recv()
	assert.That(t, errors.Is(err, io.EOF))
	assert.NoError(t, stream.Close())

	assert.Equal(t, conn.Stats(), map[string]drpcstats.Stats{
		"/service.Service/Method1": {Read: 12 + 2, Written: 26 + 26},
		"/service.Service/Method3": {Read: 6, Written: 26},
	})
}
