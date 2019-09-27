// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package grpccompat

import (
	"context"
	"errors"
	"flag"
	fmt "fmt"
	"io"
	"net"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/zeebo/errs"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"storj.io/drpc/drpcconn"
	"storj.io/drpc/drpcctx"
	"storj.io/drpc/drpcserver"
)

var fullErrors = flag.Bool("full-errors", false, "if true, display full errors in logs")

//
// test result helpers
//

type errResult int

const (
	errResult_Invalid errResult = iota
	errResult_None
	errResult_Canceled
	errResult_DeadlineExceeded
	errResult_EOF
	errResult_Marker
	errResult_Other
)

func (e errResult) String() string {
	switch e {
	case errResult_None:
		return "None"
	case errResult_Canceled:
		return "Canceled"
	case errResult_DeadlineExceeded:
		return "DeadlineExceeded"
	case errResult_EOF:
		return "EOF"
	case errResult_Marker:
		return "Marker"
	case errResult_Other:
		return "Other"
	default:
		return "Invalid"
	}
}

type result struct {
	out *Out
	err error
}

func (res result) String() string {
	if *fullErrors {
		return fmt.Sprintf("<out:%s err:%s[%v]>", res.out, getErrResult(res.err), res.err)
	}
	return fmt.Sprintf("<out:%s err:%s>", res.out, getErrResult(res.err))
}

func getResult(out *Out, err error) (res result) {
	res.out = out
	res.err = err
	return res
}

func getErrResult(err error) errResult {
	switch code := status.Code(err); {
	case err == nil:
		return errResult_None
	case code == codes.Canceled, errors.Is(err, context.Canceled):
		return errResult_Canceled
	case code == codes.DeadlineExceeded, errors.Is(err, context.DeadlineExceeded):
		return errResult_DeadlineExceeded
	case errors.Is(err, io.EOF):
		return errResult_EOF
	case strings.Contains(err.Error(), "marker"):
		return errResult_Marker
	default:
		return errResult_Other
	}
}

func resultsEqual(a, b result) bool {
	return reflect.DeepEqual(a.out, b.out) && getErrResult(a.err) == getErrResult(b.err)
}

func allResultsEqual(a, b []result) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !resultsEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

type testFunc func(*testing.T, Client, func(*Out, error))

func collectResults(t *testing.T, cli Client, fn testFunc) []result {
	var results []result
	fn(t, cli, func(out *Out, err error) {
		results = append(results, getResult(out, err))
	})
	return results
}

func testCompat(t *testing.T, impl *serviceImpl, fn testFunc) {
	sg := runtime.NumGoroutine()
	defer func() {
		start := time.Now()
		for {
			if cg := runtime.NumGoroutine(); sg == cg {
				return
			} else if time.Since(start) > 10*time.Second {
				t.Fatal("goroutine leak:", sg, "=>", cg)
			}
		}
	}()

	grpcClient, close := createGRPCConnection(impl.GRPC())
	defer close()
	grpcResults := collectResults(t, grpcWrapper{grpcClient}, fn)
	t.Logf("grpc: %s", grpcResults)

	drpcClient, close := createDRPCConnection(impl.DRPC())
	defer close()
	drpcResults := collectResults(t, drpcWrapper{drpcClient}, fn)
	t.Logf("drpc: %s", drpcResults)

	if !allResultsEqual(grpcResults, drpcResults) {
		t.FailNow()
	}
}

//
// helpers
//

func in(n int64) *In   { return &In{In: n} }
func out(n int64) *Out { return &Out{Out: n} }

func createDRPCConnection(server DRPCServiceServer) (DRPCServiceClient, func()) {
	ctx := drpcctx.NewTracker(context.Background())
	c1, c2 := net.Pipe()

	srv := drpcserver.New()
	DRPCRegisterService(srv, server)
	ctx.Run(func(ctx context.Context) { _ = srv.ServeOne(ctx, c1) })
	conn := drpcconn.New(c2)

	return NewDRPCServiceClient(conn), func() {
		_ = conn.Close()
		ctx.Cancel()
		ctx.Wait()
	}
}

func createGRPCConnection(server ServiceServer) (ServiceClient, func()) {
	ctx := drpcctx.NewTracker(context.Background())
	c1, c2 := net.Pipe()

	srv := grpc.NewServer()
	RegisterServiceServer(srv, server)

	lis := makeListener(ctx, c1)
	ctx.Run(func(context.Context) { _ = srv.Serve(lis) })
	cc, _ := grpc.Dial("",
		grpc.WithInsecure(),
		grpc.WithContextDialer(makeDialer(c2)))

	return NewServiceClient(cc), func() {
		_ = lis.Close()
		_ = cc.Close()
		ctx.Cancel()
		ctx.Wait()
	}
}

//
// connection helpers
//

func makeDialer(conn net.Conn) func(context.Context, string) (net.Conn, error) {
	return func(context.Context, string) (net.Conn, error) { return conn, nil }
}

type listenOne struct {
	conn   net.Conn
	done   <-chan struct{}
	cancel func()
}

func makeListener(ctx context.Context, conn net.Conn) *listenOne {
	ctx, cancel := context.WithCancel(ctx)
	return &listenOne{
		conn:   conn,
		done:   ctx.Done(),
		cancel: cancel,
	}
}

func (l *listenOne) Close() error   { l.cancel(); return nil }
func (l *listenOne) Addr() net.Addr { return nil }
func (l *listenOne) Accept() (conn net.Conn, err error) {
	if l.conn != nil {
		conn, l.conn = l.conn, nil
		return conn, nil
	}
	<-l.done
	return nil, errs.New("listener closed")
}

//
// agnostic client impl
//

