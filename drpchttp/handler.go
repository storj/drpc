// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package drpchttp

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"storj.io/drpc"
	"storj.io/drpc/drpcerr"
)

// New returns a net/http.Handler that dispatches to the passed in drpc.Handler.
//
// The returned value handles unitary RPCs over an http request. The RPCs are hosted
// at a path based on their name, like `/service.Server/Method` and accept the request
// message in JSON. The response will either be of the form
//
//    {
//      "status": "ok",
//      "response": ...
//    }
//
// if the request was successful, or
//
//    {
//      "status": "error",
//      "error": ...,
//      "code": ...
//    }
//
// where error is a textual description of the error, and code is the numeric code
// that was set with drpcerr, if any.
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
	ctx, err := Context(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data, err := h.serveHTTP(ctx, req.URL.Path, req.Body)
	if err != nil {
		data, err = json.MarshalIndent(map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
			"code":   drpcerr.Code(err),
		}, "", "  ")
	} else {
		data, err = json.MarshalIndent(map[string]interface{}{
			"status":   "ok",
			"response": json.RawMessage(data),
		}, "", " ")
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}

func (h wrapper) serveHTTP(ctx context.Context, rpc string, body io.Reader) ([]byte, error) {
	const maxSize = 4 << 20
	data, err := ioutil.ReadAll(io.LimitReader(body, maxSize))
	if err != nil {
		return nil, err
	} else if len(data) >= maxSize {
		return nil, drpc.ProtocolError.New("incoming message size limit exceeded")
	}

	stream := &unitaryStream{
		ctx: ctx,
		in:  data,
	}

	if err := h.handler.HandleRPC(stream, rpc); err != nil {
		return nil, err
	}
	return stream.out, nil
}
