// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"storj.io/drpc"
	"storj.io/drpc/drpcmetadata"
)

// otelConn wraps a drpc.Conn with tracing information.
type otelConn struct {
	drpc.Conn
}

// Invoke implements drpc.Conn's Invoke method with tracing information injected into the context.
func (c *otelConn) Invoke(ctx context.Context, rpc string, enc drpc.Encoding, in drpc.Message, out drpc.Message) (err error) {
	ctx, span := tracer.Start(ctx, rpc)
	defer span.End()

	return c.Conn.Invoke(addMetadata(ctx), rpc, enc, in, out)
}

// NewStream implements drpc.Conn's NewStream method with tracing information injected into the context.
func (c *otelConn) NewStream(ctx context.Context, rpc string, enc drpc.Encoding) (_ drpc.Stream, err error) {
	ctx, span := tracer.Start(ctx, rpc)
	defer span.End()

	return c.Conn.NewStream(addMetadata(ctx), rpc, enc)
}

// addMetadata propagates the headers into a map that we inject into drpc metadata so they are
// sent across the wire for the server to get.
func addMetadata(ctx context.Context) context.Context {
	metadata := make(map[string]string)
	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(metadata))
	return drpcmetadata.AddPairs(ctx, metadata)
}
