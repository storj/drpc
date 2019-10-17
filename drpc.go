// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpc

import (
	"context"
	"io"
	"time"

	"github.com/zeebo/errs"
)

var (
	Error         = errs.Class("drpc")
	InternalError = errs.Class("internal error")
	ProtocolError = errs.Class("protocol error")
)

type Transport interface {
	io.Reader
	io.Writer
	io.Closer

	// SetReadDeadline sets the deadline for future Read calls
	// and any currently-blocked Read call.
	// A zero value for t means Read will not time out.
	SetReadDeadline(t time.Time) error

	// SetWriteDeadline sets the deadline for future Write calls
	// and any currently-blocked Write call.
	// Even if write times out, it may return n > 0, indicating that
	// some of the data was successfully written.
	// A zero value for t means Write will not time out.
	SetWriteDeadline(t time.Time) error
}

type Message interface {
	Reset()
	String() string
	ProtoMessage()
}

type Conn interface {
	Close() error
	Transport() Transport

	Invoke(ctx context.Context, rpc string, in, out Message) error
	NewStream(ctx context.Context, rpc string) (Stream, error)
}

type Stream interface {
	Context() context.Context

	MsgSend(msg Message) error
	MsgRecv(msg Message) error

	CloseSend() error
	Close() error
}

type Handler = func(srv interface{}, ctx context.Context, in1, in2 interface{}) (out Message, err error)

type Description interface {
	NumMethods() int
	Method(n int) (rpc string, handler Handler, method interface{}, ok bool)
}

type Server interface {
	Register(srv interface{}, desc Description)
}
