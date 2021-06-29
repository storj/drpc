// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package backcompat

import (
	"context"
	"errors"
	"io"
	"net"

	"github.com/zeebo/errs"

	"storj.io/drpc/drpcconn"
	"storj.io/drpc/internal/backcompat/servicedefs"
)

func runClient(ctx context.Context, addr string) error {
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return errs.Wrap(err)
	}
	defer func() { _ = conn.Close() }()

	cli := servicedefs.NewDRPCServiceClient(drpcconn.New(conn))

	{ // check method 1
		out, err := cli.Method1(ctx, &servicedefs.In{In: 10})
		if err != nil {
			return errs.Wrap(err)
		} else if out.Out != 10 {
			return errs.New("invalid out value")
		}
	}

	{ // check method 2
		stream, err := cli.Method2(ctx)
		if err != nil {
			return errs.Wrap(err)
		}
		for i := 0; i < 5; i++ {
			if err := stream.Send(&servicedefs.In{In: 0}); err != nil {
				return errs.Wrap(err)
			}
		}
		out, err := stream.CloseAndRecv()
		if err != nil {
			return errs.Wrap(err)
		} else if out.Out != 5 {
			return errs.New("invalid out value")
		}
	}

	{ // check method 3
		stream, err := cli.Method3(ctx, &servicedefs.In{In: 7})
		if err != nil {
			return errs.Wrap(err)
		}
		for i := 0; i < 7; i++ {
			_, err := stream.Recv()
			if err != nil {
				return errs.Wrap(err)
			}
		}
		_, err = stream.Recv()
		if !errors.Is(err, io.EOF) {
			return errs.New("invalid last receive (method3): %w", err)
		}
	}

	{ // check method 4
		stream, err := cli.Method4(ctx)
		if err != nil {
			return errs.Wrap(err)
		}
		for i := 0; i < 15; i++ {
			if err := stream.Send(&servicedefs.In{In: 0}); err != nil {
				return errs.Wrap(err)
			}
		}
		if err := stream.CloseSend(); err != nil {
			return errs.Wrap(err)
		}
		for i := 0; i < 15; i++ {
			_, err := stream.Recv()
			if err != nil {
				return errs.Wrap(err)
			}
		}
		_, err = stream.Recv()
		if !errors.Is(err, io.EOF) {
			return errs.New("invalid last receive (method4): %w", err)
		}
	}

	return nil
}