type Client interface {
	Method1(ctx context.Context, in *In) (*Out, error)
	Method2(ctx context.Context) (Client_Method2Stream, error)
	Method3(ctx context.Context, in *In) (Client_Method3Stream, error)
	Method4(ctx context.Context) (Client_Method4Stream, error)
}

type Client_Method2Stream interface {
	Send(*In) error
	CloseAndRecv() (*Out, error)
}

type Client_Method3Stream interface {
	Recv() (*Out, error)
}

type Client_Method4Stream interface {
	Send(*In) error
	Recv() (*Out, error)
}

//
// grpc client wrapper
//

type grpcWrapper struct{ c ServiceClient }

func (g grpcWrapper) Method1(ctx context.Context, in *In) (*Out, error) {
	return g.c.Method1(ctx, in)
}
func (g grpcWrapper) Method2(ctx context.Context) (Client_Method2Stream, error) {
	return g.c.Method2(ctx)
}
func (g grpcWrapper) Method3(ctx context.Context, in *In) (Client_Method3Stream, error) {
	return g.c.Method3(ctx, in)
}
func (g grpcWrapper) Method4(ctx context.Context) (Client_Method4Stream, error) {
	return g.c.Method4(ctx)
}

//
// drpc client wrapper
//

type drpcWrapper struct{ c DRPCServiceClient }

func (d drpcWrapper) Method1(ctx context.Context, in *In) (*Out, error) {
	return d.c.Method1(ctx, in)
}
func (d drpcWrapper) Method2(ctx context.Context) (Client_Method2Stream, error) {
	return d.c.Method2(ctx)
}
func (d drpcWrapper) Method3(ctx context.Context, in *In) (Client_Method3Stream, error) {
	return d.c.Method3(ctx, in)
}
func (d drpcWrapper) Method4(ctx context.Context) (Client_Method4Stream, error) {
	return d.c.Method4(ctx)
}

//
// agnostic server impl
//

type serviceImpl struct {
	Method1Fn func(ctx context.Context, in *In) (*Out, error)
	Method2Fn func(stream Server_Method2Stream) error
	Method3Fn func(in *In, stream Server_Method3Stream) error
	Method4Fn func(stream Server_Method4Stream) error
}

type Server_Method2Stream interface {
	Context() context.Context

	Recv() (*In, error)
	SendAndClose(*Out) error
}

type Server_Method3Stream interface {
	Context() context.Context

	Send(*Out) error
}

type Server_Method4Stream interface {
	Context() context.Context

	Send(*Out) error
	Recv() (*In, error)
}

func (i *serviceImpl) DRPC() (d *drpcImpl) {
	d = new(drpcImpl)
	if i.Method1Fn != nil {
		d.Method1Fn = func(ctx context.Context, in *In) (*Out, error) { return i.Method1Fn(ctx, in) }
	}
	if i.Method2Fn != nil {
		d.Method2Fn = func(stream DRPCService_Method2Stream) error { return i.Method2Fn(stream) }
	}
	if i.Method3Fn != nil {
		d.Method3Fn = func(in *In, stream DRPCService_Method3Stream) error { return i.Method3Fn(in, stream) }
	}
	if i.Method4Fn != nil {
		d.Method4Fn = func(stream DRPCService_Method4Stream) error { return i.Method4Fn(stream) }
	}
	return d
}

func (i *serviceImpl) GRPC() (g *grpcImpl) {
	g = new(grpcImpl)
	if i.Method1Fn != nil {
		g.Method1Fn = func(ctx context.Context, in *In) (*Out, error) { return i.Method1Fn(ctx, in) }
	}
	if i.Method2Fn != nil {
		g.Method2Fn = func(stream Service_Method2Server) error { return i.Method2Fn(stream) }
	}
	if i.Method3Fn != nil {
		g.Method3Fn = func(in *In, stream Service_Method3Server) error { return i.Method3Fn(in, stream) }
	}
	if i.Method4Fn != nil {
		g.Method4Fn = func(stream Service_Method4Server) error { return i.Method4Fn(stream) }
	}
	return g
}

//
// drpc server impl
//

type drpcImpl struct {
	Method1Fn func(ctx context.Context, in *In) (*Out, error)
	Method2Fn func(stream DRPCService_Method2Stream) error
	Method3Fn func(in *In, stream DRPCService_Method3Stream) error
	Method4Fn func(stream DRPCService_Method4Stream) error
}

func (d drpcImpl) Method1(ctx context.Context, in *In) (*Out, error) {
	return d.Method1Fn(ctx, in)
}

func (d drpcImpl) Method2(stream DRPCService_Method2Stream) error {
	return d.Method2Fn(stream)
}

func (d drpcImpl) Method3(in *In, stream DRPCService_Method3Stream) error {
	return d.Method3Fn(in, stream)
}

func (d drpcImpl) Method4(stream DRPCService_Method4Stream) error {
	return d.Method4Fn(stream)
}

//
// grpc server impl
//

type grpcImpl struct {
	Method1Fn func(ctx context.Context, in *In) (*Out, error)
	Method2Fn func(stream Service_Method2Server) error
	Method3Fn func(in *In, stream Service_Method3Server) error
	Method4Fn func(stream Service_Method4Server) error
}

func (g grpcImpl) Method1(ctx context.Context, in *In) (*Out, error) {
	return g.Method1Fn(ctx, in)
}

func (g grpcImpl) Method2(stream Service_Method2Server) error {
	return g.Method2Fn(stream)
}

func (g grpcImpl) Method3(in *In, stream Service_Method3Server) error {
	return g.Method3Fn(in, stream)
}

func (g grpcImpl) Method4(stream Service_Method4Server) error {
	return g.Method4Fn(stream)
}
