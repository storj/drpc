// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcconn

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/zeebo/errs"

	"storj.io/drpc"
	"storj.io/drpc/drpcmanager"
	"storj.io/drpc/drpcstream"
	"storj.io/drpc/drpcwire"
)

var connClosed = drpc.Error.New("conn closed")

type Conn struct {
	tr  drpc.Transport
	man *drpcmanager.Manager
}

var _ drpc.Conn = (*Conn)(nil)

func New(tr drpc.Transport) *Conn {
	return &Conn{
		tr:  tr,
		man: drpcmanager.New(tr, nil),
	}
}

func (c *Conn) Transport() drpc.Transport {
	return c.tr
}

func (c *Conn) Close() (err error) {
	return c.man.Close()
}

func (c *Conn) Invoke(ctx context.Context, rpc string, in, out drpc.Message) (err error) {
	data, err := proto.Marshal(in)
	if err != nil {
		return errs.Wrap(err)
	}
	stream, err := c.man.NewStream(ctx, 0)
	if err != nil {
		return errs.Wrap(err)
	}
	var eg errs.Group
	eg.Add(c.doInvoke(ctx, stream, []byte(rpc), data, out))
	eg.Add(stream.Close())
	return errs.Wrap(eg.Err())
}

func (c *Conn) doInvoke(ctx context.Context, stream *drpcstream.Stream, rpc, data []byte, out drpc.Message) (err error) {
	if err := stream.RawWrite(drpcwire.Kind_Invoke, []byte(rpc)); err != nil {
		return errs.Wrap(err)
	}
	if err := stream.RawWrite(drpcwire.Kind_Message, data); err != nil {
		return errs.Wrap(err)
	}
	if err := stream.CloseSend(); err != nil {
		return errs.Wrap(err)
	}
	if err := stream.MsgRecv(out); err != nil {
		return errs.Wrap(err)
	}
	return nil
}

func (c *Conn) NewStream(ctx context.Context, rpc string) (_ drpc.Stream, err error) {
	stream, err := c.man.NewStream(ctx, 0)
	if err != nil {
		return nil, errs.Wrap(err)
	}
	err = c.doNewStream(ctx, stream, []byte(rpc))
	if err != nil {
		err = errs.Combine(err, stream.Close())
		return nil, errs.Wrap(err)
	}
	return stream, nil
}

func (c *Conn) doNewStream(ctx context.Context, stream *drpcstream.Stream, rpc []byte) error {
	if err := stream.RawWrite(drpcwire.Kind_Invoke, []byte(rpc)); err != nil {
		return errs.Wrap(err)
	}
	if err := stream.RawFlush(); err != nil {
		return errs.Wrap(err)
	}
	return nil
}
