// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package integration

import (
	"context"
	"io"
	"net"
	"strconv"

	"github.com/spacemonkeygo/monkit/v3"
	"github.com/zeebo/errs"

	drpc "storj.io/drpc"
	"storj.io/drpc/drpcconn"
	"storj.io/drpc/drpcctx"
	"storj.io/drpc/drpcerr"
	"storj.io/drpc/drpcmetadata"
	"storj.io/drpc/drpcmux"
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

	mux := drpcmux.New()
	traceHandler := handler{
		mu: mux,
	}
	_ = DRPCRegisterService(mux, server)
	srv := drpcserver.New(&traceHandler)
	ctx.Run(func(ctx context.Context) { _ = srv.ServeOne(ctx, c1) })
	conn := drpcconn.New(c2)

	return NewDRPCServiceClient(conn), func() {
		_ = conn.Close()
		ctx.Cancel()
		ctx.Wait()
	}
}

type streamWrapper struct {
	drpc.Stream
	ctx context.Context
}

func (s *streamWrapper) Context() context.Context { return s.ctx }

type handler struct {
	mu *drpcmux.Mux
}

func (handler *handler) HandleRPC(stream drpc.Stream, rpc string) (err error) {
	streamCtx := stream.Context()
	metadata, ok := drpcmetadata.Get(streamCtx)
	if ok {
		parentID, err := strconv.ParseInt(metadata[INVOKE_HEADER_PARENTID], 10, 64)
		if err != nil {
			return errs.New("parse error")
		}

		traceID, err := strconv.ParseInt(metadata[INVOKE_HEADER_TRACEID], 10, 64)
		if err != nil {
			return errs.New("parse error")
		}
		newTrace := monkit.NewTrace(traceID)
		newTrace.Set(2, parentID)
		f := mon.Func()
		defer f.RemoteTrace(&streamCtx, monkit.NewId(), newTrace)(&err)
	}

	return handler.mu.HandleRPC(&streamWrapper{Stream: stream, ctx: streamCtx}, rpc)
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
		defer mon.Task()(&ctx)(nil)
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
