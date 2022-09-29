// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"context"
	"io"
	"net"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"storj.io/drpc/drpcmux"
	"storj.io/drpc/drpcserver"

	"storj.io/drpc/examples/opentelemetry/pb"
)

type CookieMonsterServer struct {
	pb.DRPCCookieMonsterUnimplementedServer
	// struct fields
}

// EatCookie turns a cookie into crumbs.
func (s *CookieMonsterServer) EatCookie(ctx context.Context, cookie *pb.Cookie) (*pb.Crumbs, error) {
	ctx, span := tracer.Start(ctx, "EatCookie")
	defer span.End()

	return s.chewCookie(ctx, cookie), nil
}

func (s *CookieMonsterServer) chewCookie(ctx context.Context, cookie *pb.Cookie) *pb.Crumbs {
	_, span := tracer.Start(ctx, "chewCookie")
	defer span.End()

	return &pb.Crumbs{
		Cookie: cookie,
	}
}

func main() {
	err := Main(context.Background())
	if err != nil {
		panic(err)
	}
}

func Main(ctx context.Context) error {
	// start outputting spans to standard output
	defer startTelemetryToStdout()()

	// create an RPC server
	cookieMonster := &CookieMonsterServer{}

	// create a drpc RPC mux
	m := drpcmux.New()

	// register the proto-specific methods on the mux
	err := pb.DRPCRegisterCookieMonster(m, cookieMonster)
	if err != nil {
		return err
	}

	// wrap the mux with the otel handler
	h := &otelHandler{handler: m}

	// create a drpc server
	s := drpcserver.New(h)

	// listen on a tcp socket
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		return err
	}

	// run the server
	// N.B.: if you want TLS, you need to wrap the net.Listener with
	// TLS before passing to Serve here.
	return s.Serve(ctx, lis)
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
			semconv.ServiceNameKey.String("opentelemetry-server"),
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
