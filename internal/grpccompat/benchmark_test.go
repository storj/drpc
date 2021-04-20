// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package grpccompat

import (
	"context"
	"io"
	"testing"

	"github.com/zeebo/assert"
)

var benchmarkImpl = &serviceImpl{
	Method1Fn: func(ctx context.Context, in *In) (*Out, error) {
		return out(in.In), nil
	},

	Method2Fn: func(stream ServerMethod2Stream) error {
		for {
			_, err := stream.Recv()
			if err == io.EOF {
				break
			}
		}
		return stream.SendAndClose(out(3))
	},

	Method3Fn: func(in *In, stream ServerMethod3Stream) error {
		o := out(5)
		for i := int64(0); i < in.In; i++ {
			_ = stream.Send(o)
		}
		return nil
	},

	Method4Fn: func(stream ServerMethod4Stream) error {
		o := out(5)
		for {
			_, err := stream.Recv()
			if err == io.EOF {
				return nil
			} else if err != nil {
				return err
			}
			_ = stream.Send(o)
		}
	},
}

func benchmarkBoth(b *testing.B, fn func(b *testing.B, client Client)) {
	b.Run("GRPC", func(b *testing.B) {
		conn, close := createGRPCConnection(benchmarkImpl.GRPC())
		defer close()
		fn(b, grpcWrapper{conn})
	})

	b.Run("DRPC", func(b *testing.B) {
		conn, close := createDRPCConnection(benchmarkImpl.DRPC())
		defer close()
		fn(b, drpcWrapper{conn})
	})
}

func BenchmarkUnitary(b *testing.B) {
	benchmarkBoth(b, func(b *testing.B, client Client) {
		in := &In{In: 5}
		ctx := context.Background()

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = client.Method1(ctx, in)
		}
	})
}

func BenchmarkInputStream(b *testing.B) {
	benchmarkBoth(b, func(b *testing.B, client Client) {
		in := &In{In: 5}
		ctx := context.Background()

		stream, err := client.Method2(ctx)
		assert.NoError(b, err)

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = stream.Send(in)
		}
		_, _ = stream.CloseAndRecv()
	})
}

func BenchmarkOutputStream(b *testing.B) {
	benchmarkBoth(b, func(b *testing.B, client Client) {
		ctx := context.Background()

		stream, err := client.Method3(ctx, in(int64(b.N)))
		assert.NoError(b, err)

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = stream.Recv()
		}
	})
}

func BenchmarkBidirectionalStream(b *testing.B) {
	benchmarkBoth(b, func(b *testing.B, client Client) {
		in := &In{In: 5}
		ctx := context.Background()

		stream, err := client.Method4(ctx)
		assert.NoError(b, err)

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = stream.Send(in)
			_, _ = stream.Recv()
		}
	})
}
