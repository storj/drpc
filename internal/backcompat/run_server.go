// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package backcompat

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/zeebo/errs"

	"storj.io/drpc/drpcmux"
	"storj.io/drpc/drpcserver"
	"storj.io/drpc/internal/backcompat/servicedefs"
)

func runServer(ctx context.Context, addr string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return errs.Wrap(err)
	}
	defer func() { _ = lis.Close() }()

	fmt.Println(lis.Addr())

	conn, err := lis.Accept()
	if err != nil {
		return errs.Wrap(err)
	}
	defer func() { _ = conn.Close() }()

	mux := drpcmux.New()
	_ = servicedefs.DRPCRegisterService(mux, server{})
	_ = drpcserver.New(mux).ServeOne(ctx, conn)
	return nil
}

type server struct{}

func (server) Method1(ctx context.Context, in *servicedefs.In) (*servicedefs.Out, error) {
	return &servicedefs.Out{Out: in.In}, nil
}

func (server) Method2(stream servicedefs.DRPCService_Method2Stream) error {
	var i int64
	for {
		_, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return errs.Wrap(err)
		}
		i++
	}
	return errs.Wrap(stream.SendAndClose(&servicedefs.Out{Out: i}))
}

func (server) Method3(in *servicedefs.In, stream servicedefs.DRPCService_Method3Stream) error {
	for ; in.In > 0; in.In-- {
		if err := stream.Send(&servicedefs.Out{Out: in.In}); err != nil {
			return errs.Wrap(err)
		}
	}
	return nil
}

func (server) Method4(stream servicedefs.DRPCService_Method4Stream) error {
	var i int64
	for {
		_, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return errs.Wrap(err)
		}
		i++
	}
	for ; i > 0; i-- {
		if err := stream.Send(&servicedefs.Out{Out: i}); err != nil {
			return errs.Wrap(err)
		}
	}
	return nil
}
