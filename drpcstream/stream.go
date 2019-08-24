// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcstream

import (
	"context"
	"io"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/zeebo/errs"

	"storj.io/drpc"
	"storj.io/drpc/drpcutil"
	"storj.io/drpc/drpcwire"
)

type Stream struct {
	ctx context.Context
	wr  *drpcwire.Writer

	mu    sync.Mutex
	done  *drpcutil.Signal
	send  *drpcutil.Signal
	queue chan drpcwire.Packet

	// avoids allocations of closures
	pollWriteFn func(drpcwire.Frame) error
}

var _ drpc.Stream = (*Stream)(nil)

func New(ctx context.Context, wr *drpcwire.Writer) *Stream {
	s := &Stream{
		ctx: ctx,
		wr:  wr,

		done:  drpcutil.NewSignal(),
		send:  drpcutil.NewSignal(),
		queue: make(chan drpcwire.Packet),
	}
	s.pollWriteFn = s.pollWrite
	return s
}

//
// accessors
//

func (s *Stream) Context() context.Context { return s.ctx }

func (s *Stream) DoneSig() *drpcutil.Signal   { return s.done }
func (s *Stream) SendSig() *drpcutil.Signal   { return s.send }
func (s *Stream) Queue() chan drpcwire.Packet { return s.queue }

//
// helpers
//

func combineTwoErrors(err1, err2 error) error {
	if err1 == nil {
		return err2
	}
	if err2 == nil {
		return err1
	}
	return errs.Combine(err1, err2)
}

func (s *Stream) newPacket(kind drpcwire.PacketKind, data []byte) drpcwire.Packet {
	return drpcwire.Packet{
		Kind: kind,
		Data: data,
	}
}

func (s *Stream) pollWrite(fr drpcwire.Frame) (err error) {
	s.mu.Lock()
	select {
	case <-s.done.Signal():
		err = s.done.Err()
	case <-s.send.Signal():
		err = s.send.Err()
	default:
		err = s.wr.WriteFrame(fr)
	}
	s.mu.Unlock()
	return err
}

func (s *Stream) sendPacket(kind drpcwire.PacketKind, data []byte) error {
	err1 := s.wr.WritePacket(s.newPacket(kind, data))
	err2 := s.wr.Flush()
	return combineTwoErrors(err1, err2)
}

//
// raw read/write
//

func (s *Stream) RawWrite(kind drpcwire.PacketKind, data []byte) error {
	err := drpcwire.SplitN(s.newPacket(kind, data), 0, s.pollWriteFn)
	if err != nil {
		return combineTwoErrors(err, s.SendError(err))
	}
	return nil
}

func (s *Stream) RawFlush() (err error) {
	s.mu.Lock()
	select {
	case <-s.done.Signal():
		err = s.done.Err()
	default:
		err = s.wr.Flush()
	}
	s.mu.Unlock()

	if err != nil {
		return combineTwoErrors(err, s.SendError(err))
	}
	return nil
}

func (s *Stream) RawRecv() ([]byte, error) {
	pkt, ok := <-s.queue
	if !ok {
		return nil, io.EOF
	}
	return pkt.Data, nil
}

//
// msg read/write
//

func (s *Stream) MsgSend(msg drpc.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	if err := s.RawWrite(drpcwire.PacketKind_Message, data); err != nil {
		return err
	}
	if err := s.RawFlush(); err != nil {
		return err
	}
	return nil
}

func (s *Stream) MsgRecv(msg drpc.Message) error {
	data, err := s.RawRecv()
	if err != nil {
		return err
	}
	return proto.Unmarshal(data, msg)
}

//
// terminal messages
//

func (s *Stream) SendError(err error) error {
	s.mu.Lock()
	select {
	case <-s.done.Signal():
		err = s.done.Err()
	default:
		err = s.sendPacket(drpcwire.PacketKind_Error, []byte(err.Error()))
		s.done.Set(drpc.Error.New("stream terminated by sending error"))
	}
	s.mu.Unlock()

	return err
}

func (s *Stream) Close() error {
	var err error

	s.mu.Lock()
	select {
	case <-s.done.Signal():
	default:
		err = s.sendPacket(drpcwire.PacketKind_Close, nil)
		s.done.Set(drpc.Error.New("stream terminated by sending close"))
	}
	s.mu.Unlock()

	return err
}

func (s *Stream) SendCancel() error {
	var err error

	s.mu.Lock()
	select {
	case <-s.done.Signal():
		err = s.done.Err()
	default:
		err = s.sendPacket(drpcwire.PacketKind_Cancel, nil)
		s.done.Set(context.Canceled)
	}
	s.mu.Unlock()

	return err
}

func (s *Stream) CloseSend() error {
	var err error

	s.mu.Lock()
	select {
	case <-s.send.Signal():
	case <-s.done.Signal():
		err = s.done.Err()
	default:
		err = s.sendPacket(drpcwire.PacketKind_CloseSend, nil)
		s.send.Set(drpc.Error.New("send closed"))
	}
	s.mu.Unlock()

	return err
}
