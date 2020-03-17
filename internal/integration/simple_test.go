// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package integration

import (
	"context"
	"io"
	"net/http"
	"strconv"
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

const INVOKE_HEADER_TRACEID = "trace-id"

const INVOKE_HEADER_PARENTID = "parent-id"

func TestSimple(t *testing.T) {
	go http.ListenAndServe("localhost:9000", present.HTTP(monkit.Default))
	collector, err := jaeger.NewUDPCollector("localhost:5775", 250, "test")
	if err != nil {
		panic(err)
	}
	jaeger.RegisterJaeger(monkit.Default, collector, jaeger.Options{
		Fraction: 1})

	trackerCtx := context.Background()
	tracker := drpcctx.NewTracker(trackerCtx)
	defer mon.Task()(&tracker.Context)(nil)
	defer tracker.Wait()
	defer tracker.Cancel()

	cli, close := createConnection(standardImpl)
	defer close()

	{
		span := monkit.SpanFromCtx(tracker.Context)
		tracker.Context = drpcctx.WithMetadata(tracker.Context, INVOKE_HEADER_TRACEID, strconv.FormatInt(span.Trace().Id(), 10))
		tracker.Context = drpcctx.WithMetadata(tracker.Context, INVOKE_HEADER_PARENTID, strconv.FormatInt(span.Id(), 10))
		out, err := cli.Method1(tracker, &In{In: 1})
		assert.NoError(t, err)
		assert.DeepEqual(t, out, &Out{Out: 1})
		time.Sleep(5 * time.Second)
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
}
