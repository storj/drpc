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
func New(handler drpc.Handler) http.Handler {
	return wrapper{handler: handler}
}

// wrapper implements net/http.Handler by dispatching to the provided drpc.Handler.
type wrapper struct {
	handler drpc.Handler
}

func (h wrapper) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	data, serr := h.serveHTTP(req)
	if serr != nil {
		http.Error(w, serr.JSON(), serr.status)
		return
	}
	w.Header().Set("Content-Type", w.Header().Get("Content-Type"))
	_, _ = w.Write(data)
}

func (h wrapper) serveHTTP(req *http.Request) (data []byte, serr *statusErr) {
	ctx, err := Context(req)
	if err != nil {
		return nil, wrapStatusErr(http.StatusInternalServerError, err)
	}

	ct := req.Header.Get("Content-Type")
	switch ct {
	case "application/protobuf":
	case "application/json":
	default:
		return nil, newStatusErr(http.StatusUnsupportedMediaType, "invalid content type: %q", ct)
	}

	const maxSize = 4 << 20
	data, err = ioutil.ReadAll(io.LimitReader(req.Body, maxSize))
	if err != nil {
		return nil, wrapStatusErr(http.StatusInternalServerError, err)
	} else if len(data) >= maxSize {
		return nil, newStatusErr(http.StatusInternalServerError, "message size limit exceeded")
	}

	stream := &unitaryStream{
		ctx:  ctx,
		json: ct == "application/json",
		in:   data,
	}

	if err := h.handler.HandleRPC(stream, req.URL.Path); err != nil {
		return nil, wrapStatusErr(http.StatusInternalServerError, err)
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
	err    error
}

func newStatusErr(status int, format string, args ...interface{}) *statusErr {
	return wrapStatusErr(status, errs.New(format, args...))
}

func wrapStatusErr(status int, err error) *statusErr {
	return &statusErr{status: status, err: err}
}

func (s *statusErr) Error() string { return s.err.Error() }
func (s *statusErr) Cause() error  { return s.err }
func (s *statusErr) Unwrap() error { return s.err }

func (s *statusErr) JSON() string {
	code := "unknown"
	if dcode := drpcerr.Code(s.err); dcode != 0 {
		code = fmt.Sprintf("drpcerr(%d)", dcode)
	}
	data, _ := json.MarshalIndent(map[string]interface{}{
		"code": code,
		"msg":  s.err.Error(),
	}, "", "    ")
	return string(data)
}
