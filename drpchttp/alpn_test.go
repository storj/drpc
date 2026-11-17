package drpchttp

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"storj.io/drpc"
	"storj.io/drpc/drpcconn"
	"storj.io/drpc/drpcmux"
	"storj.io/drpc/drpcserver"
	"storj.io/drpc/drpctest"
)

func TestALPN(t *testing.T) {
	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	// Set up a DPRC server:
	//
	// A real server would obviously register services.
	dsrv := drpcserver.New(drpcmux.New())

	// Create a TLS config:
	//
	// A real server would add "h2", etc
	cfg := &tls.Config{
		NextProtos: []string{"http/1.1"},
	}
	// Test the function actually modifies the NextProtos:
	cfg, err := ConfigureTLS(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(cfg.NextProtos, TLSNextProto) {
		t.Errorf("NextProtos (%v) does not contain %#q", cfg.NextProtos, TLSNextProto)
	}
	t.Logf("server tls.Config NextProtos: %v", cfg.NextProtos)

	// Create a test HTTP server.
	srv := httptest.NewUnstartedServer(http.HandlerFunc(nil))

	srv.TLS = cfg
	srv.Config.BaseContext = func(_ net.Listener) context.Context { return ctx }
	// Configure other protocol hooks to fail the test if called.
	srv.Config.TLSNextProto = map[string]func(*http.Server, *tls.Conn, http.Handler){
		"":         func(_ *http.Server, _ *tls.Conn, _ http.Handler) { t.Error("got non-ALPN request") },
		"http/1.1": func(_ *http.Server, _ *tls.Conn, _ http.Handler) { t.Error("got http/1.1 request") },
	}
	// Test the configuration function actually modifies the TLSNextProto map:
	if err := ConfigureNextProto(srv.Config, dsrv, nil); err != nil {
		t.Fatal(err)
	}
	if _, ok := srv.Config.TLSNextProto[TLSNextProto]; !ok {
		t.Error("protocol hook not set")
	}

	srv.StartTLS()
	t.Cleanup(srv.Close)

	// Do a bunch of client setup:
	//
	// The server setup does not add the server's root CA (it adds it to a
	// created [http.Transport]), so we must do it manually.
	addr := srv.Listener.Addr()
	roots := x509.NewCertPool()
	roots.AddCert(srv.Certificate())
	clCfg := &tls.Config{
		RootCAs: roots,
	}
	clCfg, err = ConfigureTLS(clCfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("config tls.Config NextProtos: %v", clCfg.NextProtos)
	td := tls.Dialer{
		NetDialer: &net.Dialer{},
		Config:    clCfg,
	}
	// Open a TLS connection.
	conn, err := td.DialContext(ctx, addr.Network(), addr.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	})

	// Now create a DRPC connection over the TLS connection and call a
	// nonexistent endpoint.
	c := drpcconn.New(conn)
	err = c.Invoke(ctx, "/", new(bogusEncoding), new(bogusMsg), nil)
	if got, want := err.Error(), `unknown rpc: "/"`; !strings.Contains(got, want) {
		t.Errorf("got: %#q, want: %#q", got, want)
	}

	// Check that the proper protocol was used. Must do this after the [Invoke]
	// call because the TLS handshake completes on the first read or write.
	t.Logf("negotiated protocol: %q", conn.(*tls.Conn).ConnectionState().NegotiatedProtocol)
}

type bogusMsg struct {
	OK bool
}

type bogusEncoding struct{}

func (b *bogusEncoding) Marshal(msg drpc.Message) ([]byte, error) { return json.Marshal(msg) }
func (b *bogusEncoding) Unmarshal(buf []byte, msg drpc.Message) error {
	return json.Unmarshal(buf, msg)
}

func ExampleConfigureNextProto() {
	// Set up the HTTP server. The ALPN support uses the server's accept loop.
	hSrv := new(http.Server)
	hSrv.TLSNextProto = make(map[string]func(*http.Server, *tls.Conn, http.Handler))
	// A production server may want to explicitly enable HTTP/2:
	/*
		h2Srv := new(http2.Server)
		// Configure Handler, etc...
		http2.ConfigureServer(hSrv, h2Srv)
	*/

	// Set up the DRPC server.
	dSrv := drpcserver.New(nil)

	ConfigureNextProto(hSrv, dSrv, context.TODO())

	hSrv.ListenAndServeTLS("cert.pem", "key.pem")
}

func ExampleConfigureTLS_client() {
	// Set up the TLS config.
	cfg, _ := ConfigureTLS(new(tls.Config))
	dialer := tls.Dialer{Config: cfg}

	// Open a TLS connection.
	conn, _ := dialer.DialContext(context.TODO(), "tcp", "[::]:https")
	defer conn.Close()

	// Now, create a DRPC connection over the TLS connection.
	c := drpcconn.New(conn)
	c.Close()
}
