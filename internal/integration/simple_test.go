// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package integration

//go:generate bash -c "go install storj.io/drpc/cmd/protoc-gen-drpc && protoc --drpc_out=plugins=drpc:. service.proto"

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/zeebo/assert"
	"github.com/zeebo/errs"

	"storj.io/drpc/drpcconn"
	"storj.io/drpc/drpcerr"
	"storj.io/drpc/drpcserver"
)

func TestSimple(t *testing.T) {
	rwc := func(r io.Reader, w io.Writer, c io.Closer) io.ReadWriteCloser {
		return struct {
			io.Reader
			io.Writer
			io.Closer
		}{r, w, c}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pr1, pw1 := io.Pipe()
	pr2, pw2 := io.Pipe()
	defer pr1.Close()
	defer pr2.Close()

	srv := drpcserver.New()
	srv.Register(new(impl), new(DRPCServiceDescription))
	go srv.ServeOne(rwc(pr2, pw1, pr2))

	conn := drpcconn.New(rwc(pr1, pw2, pr1))
	defer conn.Close()
	cli := NewDRPCServiceClient(conn)

	{
		out, err := cli.Method1(ctx, &In{In: 1})
		assert.NoError(t, err)
		assert.DeepEqual(t, out, &Out{Out: 1})
	}

	{
		stream, err := cli.Method2(ctx)
		assert.NoError(t, err)
		assert.NoError(t, stream.Send(&In{In: 2}))
		assert.NoError(t, stream.Send(&In{In: 2}))
		out, err := stream.CloseAndRecv()
		assert.NoError(t, err)
		assert.DeepEqual(t, out, &Out{Out: 2})
	}

	{
		stream, err := cli.Method3(ctx, &In{In: 3})
		assert.NoError(t, err)
		for {
			out, err := stream.Recv()
			if err == io.EOF {
				break
			}
			assert.NoError(t, err)
			assert.DeepEqual(t, out, &Out{Out: 3})
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
			if err == io.EOF {
				break
			}
			assert.NoError(t, err)
			assert.DeepEqual(t, out, &Out{Out: 4})
		}
	}

	{
		_, err := cli.Method1(ctx, &In{In: 5})
		assert.Error(t, err)
		assert.Equal(t, drpcerr.Code(err), 5)
	}
}

type impl struct{}

func (impl) DRPCMethod1(ctx context.Context, in *In) (*Out, error) {
	fmt.Println("SRV1 0 <=", in)
	if in.In != 1 {
		return nil, drpcerr.WithCode(errs.New("test"), uint64(in.In))
	}
	return &Out{Out: 1}, nil
}

func (impl) DRPCMethod2(stream DRPCService_Method2Stream) error {
	for {
		in, err := stream.Recv()
		fmt.Println("SRV2 0 <=", err, in)
		if err != nil {
			break
		}
	}
	err := stream.SendAndClose(&Out{Out: 2})
	fmt.Println("SRV2 1 <=", err)
	return err
}

func (impl) DRPCMethod3(in *In, stream DRPCService_Method3Stream) error {
	fmt.Println("SRV3 0 <=", in)
	fmt.Println("SRV3 1 <=", stream.Send(&Out{Out: 3}))
	fmt.Println("SRV3 2 <=", stream.Send(&Out{Out: 3}))
	fmt.Println("SRV3 3 <=", stream.Send(&Out{Out: 3}))
	return nil
}

func (impl) DRPCMethod4(stream DRPCService_Method4Stream) error {
	for {
		in, err := stream.Recv()
		fmt.Println("SRV4 0 <=", err, in)
		if err != nil {
			break
		}
	}
	fmt.Println("SRV4 1 <=", stream.Send(&Out{Out: 4}))
	fmt.Println("SRV4 2 <=", stream.Send(&Out{Out: 4}))
	fmt.Println("SRV4 3 <=", stream.Send(&Out{Out: 4}))
	fmt.Println("SRV4 4 <=", stream.Send(&Out{Out: 4}))
	return nil
}
