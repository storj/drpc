// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package drpchttp

import (
	"context"
	"encoding/json"
	"io"

	"storj.io/drpc"
)

type unitaryStream struct {
	ctx context.Context
	in  []byte
	out []byte
}

func (us *unitaryStream) Context() context.Context { return us.ctx }
func (us *unitaryStream) CloseSend() error         { return nil }
func (us *unitaryStream) Close() error             { return nil }

func (us *unitaryStream) MsgSend(msg drpc.Message, enc drpc.Encoding) (err error) {
	if us.out != nil {
		return io.EOF
	}
	us.out, err = JSONMarshal(msg, enc)
	return err
}

func (us *unitaryStream) MsgRecv(msg drpc.Message, enc drpc.Encoding) (err error) {
	if us.in == nil {
		return io.EOF
	}
	us.in, err = nil, JSONUnmarshal(msg, enc, us.in)
	return err
}

// JSONMarshal looks for a JSONMarshal method on the encoding and calls that if it
// exists. Otherwise, it does a normal message marshal before doing a JSON marshal.
func JSONMarshal(msg drpc.Message, enc drpc.Encoding) ([]byte, error) {
	if enc, ok := enc.(interface {
		JSONMarshal(msg drpc.Message) ([]byte, error)
	}); ok {
		return enc.JSONMarshal(msg)
	}

	// fallback to normal Marshal + JSON Marshal
	buf, err := enc.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return json.Marshal(buf)
}

// JSONUnmarshal looks for a JSONUnmarshal method on the encoding and calls that
// if it exists. Otherwise, it JSON unmarshals the buf before doing a normal
// message unmarshal.
func JSONUnmarshal(msg drpc.Message, enc drpc.Encoding, buf []byte) error {
	if enc, ok := enc.(interface {
		JSONUnmarshal(buf []byte, msg drpc.Message) error
	}); ok {
		return enc.JSONUnmarshal(buf, msg)
	}

	// fallback to JSON Unmarshal + normal Unmarshal
	var data []byte
	if err := json.Unmarshal(buf, &data); err != nil {
		return err
	}
	return enc.Unmarshal(data, msg)
}
