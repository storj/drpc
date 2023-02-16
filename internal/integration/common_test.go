// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package integration

import (
	"context"
	"io"
	"net"
	"strconv"
	"sync"
	"testing"

	"github.com/zeebo/errs"

	"storj.io/drpc/drpcconn"
	"storj.io/drpc/drpcerr"
	"storj.io/drpc/drpcmanager"
	"storj.io/drpc/drpcmetadata"
	"storj.io/drpc/drpcmux"
	"storj.io/drpc/drpcserver"
	"storj.io/drpc/drpcsignal"
	"storj.io/drpc/drpctest"
)

//
// helpers
//

func data(n int64) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = byte(int(n) + i)
	}
	return out
}

func in(n int64) *In   { return &In{In: n} }
func out(n int64) *Out { return &Out{Out: n} }

func createRawConnection(t testing.TB, server DRPCServiceServer, ctx *drpctest.Tracker) *drpcconn.Conn {
	c1, c2 := net.Pipe()
	mux := drpcmux.New()
	_ = DRPCRegisterService(mux, server)
	srv := drpcserver.New(mux)
	ctx.Run(func(ctx context.Context) { _ = srv.ServeOne(ctx, c1) })
	return drpcconn.NewWithOptions(c2, drpcconn.Options{
		Manager: drpcmanager.Options{
			SoftCancel: true,
		},
	})
}

func createConnection(t testing.TB, server DRPCServiceServer) (DRPCServiceClient, func()) {
	ctx := drpctest.NewTracker(t)
	conn := createRawConnection(t, server, ctx)
	return NewDRPCServiceClient(conn), func() {
		_ = conn.Close()
		ctx.Close()
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

		var out int64 = 1
		if metadata, ok := drpcmetadata.Get(ctx); ok {
			v, _ := strconv.ParseInt(metadata["inc"], 10, 64)
			out += v
		}

		return &Out{Out: out, Data: in.Data}, nil
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

	Method3Fn: func(_ *In, stream DRPCService_Method3Stream) error {
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

//
// drpc.Transport that allows one to control when blocking starts/stops
//

type transportBlocker struct {
	cond  *sync.Cond
	done  bool
	write bool
}

func newTransportBlocker() *transportBlocker {
	return &transportBlocker{
		cond:  sync.NewCond(new(sync.Mutex)),
		write: true,
	}
}

func (w *transportBlocker) BlockWrites() {
	w.cond.L.Lock()
	defer w.cond.L.Unlock()

	w.write = false
	w.cond.Broadcast()
}

func (w *transportBlocker) UnblockWrites() {
	w.cond.L.Lock()
	defer w.cond.L.Unlock()

	w.write = true
	w.cond.Broadcast()
}

func (w *transportBlocker) Read(p []byte) (n int, err error) {
	w.cond.L.Lock()
	defer w.cond.L.Unlock()

	for !w.done {
		w.cond.Wait()
	}

	return 0, io.EOF
}

func (w *transportBlocker) Write(p []byte) (n int, err error) {
	w.cond.L.Lock()
	defer w.cond.L.Unlock()

	for !w.write && !w.done {
		w.cond.Wait()
	}

	if w.done {
		return 0, io.EOF
	}
	return len(p), nil
}

func (w *transportBlocker) Close() error {
	w.cond.L.Lock()
	defer w.cond.L.Unlock()

	w.done = true
	w.cond.Broadcast()
	return nil
}
