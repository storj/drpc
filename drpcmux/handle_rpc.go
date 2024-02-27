// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcmux

import (
	"reflect"

	"github.com/zeebo/errs"

	"storj.io/drpc"
)

// HandleRPC handles the rpc that has been requested by the stream.
func (m *Mux) HandleRPC(stream drpc.Stream, rpc string) (err error) {
	data, ok := m.rpcs[rpc]
	if !ok {
		return drpc.ProtocolError.New("unknown rpc: %q", rpc)
	}

	var (
		in  = interface{}(stream)
		msg drpc.Message
	)
	if data.in1 != streamType {
		// If its an vtprotobuf supported message type
		if data.in1.Implements(vtMessageType) {
			p := reflect.New(data.in1.Elem()).Interface().(drpc.VTProtoMessage).FromVTPool()
			if p == nil {
				return drpc.InternalError.New("unable to get message from vt pool")
			}
			msg, ok = p.(drpc.Message)
			if !ok {
				return drpc.InternalError.New("invalid rpc input type")
			}

			defer p.(drpc.VTProtoMessage).ReturnToVTPool()
		} else {
			m, ok := reflect.New(data.in1.Elem()).Interface().(drpc.Message)
			if !ok {
				return drpc.InternalError.New("invalid rpc input type")
			}
			msg = m
		}
		if err := stream.MsgRecv(msg, data.enc); err != nil {
			return errs.Wrap(err)
		}
		in = msg
	}

	out, err := data.receiver(data.srv, stream.Context(), in, stream)
	switch {
	case err != nil:
		return errs.Wrap(err)
	case out != nil && !reflect.ValueOf(out).IsNil():
		return stream.MsgSend(out, data.enc)
	default:
		return stream.CloseSend()
	}
}
