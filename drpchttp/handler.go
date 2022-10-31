// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package drpchttp

import (
	"fmt"
	"net/http"
	"reflect"

	"storj.io/drpc"
	"storj.io/drpc/drpcerr"
)

// New returns a net/http.Handler that dispatches to the passed in
// drpc.Handler. See NewWithOptions for more details.
func New(handler drpc.Handler) http.Handler {
	return NewWithOptions(handler)
}

// NewWithOptions returns a net/http.Handler that dispatches to the passed in
// drpc.Handler. The RPCs are hosted at a path based on their name, like
// `/service.Server/Method`.
//
// Metadata can be attached by adding the "X-Drpc-Metadata" header to the request
// possibly multiple times. The format is
//
//	X-Drpc-Metadata: percentEncode(key)=percentEncode(value)
//
// where percentEncode is the encoding used for query strings. Only the '%' and '='
// characters are necessary to be escaped.
//
// The specific protocol for the request and response used is chosen by the
// request's Content-Type. By default the content types "application/json" and
// "application/protobuf" correspond to unitary-only RPCs that respond with the
// same Content-Type as the incoming request upon success. Upon failure, the
// response code will not be 200 OK, the response content type will always be
// "application/json", and the body will look something like
//
//	{
//	  "code": "...",
//	  "msg": "..."
//	}
//
// where msg is a textual description of the error, and code is a short string
// that describes the kind of error that happened, if possible. If nothing
// could be detected, then the string "unknown" is used for the code.
//
// The content types "application/grpc-web+proto", "application/grpc-web+json",
// "application/grpc-web-text+proto", and "application/grpc-web-text+json" will
// serve unitary and server-streaming RPCs using the protocol described by
// the grpc-web project. Informally, messages are framed with a 5 byte
// header where the first byte is some flags, and the second through fourth
// are the message length in big endian. Response codes and status messages
// are sent as HTTP Trailers. The "-text" series of content types mean that
// the whole request and response bodies are base64 encoded.
func NewWithOptions(handler drpc.Handler, os ...Option) http.Handler {
	opts := options{protocols: defaultProtocols()}
	for _, o := range os {
		o.apply(&opts)
	}
	return wrapper{
		handler: handler,
		opts:    opts,
	}
}

// wrapper implements net/http.Handler by dispatching to the provided drpc.Handler.
type wrapper struct {
	handler drpc.Handler
	opts    options
}

func (w wrapper) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	pr, ok := w.opts.protocols[req.Header.Get("Content-Type")]
	if !ok {
		pr = w.opts.protocols["*"]
	}

	ctx, err := Context(req)
	if err == nil {
		req = req.WithContext(ctx)
	}

	st := pr.NewStream(rw, req)
	st.Finish(w.handler.HandleRPC(st, req.URL.Path))
}

// getCode returns a string code for the provided error, or "unknown" if it
// cannot find one. It uses reflect to pull Twirp codes out of the error
// without having to import and depend on the Twirp module.
func getCode(err error) string {
	code := "unknown"
	if dcode := drpcerr.Code(err); dcode != 0 {
		code = fmt.Sprintf("drpcerr(%d)", dcode)
	}
	for i := 0; i < 100; i++ {
		if m := reflect.ValueOf(err).MethodByName("Code"); m.IsValid() {
			if mt := m.Type(); mt.NumIn() == 0 && mt.NumOut() == 1 &&
				mt.Out(0).Kind() == reflect.String {
				return m.Call(nil)[0].String()
			}
		}
		switch v := err.(type) { //nolint: errorlint
		case interface{ Cause() error }:
			err = v.Cause()
		case interface{ Unwrap() error }:
			err = v.Unwrap()
		default:
			return code
		}
	}
	return code
}
