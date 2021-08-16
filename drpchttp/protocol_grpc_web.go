// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package drpchttp

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"

	"github.com/zeebo/errs"

	"storj.io/drpc"
	"storj.io/drpc/drpcerr"
)

//
// protocol handler
//

type grpcWebProtocol struct {
	ct        string
	read      readFunc
	write     writeFunc
	marshal   marshalFunc
	unmarshal unmarshalFunc
}

func (gwp grpcWebProtocol) NewStream(rw http.ResponseWriter, req *http.Request) Stream {
	rw.Header().Set("Content-Type", gwp.ct)
	return &grpcWebStream{
		ctx: req.Context(),
		gwp: gwp,
		in:  req.Body,
		rw:  rw,
	}
}

func (gwp grpcWebProtocol) framedWrite(rw http.ResponseWriter, hdr byte, buf []byte) error {
	tmp := [5]byte{0: hdr}
	binary.BigEndian.PutUint32(tmp[1:5], uint32(len(buf)))
	return gwp.write(rw, append(tmp[:], buf...))
}

//
// stream type
//

type grpcWebStream struct {
	ctx context.Context
	gwp grpcWebProtocol
	in  io.ReadCloser
	rw  http.ResponseWriter
}

func (gws *grpcWebStream) Context() context.Context { return gws.ctx }
func (gws *grpcWebStream) CloseSend() error         { return nil }
func (gws *grpcWebStream) Close() error             { return gws.in.Close() }

func (gws *grpcWebStream) MsgSend(msg drpc.Message, enc drpc.Encoding) (err error) {
	data, err := gws.gwp.marshal(msg, enc)
	if err != nil {
		return err
	} else if len(data) >= maxSize {
		return errs.New("message too large")
	} else if err := gws.gwp.framedWrite(gws.rw, 0, data); err != nil {
		return err
	} else if fl, ok := gws.rw.(http.Flusher); ok {
		fl.Flush()
	}
	return nil
}

func (gws *grpcWebStream) MsgRecv(msg drpc.Message, enc drpc.Encoding) (err error) {
	buf, err := gws.gwp.read(gws.in)
	if err != nil {
		return err
	}
	return gws.gwp.unmarshal(buf, msg, enc)
}

var nlSpace = strings.NewReplacer("\n", " ", "\r", " ")

func (gws *grpcWebStream) Finish(err error) {
	// if there is an error and the code is "0" (Ok) either
	// because it is unset or explicitly set to 0, then set it
	// to "2" (Unknown) so that an error status is sent instead.
	status := strconv.FormatUint(drpcerr.Code(err), 10)
	if err != nil && status == "0" {
		status = "2"
	}

	var buf bytes.Buffer
	write := func(k, v string) {
		buf.WriteString(k)
		buf.WriteString(": ")
		buf.WriteString(textproto.TrimString(nlSpace.Replace(v)))
		buf.WriteString("\r\n")
	}

	write("grpc-status", status)
	if err != nil {
		write("grpc-code", getCode(err))
		write("grpc-message", err.Error())
	}

	_ = gws.gwp.framedWrite(gws.rw, 128, buf.Bytes())
}
