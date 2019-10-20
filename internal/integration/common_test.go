// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package integration

import (
	"context"
	"net"

	"github.com/zeebo/errs"
	"storj.io/drpc/drpcconn"
	"storj.io/drpc/drpcctx"
	"storj.io/drpc/drpcerr"
	"storj.io/drpc/drpcserver"
)

// because we have approximately no other heap usage, trying to run the tests with
// very high counts is a worst case for the garbage collector: it ends up running
// far too often and deleting everything. this makes it take ever longer (superlinear
// for some reason) to run with higher counts. this ballast fixes it by making it so
// that the heap goal is larger causing gc to run less frequently.
var ballast = make([]byte, 20*1024*1024) //nolint

//
// helpers
//

func in(n int64) *In   { return &In{In: n} }
func out(n int64) *Out { return &Out{Out: n} } //nolint

func createConnection(server DRPCServiceServer) (DRPCServiceClient, func()) {
	ctx := drpcctx.NewTracker(context.Background())
	c1, c2 := net.Pipe()

	srv := drpcserver.New()
	DRPCRegisterService(srv, server)
	ctx.Run(func(ctx context.Context) { _ = srv.ServeOne(ctx, c1) })
	conn := drpcconn.New(c2)

	return NewDRPCServiceClient(conn), func() {
		_ = conn.Close()
		ctx.Cancel()
		ctx.Wait()
	}
}

//
// server impl
//

type impl struct {
	Method1Fn func(ctx context.Context, in *In) (*Out, error)
	Method2Fn func(stream DRPCService_Method2Stream) error
	Method3Fn func(in *In, stream DRPCService_Method3Stream) error
	Method4Fn func(stream DRPCService_Method4Stream) error
}

func (i impl) Method1(ctx context.Context, in *In) (*Out, error) {
	return i.Method1Fn(ctx, in)
}

func (i impl) Method2(stream DRPCService_Method2Stream) error {
	return i.Method2Fn(stream)
}

func (i impl) Method3(in *In, stream DRPCService_Method3Stream) error {
	return i.Method3Fn(in, stream)
}

func (i impl) Method4(stream DRPCService_Method4Stream) error {
	return i.Method4Fn(stream)
}

//
// standard impl
//

var standardImpl = impl{
	Method1Fn: func(ctx context.Context, in *In) (*Out, error) {
		if in.In != 1 {
			return nil, drpcerr.WithCode(errs.New("test"), uint64(in.In))
		}
		return &Out{Out: 1}, nil
	},

	Method2Fn: func(stream DRPCService_Method2Stream) error {
		for {
			_, err := stream.Recv()
			if err != nil {
				break
			}
		}
		return stream.SendAndClose(&Out{Out: 2})
	},

	Method3Fn: func(in *In, stream DRPCService_Method3Stream) error {
		_ = stream.Send(&Out{Out: 3})
		_ = stream.Send(&Out{Out: 3})
		_ = stream.Send(&Out{Out: 3})
		return nil
	},

	Method4Fn: func(stream DRPCService_Method4Stream) error {
		for {
			_, err := stream.Recv()
			if err != nil {
				break
			}
		}
		_ = stream.Send(&Out{Out: 4})
		_ = stream.Send(&Out{Out: 4})
		_ = stream.Send(&Out{Out: 4})
		_ = stream.Send(&Out{Out: 4})
		return nil
	},
}
