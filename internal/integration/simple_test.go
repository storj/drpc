// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package integration

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/spacemonkeygo/monkit/v3"
	"github.com/spacemonkeygo/monkit/v3/present"
	"github.com/zeebo/assert"

	"storj.io/drpc/drpcctx"
	"storj.io/drpc/drpcerr"
	jaeger "storj.io/monkit-jaeger"
)

var mon = monkit.Package()

func TestSimple(t *testing.T) {
	go http.ListenAndServe("localhost:9000", present.HTTP(monkit.Default))
	collector, err := jaeger.NewUDPCollector("localhost:5775", 250, "test")
	if err != nil {
		panic(err)
	}
	jaeger.RegisterJaeger(monkit.Default, collector, jaeger.Options{
		Fraction: 1})

	tracker := drpcctx.NewTracker(context.Background())
	defer mon.Task()(&tracker.Context)(nil)
	defer tracker.Wait()
	defer tracker.Cancel()

	cli, close := createConnection(standardImpl)
	defer close()

	{
		out, err := cli.Method1(tracker, &In{In: 1})
		assert.NoError(t, err)
		assert.DeepEqual(t, out, &Out{Out: 1})
	}

	{
		stream, err := cli.Method2(tracker)
		assert.NoError(t, err)
		assert.NoError(t, stream.Send(&In{In: 2}))
		assert.NoError(t, stream.Send(&In{In: 2}))
		out, err := stream.CloseAndRecv()
		assert.NoError(t, err)
		assert.DeepEqual(t, out, &Out{Out: 2})
	}

	{
		stream, err := cli.Method3(tracker, &In{In: 3})
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
		stream, err := cli.Method4(tracker)
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
		_, err := cli.Method1(tracker, &In{In: 5})
		assert.Error(t, err)
		assert.Equal(t, drpcerr.Code(err), 5)
	}

	time.Sleep(5 * time.Second)

}
