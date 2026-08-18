package drpchttp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"

	"storj.io/drpc/drpcserver"
)

// TLSNextProto the ALPN protocol ID used for switching to DRPC.
const TLSNextProto = `drpc/0`

// ErrNextProtosUnconfigured is returned from [ConfigureNextProto] when an
// [http.Server] is not explicitly configured for protocol negotiation.
var ErrNextProtosUnconfigured = errors.New("drpchttp: (*http.Server).TLSNextProto not populated; doing nothing")

// ConfigureNextProto adds a "next protocol" handler to the passed [http.Server]
// that dispatches connections to the passed [drpcserver.Server]. The "fallback"
// [context.Context] is used for connections if a suitable Context cannot be
// derived from the [http] interface. If nil is passed, [context.Background]
// will be used.
//
// This function is only effective if the [http.Server] is serving over a TLS
// connection. If [http.Server.TLSNextProto] is not populated,
// [ErrNextProtosUnconfigured] will be reported. This is done to avoid
// accidentally disabling HTTP/2 support, which is only enabled by default if
// TLSNextProto is not populated. See [golang.org/x/net/http2] for explicit
// HTTP/2 configuration.
//
// If [http.Server.TLSConfig] is populated, [ConfigureTLS] is called
// automatically. Note that it's only used if the [http.Server.ServeTLS] or
// [http.Server.ListenAndServeTLS] methods are used.
func ConfigureNextProto(hs *http.Server, srv *drpcserver.Server, fallback context.Context) error {
	const errPrefix = `drpchttp: can't setup ALPN: `
	switch {
	case hs == nil:
		return errors.New(errPrefix + "nil http.Server")
	case srv == nil:
		return errors.New(errPrefix + "nil drpcserver.Server")
	case hs.TLSNextProto == nil:
		return ErrNextProtosUnconfigured
	}
	if fallback == nil {
		fallback = context.Background()
	}

	// This is patterned on the go http2 package.
	//
	// This handler ignores the passed http.Handler argument and instead hijacks
	// the Connection and hands it to the DRPC server.

	if cfg := hs.TLSConfig; cfg != nil {
		var err error
		hs.TLSConfig, err = ConfigureTLS(cfg)
		if err != nil {
			return fmt.Errorf(errPrefix+"%w", err)
		}
	}

	protoHandler := func(s *http.Server, c *tls.Conn, h http.Handler) {
		// According to a comment in x/net/http2, there's an unadvertised method
		// on the Handler implementation that returns the Context. Technically
		// an internal detail, but use it if we can.
		var ctx context.Context
		type baseContexter interface {
			BaseContext() context.Context
		}
		switch bc, ok := h.(baseContexter); {
		case ok:
			ctx = bc.BaseContext()
		default:
			ctx = fallback
		}

		// Dunno if there's a better place or way to get a logger or something
		// that can handle a returned error.
		log := s.ErrorLog

		if err := srv.ServeOne(ctx, c); err != nil && log != nil {
			log.Printf("drpc error: %v", err)
		}
	}
	hs.TLSNextProto[TLSNextProto] = protoHandler

	return nil
}

// ConfigureTLS returns a copy of the passed [tls.Config] modified to enable
// DRPC as a negotiated protocol.
//
// This is needed for client configurations and server configurations that do
// not use [http.Server.ServeTLS].
func ConfigureTLS(cfg *tls.Config) (*tls.Config, error) {
	// Should this just modify the passed-in config?
	n := cfg.Clone()
	n.NextProtos = append(n.NextProtos, TLSNextProto)
	return n, nil
}
