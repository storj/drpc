// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcconn

import (
	"context"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/zeebo/errs"

	"storj.io/drpc"
	"storj.io/drpc/drpcstream"
	"storj.io/drpc/drpcutil"
	"storj.io/drpc/drpcwire"
)

type Conn struct {
	tr *drpcwire.Transport

	once  sync.Once
	done  *drpcutil.Signal
	start chan struct{}
	stop  chan struct{}
}

var _ drpc.Conn = (*Conn)(nil)

func New(tr drpc.Transport) *Conn {
	return &Conn{
		tr:    drpcwire.NewTransport(tr),
		done:  drpcutil.NewSignal(),
		start: make(chan struct{}, 1),
		stop:  make(chan struct{}),
	}
}

func (c *Conn) Transport() drpc.Transport {
	return c.tr
}

func (c *Conn) Close() (err error) {
	c.once.Do(func() {
		c.done.Set(drpc.Error.New("conn closed"))
		err = c.tr.Close()
		c.start <- struct{}{}
	})
	return err
}

func (c *Conn) acquireSemaphore(ctx context.Context) (err error) {
	select {
	case <-c.done.Signal():
		return c.done.Err()
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	select {
	case <-c.done.Signal():
		return c.done.Err()
	case <-ctx.Done():
		return ctx.Err()
	case c.start <- struct{}{}:
		return nil
	}
}

func (c *Conn) manageStream(ctx context.Context, stream *drpcstream.Stream) {
	queueClosed := false

	for {
		switch pkt, err := c.tr.ReadPacket(); {

		// terminal conditions
		case c.done.IsSet():
			stream.DoneSig().Set(c.done.Err())
		case err != nil:
			stream.DoneSig().Set(err)
		case pkt.Kind == drpcwire.PacketKind_Error:
			stream.DoneSig().Set(errs.New("%s", pkt.Data))
		case pkt.Kind == drpcwire.PacketKind_Cancel:
			stream.DoneSig().Set(context.Canceled)
		case pkt.Kind == drpcwire.PacketKind_Invoke:
			stream.DoneSig().Set(drpc.ProtocolError.New("invalid invoke sent to client"))
		case pkt.Kind == drpcwire.PacketKind_Close:
			stream.DoneSig().Set(drpc.Error.New("remote closed stream"))
		case pkt.Kind == drpcwire.PacketKind_Message && queueClosed:
			stream.DoneSig().Set(drpc.ProtocolError.New("message send after SendClose"))

		// CloseSend means the queue is done with packets
		case pkt.Kind == drpcwire.PacketKind_CloseSend:
			close(stream.Queue())
			queueClosed = true
			continue

		// Message means we try to send it into the stream's receive queue
		case pkt.Kind == drpcwire.PacketKind_Message:
			select {
			case <-c.done.Signal():
				stream.DoneSig().Set(c.done.Err())
			case <-ctx.Done():
				_ = stream.SendCancel()
			default:
				select {
				case <-c.done.Signal():
					stream.DoneSig().Set(c.done.Err())
				case <-ctx.Done():
					_ = stream.SendCancel()
				case stream.Queue() <- pkt:
					continue
				}
			}
		}

		if !queueClosed {
			close(stream.Queue())
		}
		c.stop <- struct{}{}
		<-c.start
		return
	}
}

func (c *Conn) manageContext(ctx context.Context) {
	select {
	case <-c.stop:
	case <-ctx.Done():
		_ = c.Close()
		<-c.stop
	}
}

func (c *Conn) Invoke(ctx context.Context, rpc string, in, out drpc.Message) (err error) {
	if err := c.acquireSemaphore(ctx); err != nil {
		return err
	}

	data, err := proto.Marshal(in)
	if err != nil {
		return err
	}

	stream := drpcstream.New(ctx, c.tr.Writer)
	go c.manageStream(ctx, stream)
	go c.manageContext(ctx)

	if err := stream.RawWrite(drpcwire.PacketKind_Invoke, []byte(rpc)); err != nil {
		return errs.Combine(err, c.Close())
	}
	if err := stream.RawWrite(drpcwire.PacketKind_Message, data); err != nil {
		return errs.Combine(err, c.Close())
	}
	if err := stream.CloseSend(); err != nil {
		return errs.Combine(err, c.Close())
	}
	if err := stream.MsgRecv(out); err != nil {
		return errs.Combine(err, c.Close())
	}
	if err := stream.Close(); err != nil {
		return errs.Combine(err, c.Close())
	}
	return nil
}

func (c *Conn) NewStream(ctx context.Context, rpc string) (_ drpc.Stream, err error) {
	if err := c.acquireSemaphore(ctx); err != nil {
		return nil, err
	}

	stream := drpcstream.New(ctx, c.tr.Writer)
	go c.manageStream(ctx, stream)
	go c.manageContext(ctx)

	if err := stream.RawWrite(drpcwire.PacketKind_Invoke, []byte(rpc)); err != nil {
		return nil, errs.Combine(err, c.Close())
	}
	if err := stream.RawFlush(); err != nil {
		return nil, errs.Combine(err, c.Close())
	}
	return stream, nil
}
