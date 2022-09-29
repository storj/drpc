// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"storj.io/drpc/drpcconn"

	"storj.io/drpc/examples/opentelemetry/pb"
)

func main() {
	err := Main(context.Background())
	if err != nil {
		panic(err)
	}
}

func Main(ctx context.Context) error {
	// start outputting spans to standard output
	defer startTelemetryToStdout()()

	// dial the drpc server
	rawconn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		return err
	}
	// N.B.: If you want TLS, you need to wrap the net.Conn with TLS before
	// making a DRPC conn.

	// convert the net.Conn to a drpc.Conn
	conn := drpcconn.New(rawconn)
	defer conn.Close()

	// wrap the drpc.Conn with otel
	oconn := &otelConn{Conn: conn}

	// make a drpc proto-specific client
	client := pb.NewDRPCCookieMonsterClient(oconn)

	// set a deadline for the operation
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	// run the RPC
	crumbs, err := client.EatCookie(ctx, &pb.Cookie{
		Type: pb.Cookie_Oatmeal,
	})
	if err != nil {
		return err
	}

	// check the results
	_, err = fmt.Println(crumbs.Cookie.Type.String())
	return err
}

//
// otel things
//

var tracer = otel.Tracer("storj.io/drpc/examples/opentelemetry/server")

// newExporter returns a console exporter.
func newExporter(w io.Writer) (trace.SpanExporter, error) {
	return stdouttrace.New(
		stdouttrace.WithWriter(w),
		// Use human-readable output.
		stdouttrace.WithPrettyPrint(),
		// Do not print timestamps for the demo.
		stdouttrace.WithoutTimestamps(),
	)
}

// newResource returns a resource describing this application.
func newResource() *resource.Resource {
	r, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("opentelemetry-client"),
			semconv.ServiceVersionKey.String("v0.1.0"),
			attribute.String("environment", "demo"),
		),
	)
	return r
}

func startTelemetryToStdout() func() {
	otel.SetTextMapPropagator(propagation.TraceContext{})

	exp, err := newExporter(os.Stdout)
	if err != nil {
		panic(err)
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithResource(newResource()),
	)

	otel.SetTracerProvider(tp)

	return func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			panic(err)
		}
	}
}
