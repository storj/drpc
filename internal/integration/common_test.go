// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package integration

import (
	"context"
	"fmt"
	"io"

	"github.com/zeebo/errs"
	"storj.io/drpc/drpcconn"
	"storj.io/drpc/drpcctx"
	"storj.io/drpc/drpcdebug"
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

func in(n int64) *In { return &In{In: n} }

func rwc(r io.Reader, w io.Writer, c io.Closer) io.ReadWriteCloser {
	return struct {
		io.Reader
		io.Writer
		io.Closer
	}{r, w, c}
}

func createConnection(ctx *drpcctx.Tracker) (DRPCServiceClient, func()) {
	pr1, pw1 := io.Pipe()
	pr2, pw2 := io.Pipe()

	srv := drpcserver.New()
	DRPCRegisterService(srv, new(impl))
	ctx.Run(func(ctx context.Context) { _ = srv.ServeOne(ctx, rwc(pr2, pw1, pr2)) })
	conn := drpcconn.New(rwc(pr1, pw2, pr1))

	return NewDRPCServiceClient(conn), func() {
		conn.Close()
		pr2.Close()
		pr1.Close()
	}
}

//
// server impl
//

type impl struct{}

func (impl) Method1(ctx context.Context, in *In) (*Out, error) {
	drpcdebug.Log(func() string { return fmt.Sprintln("SRV1 0 <=", in) })
	if in.In != 1 {
		return nil, drpcerr.WithCode(errs.New("test"), uint64(in.In))
	}
	return &Out{Out: 1}, nil
}

func (impl) Method2(stream DRPCService_Method2Stream) error {
	for {
		in, err := stream.Recv()
		drpcdebug.Log(func() string { return fmt.Sprintln("SRV2 0 <=", err, in) })
		if err != nil {
			break
		}
	}
	err := stream.SendAndClose(&Out{Out: 2})
	drpcdebug.Log(func() string { return fmt.Sprintln("SRV2 1 <=", err) })
	return err
}

func (impl) Method3(in *In, stream DRPCService_Method3Stream) error {
	drpcdebug.Log(func() string { return fmt.Sprintln("SRV3 0 <=", in) })
	drpcdebug.Log(func() string { return fmt.Sprintln("SRV3 1 <=", stream.Send(&Out{Out: 3})) })
	drpcdebug.Log(func() string { return fmt.Sprintln("SRV3 2 <=", stream.Send(&Out{Out: 3})) })
	drpcdebug.Log(func() string { return fmt.Sprintln("SRV3 3 <=", stream.Send(&Out{Out: 3})) })
	return nil
}

func (impl) Method4(stream DRPCService_Method4Stream) error {
	for {
		in, err := stream.Recv()
		drpcdebug.Log(func() string { return fmt.Sprintln("SRV4 0 <=", err, in) })
		if err != nil {
			break
		}
	}
	drpcdebug.Log(func() string { return fmt.Sprintln("SRV4 1 <=", stream.Send(&Out{Out: 4})) })
	drpcdebug.Log(func() string { return fmt.Sprintln("SRV4 2 <=", stream.Send(&Out{Out: 4})) })
	drpcdebug.Log(func() string { return fmt.Sprintln("SRV4 3 <=", stream.Send(&Out{Out: 4})) })
	drpcdebug.Log(func() string { return fmt.Sprintln("SRV4 4 <=", stream.Send(&Out{Out: 4})) })
	return nil
}
