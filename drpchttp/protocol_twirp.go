// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package drpchttp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"storj.io/drpc"
)

//
// protocol handler
//

type twirpProtocol struct {
	ct        string
	marshal   marshalFunc
	unmarshal unmarshalFunc
}

func (tp twirpProtocol) NewStream(rw http.ResponseWriter, req *http.Request) Stream {
	rw.Header().Set("Content-Type", tp.ct)
	return &twirpStream{
		ctx:  req.Context(),
		tp:   tp,
		body: req.Body,
		rw:   rw,
	}
}

//
// stream type
//

type twirpStream struct {
	ctx  context.Context
	tp   twirpProtocol
	body io.ReadCloser
	rw   http.ResponseWriter

	response []byte
	recvErr  error
	sendErr  error
}

func setErrorOrEOF(errp *error, err error) {
	if err == nil {
		err = io.EOF
	}
	*errp = err
}

func (ts *twirpStream) Context() context.Context { return ts.ctx }
func (ts *twirpStream) CloseSend() error         { return nil }
func (ts *twirpStream) Close() error             { return nil }

func (ts *twirpStream) MsgSend(msg drpc.Message, enc drpc.Encoding) (err error) {
	if ts.sendErr != nil {
		return ts.sendErr
	}
	ts.response, err = ts.tp.marshal(msg, enc)
	setErrorOrEOF(&ts.sendErr, err)
	return err
}

func (ts *twirpStream) MsgRecv(msg drpc.Message, enc drpc.Encoding) (err error) {
	if ts.recvErr != nil {
		return ts.recvErr
	}
	buf, err := twirpRead(ts.body)
	setErrorOrEOF(&ts.recvErr, err)
	if err != nil {
		return err
	}
	return ts.tp.unmarshal(buf, msg, enc)
}

func (ts *twirpStream) Finish(err error) {
	if err == nil {
		ts.rw.WriteHeader(http.StatusOK)
		_, _ = ts.rw.Write(ts.response)
		return
	}

	code := getCode(err)
	status := twirpStatus[code]
	if status == 0 {
		status = 500
	}

	data, err := json.MarshalIndent(map[string]interface{}{
		"code": code,
		"msg":  err.Error(),
	}, "", "    ")
	if err != nil {
		http.Error(ts.rw, "", http.StatusInternalServerError)
		return
	}

	ts.rw.Header().Set("Content-Type", "application/json")
	ts.rw.WriteHeader(status)
	_, _ = ts.rw.Write(data)
}

var twirpStatus = map[string]int{
	"canceled":            408,
	"unknown":             500,
	"invalid_argument":    400,
	"malformed":           400,
	"deadline_exceeded":   408,
	"not_found":           404,
	"bad_route":           404,
	"already_exists":      409,
	"permission_denied":   403,
	"unauthenticated":     401,
	"resource_exhausted":  429,
	"failed_precondition": 412,
	"aborted":             409,
	"out_of_range":        400,
	"unimplemented":       501,
	"internal":            500,
	"unavailable":         503,
	"dataloss":            500,
}
