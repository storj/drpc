// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package grpccompat

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/zeebo/assert"
	"github.com/zeebo/errs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"storj.io/drpc"
	"storj.io/drpc/drpcconn"
	"storj.io/drpc/drpcerr"
	"storj.io/drpc/drpchttp"
	"storj.io/drpc/drpcmanager"
	"storj.io/drpc/drpcmux"
	"storj.io/drpc/drpcserver"
	"storj.io/drpc/drpctest"
)

var fullErrors = flag.Bool("full-errors", false, "if true, display full errors in logs")

//
// test result helpers
//

type errResult int

const (
	_ errResult = iota
	errResultNone
	errResultCanceled
	errResultDeadlineExceeded
	errResultEOF
	errResultMarker
	errResultOther
)

func (e errResult) String() string {
	switch e {
	case errResultNone:
		return "None"
	case errResultCanceled:
		return "Canceled"
	case errResultDeadlineExceeded:
		return "DeadlineExceeded"
	case errResultEOF:
		return "EOF"
	case errResultMarker:
		return "Marker"
	case errResultOther:
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
		return errResultNone
	case code == codes.Canceled, errors.Is(err, context.Canceled):
		return errResultCanceled
	case code == codes.DeadlineExceeded, errors.Is(err, context.DeadlineExceeded):
		return errResultDeadlineExceeded
	case errors.Is(err, io.EOF):
		return errResultEOF
	case strings.Contains(err.Error(), "marker"):
		return errResultMarker
	default:
		return errResultOther
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
	defer checkGoroutines(t)

	grpcClient, close := createGRPCConnection(t, impl.GRPC())
	defer close()
	grpcResults := collectResults(t, grpcWrapper{grpcClient}, fn)
	t.Logf("grpc: %s", grpcResults)

	drpcClient, close := createDRPCConnection(t, impl.DRPC())
	defer close()
	drpcResults := collectResults(t, drpcWrapper{drpcClient}, fn)
	t.Logf("drpc: %s", drpcResults)

	if !allResultsEqual(grpcResults, drpcResults) {
		t.FailNow()
	}
}

func testWebCompat(t *testing.T, impl *serviceImpl, fn testFunc) {
	defer checkGoroutines(t)

	results := [][]result{}

	grpcServer := createGRPCWebServer(impl.GRPC())
	defer grpcServer.Close()
	results = append(results, collectResults(t, webClient{grpcServer.URL, false}, fn))
	results = append(results, collectResults(t, webClient{grpcServer.URL, true}, fn))

	drpcServer := createDRPCWebServer(impl.DRPC())
	defer drpcServer.Close()
	results = append(results, collectResults(t, webClient{drpcServer.URL, false}, fn))
	results = append(results, collectResults(t, webClient{drpcServer.URL, true}, fn))

	grpcClient, close := createGRPCConnection(t, impl.GRPC())
	defer close()
	results = append(results, collectResults(t, grpcWrapper{grpcClient}, fn))

	drpcClient, close := createDRPCConnection(t, impl.DRPC())
	defer close()
	results = append(results, collectResults(t, drpcWrapper{drpcClient}, fn))

	for i := 0; i < len(results); i++ {
		t.Log(i, results[i])
		if i > 0 {
			assert.That(t, allResultsEqual(results[i-1], results[i]))
		}
	}
}

//
// helpers
//

func stackTrace() string {
	buf := make([]byte, 1024)
	for {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			return string(buf[:n])
		}
		buf = make([]byte, 2*len(buf))
	}
}

