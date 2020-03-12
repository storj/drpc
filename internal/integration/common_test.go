// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package integration

import (
	"context"
	"io"
	"net"

	"github.com/zeebo/errs"
	"storj.io/drpc/drpcconn"
	"storj.io/drpc/drpcctx"
	"storj.io/drpc/drpcerr"
	"storj.io/drpc/drpcserver"
	"storj.io/drpc/drpcsignal"
)

//
// helpers
//

func in(n int64) *In { return &In{In: n} }

func createConnection(server DRPCServiceServer) (DRPCServiceClient, func()) {
	ctx := drpcctx.NewTracker(context.Background())
	c1, c2 := net.Pipe()

	srv := drpcserver.New()
	DRPCRegisterService(srv, server)

	ctx.Run(func(ctx context.Context) {
		_ = srv.ServeOne(ctx, c1)
	})
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

//
// drpc.Transport that signals when calls are made and blocks until closed
//

type transportSignaler struct {
	read  drpcsignal.Signal
	write drpcsignal.Signal
	done  drpcsignal.Signal
}

func (w *transportSignaler) Close() error {
	w.done.Set(nil)
	return nil
}

func (w *transportSignaler) Read(p []byte) (n int, err error) {
	w.read.Set(nil)
	<-w.done.Signal()
	return 0, io.ErrUnexpectedEOF
}

func (w *transportSignaler) Write(p []byte) (n int, err error) {
	w.write.Set(nil)
	<-w.done.Signal()
	return 0, io.ErrUnexpectedEOF
}
