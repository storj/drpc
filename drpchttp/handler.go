// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package drpchttp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/zeebo/errs"

	"storj.io/drpc"
	"storj.io/drpc/drpcerr"
)

// New returns a net/http.Handler that dispatches to the passed in drpc.Handler.
//
// The returned value handles unitary RPCs over an http request. The RPCs are hosted
// at a path based on their name, like `/service.Server/Method` and accept the request
// message in JSON or protobuf, depending on if the requested Content-Type is equal to
// "application/json" or "application/protobuf", respectively. If the response was
// a success, the HTTP status code will be 200 OK, and the body contains the message
// encoded the same way as the request. If there was an error, the HTTP status code
// will not be 200 OK, the response body is always JSON, and will look something
// like
//
//    {
//      "code": "...",
//      "msg": "..."
//    }
//
// where msg is a textual description of the error, and code is a short string that
// describes the kind of error that happened, if possible. If nothing could be
// detected, then the string "unknown" is used for the code.
//
// Metadata can be attached by adding the "X-Drpc-Metadata" header to the request
// possibly multiple times. The format is
//
//     X-Drpc-Metadata: percentEncode(key)=percentEncode(value)
//
// where percentEncode is the encoding used for query strings. Only the '%' and '='
// characters are necessary to be escaped.
func New(handler drpc.Handler, os ...Option) http.Handler {
	opts := options{
		codeMapper: defaultCodeMapper,
	}
	for _, o := range os {
		o.apply(&opts)
	}

	return wrapper{
		handler: handler,
		opts:    opts,
	}
}

// Option configures some aspect of the handler.
type Option struct{ apply func(*options) }

// WithCodeMapper sets the function that will be called when the rpc handler
// returns an error to map the error to the json code field. For example,
// to map Twirp errors back to their appropriate code, you could write
//
// 	func twirpMapper(err error) string {
// 		var te twirp.Error
// 		if errors.As(err, &te) {
// 			return string(te.Code())
// 		}
// 		return "unknown"
// 	}
//
// and use it with
//
// 	handler := drpchttp.New(mux, drpchttp.WithCodeMapper(twirpMapper))
func WithCodeMapper(mapper func(error) string) Option {
	return Option{apply: func(opts *options) { opts.codeMapper = mapper }}
}

func defaultCodeMapper(err error) string {
	code := "unknown"
	if dcode := drpcerr.Code(err); dcode != 0 {
		code = fmt.Sprintf("drpcerr(%d)", dcode)
	}
	return code
}

type options struct {
	codeMapper func(error) string
}

// wrapper implements net/http.Handler by dispatching to the provided drpc.Handler.
type wrapper struct {
	handler drpc.Handler
	opts    options
}

func (w wrapper) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	data, serr := w.serveHTTP(req)
	if serr != nil {
		http.Error(rw, serr.JSON(), serr.status)
		return
	}
	rw.Header().Set("Content-Type", rw.Header().Get("Content-Type"))
	_, _ = rw.Write(data)
}

func (w wrapper) serveHTTP(req *http.Request) (data []byte, serr *statusErr) {
	ctx, err := Context(req)
	if err != nil {
		return nil, w.wrapStatusErr(http.StatusInternalServerError, err)
	}

	ct := req.Header.Get("Content-Type")
	switch ct {
	case "application/protobuf":
	case "application/json":
	default:
		return nil, w.newStatusErr(http.StatusUnsupportedMediaType, "invalid content type: %q", ct)
	}

	const maxSize = 4 << 20
	data, err = ioutil.ReadAll(io.LimitReader(req.Body, maxSize))
	if err != nil {
		return nil, w.wrapStatusErr(http.StatusInternalServerError, err)
	} else if len(data) >= maxSize {
		return nil, w.newStatusErr(http.StatusInternalServerError, "message size limit exceeded")
	}

	stream := &unitaryStream{
		ctx:  ctx,
		json: ct == "application/json",
		in:   data,
	}

	if err := w.handler.HandleRPC(stream, req.URL.Path); err != nil {
		return nil, w.wrapStatusErr(http.StatusInternalServerError, err)
	}

	return stream.out, nil
}

type unitaryStream struct {
	ctx  context.Context
	json bool
	in   []byte
	out  []byte
}

func (us *unitaryStream) Context() context.Context { return us.ctx }
func (us *unitaryStream) CloseSend() error         { return nil }
func (us *unitaryStream) Close() error             { return nil }

func (us *unitaryStream) MsgSend(msg drpc.Message, enc drpc.Encoding) (err error) {
	if us.out != nil {
		return io.EOF
	}
	if us.json {
		us.out, err = JSONMarshal(msg, enc)
	} else {
		us.out, err = enc.Marshal(msg)
	}
	return err
}

func (us *unitaryStream) MsgRecv(msg drpc.Message, enc drpc.Encoding) (err error) {
	if us.in == nil {
		return io.EOF
	}
	if us.json {
		us.in, err = nil, JSONUnmarshal(us.in, msg, enc)
	} else {
		us.in, err = nil, enc.Unmarshal(us.in, msg)
	}
	return err
}

type statusErr struct {
	status int
	code   string
	err    error
}

func (w wrapper) newStatusErr(status int, format string, args ...interface{}) *statusErr {
	return w.wrapStatusErr(status, errs.New(format, args...))
}

func (w wrapper) wrapStatusErr(status int, err error) *statusErr {
	return &statusErr{
		status: status,
		code:   w.opts.codeMapper(err),
		err:    err,
	}
}

func (s *statusErr) Error() string { return s.err.Error() }
func (s *statusErr) Cause() error  { return s.err }
func (s *statusErr) Unwrap() error { return s.err }

func (s *statusErr) JSON() string {
	data, _ := json.MarshalIndent(map[string]interface{}{
		"code": s.code,
		"msg":  s.err.Error(),
	}, "", "    ")
	return string(data)
}