func checkGoroutines(t *testing.T) {
	if t.Failed() {
		return
	}

	start := time.Now()
	for {
		// github.com/improbable-eng/grpc-web/go/grpcweb ends up pulling in
		// some dependency that starts a background goroutine for some reason.
		// what the holy moly github.com/desertbit/timer? ugh.
		if cg := runtime.NumGoroutine(); cg == 3 {
			return
		} else if time.Since(start) > 10*time.Second {
			t.Fatalf("goroutine leak:\n%s", stackTrace())
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func in(n int64) *In   { return &In{In: n} }
func out(n int64) *Out { return &Out{Out: n} }

func asOut(in *In) *Out {
	return &Out{Out: in.In, Buf: in.Buf, Opt: in.Opt}
}

func createDRPCConnectionWithOptions(t testing.TB, server DRPCServiceServer, opts drpcmanager.Options) (DRPCServiceClient, func()) {
	ctx := drpctest.NewTracker(t)
	c1, c2 := pipe()

	mux := drpcmux.New()
	_ = DRPCRegisterService(mux, server)
	srv := drpcserver.NewWithOptions(mux, drpcserver.Options{
		Manager: opts,
	})
	ctx.Run(func(ctx context.Context) { _ = srv.ServeOne(ctx, c1) })
	conn := drpcconn.NewWithOptions(c2, drpcconn.Options{
		Manager: opts,
	})

	return NewDRPCServiceClient(conn), func() {
		_ = conn.Close()
		ctx.Close()
	}
}

func createDRPCConnection(t testing.TB, server DRPCServiceServer) (DRPCServiceClient, func()) {
	return createDRPCConnectionWithOptions(t, server, drpcmanager.Options{})
}

func createGRPCConnection(t testing.TB, server ServiceServer) (ServiceClient, func()) {
	ctx := drpctest.NewTracker(t)
	c1, c2 := pipe()

	srv := grpc.NewServer()
	RegisterServiceServer(srv, server)

	lis := makeListener(ctx, c1)
	ctx.Run(func(context.Context) { _ = srv.Serve(lis) })
	cc, _ := grpc.Dial("",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(makeDialer(c2)))

	return NewServiceClient(cc), func() {
		_ = lis.Close()
		_ = cc.Close()
		ctx.Close()
	}
}

func createDRPCWebServer(server DRPCServiceServer) *httptest.Server {
	mux := drpcmux.New()
	_ = DRPCRegisterService(mux, server)
	return httptest.NewServer(drpchttp.New(mux))
}

func createGRPCWebServer(server ServiceServer) *httptest.Server {
	srv := grpc.NewServer()
	RegisterServiceServer(srv, server)
	return httptest.NewServer(grpcweb.WrapServer(srv))
}

//
// connection helpers
//

func pipe() (drpc.Transport, drpc.Transport) {
	type rwc struct {
		io.Reader
		io.Writer
		io.Closer
	}

	c1r, c1w := io.Pipe()
	c2r, c2w := io.Pipe()

	return rwc{c1r, c2w, c2w}, rwc{c2r, c1w, c1w}
}

func makeDialer(tr drpc.Transport) func(context.Context, string) (net.Conn, error) {
	return func(context.Context, string) (net.Conn, error) { return transportConn{tr}, nil }
}

type listenOne struct {
	tr     drpc.Transport
	done   <-chan struct{}
	cancel func()
}

func makeListener(ctx context.Context, tr drpc.Transport) *listenOne {
	ctx, cancel := context.WithCancel(ctx)
	return &listenOne{
		tr:     tr,
		done:   ctx.Done(),
		cancel: cancel,
	}
}

func (l *listenOne) Close() error   { l.cancel(); return nil }
func (l *listenOne) Addr() net.Addr { return &net.TCPAddr{IP: net.IP{3: 0}} }
func (l *listenOne) Accept() (conn net.Conn, err error) {
	if l.tr != nil {
		conn, l.tr = transportConn{l.tr}, nil
		return conn, nil
	}
	<-l.done
	return nil, errs.New("listener closed")
}

type transportConn struct {
	drpc.Transport
}

func (transportConn) LocalAddr() net.Addr                { return nil }
func (transportConn) RemoteAddr() net.Addr               { return nil }
func (transportConn) SetDeadline(t time.Time) error      { return nil }
func (transportConn) SetReadDeadline(t time.Time) error  { return nil }
func (transportConn) SetWriteDeadline(t time.Time) error { return nil }

//
// agnostic client impl
//

type Client interface {
	Method1(ctx context.Context, in *In) (*Out, error)
	Method2(ctx context.Context) (ClientMethod2Stream, error)
	Method3(ctx context.Context, in *In) (ClientMethod3Stream, error)
	Method4(ctx context.Context) (ClientMethod4Stream, error)
}

type ClientMethod2Stream interface {
	Context() context.Context

	Send(*In) error
	CloseAndRecv() (*Out, error)
}

type ClientMethod3Stream interface {
	Context() context.Context

	Recv() (*Out, error)
}

type ClientMethod4Stream interface {
	Context() context.Context

	Send(*In) error
	Recv() (*Out, error)
}

//
// web client
//

func readExactly(r io.Reader, n uint64) ([]byte, error) {
	buf := make([]byte, n)
	_, err := io.ReadFull(r, buf)
	return buf, err
}

func grpcRead(r io.Reader) ([]byte, bool, error) {
	if tmp, err := readExactly(r, 5); err != nil {
		return nil, false, err
	} else if size := binary.BigEndian.Uint32(tmp[1:5]); size > 4<<20 {
		return nil, false, errs.New("message too large")
	} else if data, err := readExactly(r, uint64(size)); errors.Is(err, io.EOF) {
		return nil, false, io.ErrUnexpectedEOF
	} else if err != nil {
		return nil, false, err
	} else {
		return data, tmp[0] == 128, nil
	}
}

func framedData(buf []byte) []byte {
	var tmp [5]byte
	binary.BigEndian.PutUint32(tmp[1:5], uint32(len(buf)))
	return append(tmp[:], buf...)
}

func parseTrailers(buf []byte) (http.Header, error) {
	buf = append(bytes.TrimSpace(buf), "\r\n\r\n"...)
	body := bufio.NewReader(bytes.NewReader(buf))
	trailer, err := textproto.NewReader(body).ReadMIMEHeader()
	return http.Header(trailer), errs.Wrap(err)
}

func handleTrailers(trailers http.Header) error {
	if status := trailers.Get("Grpc-Status"); status == "" {
		return errs.New("no returned status")
	} else if s, err := strconv.ParseUint(status, 10, 64); err != nil {
		return errs.Wrap(err)
	} else if s != 0 {
		return drpcerr.WithCode(errs.New("%s", trailers.Get("Grpc-Message")), s)
	} else {
		return nil
	}
}

type webClient struct {
	url  string
	text bool
}

func (w webClient) Method1(ctx context.Context, in *In) (*Out, error) {
	buf, err := proto.Marshal(in)
	if err != nil {
		return nil, errs.Wrap(err)
	}

	url := w.url + "/service.Service/Method1"
	body := framedData(buf)
	ct := "application/grpc-web+proto"
	if w.text {
		body = []byte(base64.StdEncoding.EncodeToString(body))
		ct = "application/grpc-web-text+proto"
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, errs.Wrap(err)
	}
	req.Header.Set("Content-Type", ct)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errs.Wrap(err)
	}
	defer func() { _ = resp.Body.Close() }()

	// sooo, this is cool. it seems like grpc-web has its own mechanism for
	// adding trailers (by setting the MSB of the first byte of the framing
	// format) rather than just like, letting http handle it. i have no idea
	// why (probably browser support), and it's very hard to find any clients
	// that aren't a bunch of javascript i don't understand. and so, here i
	// am, blindly doing my best on what i think it's going for.
	//
	// another complication, btw, is that when no response body is present,
	// servers are allowed to send the trailers as headers instead of just
	// always sending them in the response body. why all this flexability?
	// the world may never know.

	// by default, start with the headers as the trailers, because they
	// might be there.
	trailers := resp.Header.Clone()
	out := new(Out)

	var reply io.Reader = resp.Body
	if w.text {
		reply = &base64Reader{r: resp.Body}
	}

	switch data, trailer, err := grpcRead(reply); {
	case errors.Is(err, io.EOF):
	case err != nil:
		return nil, errs.Wrap(err)
	case trailer:
		trailers, err = parseTrailers(data)
		if err != nil {
			return nil, err
		}
	default:
		if err := proto.Unmarshal(data, out); err != nil {
			return nil, errs.Wrap(err)
		}
	}

	switch data, trailer, err := grpcRead(reply); {
	case errors.Is(err, io.EOF):
	case err != nil:
		return nil, errs.Wrap(err)
	case !trailer:
		return nil, errs.New("expected trailers")
	default:
		trailers, err = parseTrailers(data)
		if err != nil {
			return nil, err
		}
	}

	if err := handleTrailers(trailers); err != nil {
		return nil, err
	}
	return out, nil
}

func (w webClient) Method3(ctx context.Context, in *In) (ClientMethod3Stream, error) {
	buf, err := proto.Marshal(in)
	if err != nil {
		return nil, errs.Wrap(err)
	}

	url := w.url + "/service.Service/Method3"
	body := framedData(buf)
	ct := "application/grpc-web+proto"
	if w.text {
		body = []byte(base64.StdEncoding.EncodeToString(body))
		ct = "application/grpc-web-text+proto"
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, errs.Wrap(err)
	}
	req.Header.Set("Content-Type", ct)

	resp, err := http.DefaultClient.Do(req) //nolint: bodyclose
	if err != nil {
		return nil, errs.Wrap(err)
	}

	var reply io.Reader = resp.Body
	if w.text {
		reply = &base64Reader{r: resp.Body}
	}

	return &webClientMethod3Stream{ctx: ctx, resp: resp, r: reply}, nil
}

func (w webClient) Method2(ctx context.Context) (ClientMethod2Stream, error) {
	panic("method2 not available")
}

func (w webClient) Method4(ctx context.Context) (ClientMethod4Stream, error) {
	panic("method4 not available")
}

type webClientMethod3Stream struct {
	ctx  context.Context
	err  error
	resp *http.Response
	r    io.Reader
}

func (w *webClientMethod3Stream) Context() context.Context { return w.ctx }

func (w *webClientMethod3Stream) Recv() (_ *Out, err error) {
	if w.err != nil {
		return nil, w.err
	}

	defer func() {
		w.err = err
		if err != nil {
			_ = w.resp.Body.Close()
		}
	}()

	trailers := w.resp.Header.Clone()

	switch data, trailer, err := grpcRead(w.r); {
	case errors.Is(err, io.EOF):
	case err != nil:
		return nil, err
	case trailer:
		trailers, err = parseTrailers(data)
		if err != nil {
			return nil, err
		}
	default:
		out := new(Out)
		if err := proto.Unmarshal(data, out); err != nil {
			return nil, err
		}
		return out, nil
	}

	if err := handleTrailers(trailers); err != nil {
		return nil, err
	}
	return nil, io.EOF
}

type base64Reader struct {
	r io.Reader
	b io.Reader
}

func (b *base64Reader) Read(p []byte) (n int, err error) {
	if b.b == nil {
		b.b = base64.NewDecoder(base64.StdEncoding, b.r)
	}
	n, err = b.b.Read(p)
	if n == 0 && errors.Is(err, io.EOF) {
		b.b = base64.NewDecoder(base64.StdEncoding, b.r)
		n, err = b.b.Read(p)
	}
	return n, err
}

//
// grpc client wrapper
//

type grpcWrapper struct{ c ServiceClient }

func (g grpcWrapper) Method1(ctx context.Context, in *In) (*Out, error) {
	return g.c.Method1(ctx, in)
}
func (g grpcWrapper) Method2(ctx context.Context) (ClientMethod2Stream, error) {
	return g.c.Method2(ctx)
}
func (g grpcWrapper) Method3(ctx context.Context, in *In) (ClientMethod3Stream, error) {
	return g.c.Method3(ctx, in)
}
func (g grpcWrapper) Method4(ctx context.Context) (ClientMethod4Stream, error) {
	return g.c.Method4(ctx)
}

//
// drpc client wrapper
//

type drpcWrapper struct{ c DRPCServiceClient }

func (d drpcWrapper) Method1(ctx context.Context, in *In) (*Out, error) {
	return d.c.Method1(ctx, in)
}
func (d drpcWrapper) Method2(ctx context.Context) (ClientMethod2Stream, error) {
	return d.c.Method2(ctx)
}
func (d drpcWrapper) Method3(ctx context.Context, in *In) (ClientMethod3Stream, error) {
	return d.c.Method3(ctx, in)
}
func (d drpcWrapper) Method4(ctx context.Context) (ClientMethod4Stream, error) {
	return d.c.Method4(ctx)
}

//
// agnostic server impl
//

type serviceImpl struct {
	Method1Fn func(ctx context.Context, in *In) (*Out, error)
	Method2Fn func(stream ServerMethod2Stream) error
	Method3Fn func(in *In, stream ServerMethod3Stream) error
	Method4Fn func(stream ServerMethod4Stream) error
}

type ServerMethod2Stream interface {
	Context() context.Context

	Recv() (*In, error)
	SendAndClose(*Out) error
}

type ServerMethod3Stream interface {
	Context() context.Context

	Send(*Out) error
}

type ServerMethod4Stream interface {
	Context() context.Context

	Send(*Out) error
	Recv() (*In, error)
}

func (i *serviceImpl) DRPC() (d *drpcImpl) {
	d = new(drpcImpl)
	if i.Method1Fn != nil {
		d.Method1Fn = i.Method1Fn
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
		g.Method1Fn = i.Method1Fn
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
	DRPCServiceUnimplementedServer

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
	UnimplementedServiceServer

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
