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
	"storj.io/drpc/drpcsignal"
	"storj.io/drpc/drpcwire"
)

type Stream struct {
	ctx    context.Context
	cancel func()

	id drpcwire.ID
	wr *drpcwire.Writer

	mu     sync.Mutex
	done   drpcsignal.Signal
	send   drpcsignal.Signal
	closed bool
	queue  chan drpcwire.Packet

	// avoids allocations of closures
	pollWriteFn func(drpcwire.Frame) error
}

var _ drpc.Stream = (*Stream)(nil)

func New(ctx context.Context, sid uint64, wr *drpcwire.Writer) *Stream {
	ctx, cancel := context.WithCancel(ctx)
	s := &Stream{
		ctx:    ctx,
		cancel: cancel,

		id: drpcwire.ID{Stream: sid},
		wr: wr,

		done:  drpcsignal.New(),
		send:  drpcsignal.New(),
		queue: make(chan drpcwire.Packet),
	}

	s.pollWriteFn = s.pollWrite

	return s
}

//
// accessors
//

func (s *Stream) Context() context.Context { return s.ctx }
func (s *Stream) CancelContext()           { s.cancel() }

func (s *Stream) ID() uint64                  { return s.id.Stream }
func (s *Stream) DoneSig() *drpcsignal.Signal { return &s.done }
func (s *Stream) SendSig() *drpcsignal.Signal { return &s.send }

func (s *Stream) Queue() chan drpcwire.Packet { return s.queue }

func (s *Stream) QueueClosed() bool {
	s.mu.Lock()
	closed := s.closed
	s.mu.Unlock()
	return closed
}

func (s *Stream) CloseQueue() {
	s.mu.Lock()
	if !s.closed {
		close(s.queue)
		s.closed = true
	}
	if s.send.IsSet() {
		s.done.Set(drpc.Error.New("both sides closed sends"))
	}
	s.mu.Unlock()
}

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

func (s *Stream) newPacket(kind drpcwire.Kind, data []byte) drpcwire.Packet {
	s.id.Message++
	return drpcwire.Packet{
		Data: data,
		ID:   s.id,
		Kind: kind,
	}
}

func (s *Stream) pollWrite(fr drpcwire.Frame) (err error) {
	s.mu.Lock()
	switch {
	case s.done.IsSet():
		err = s.done.Err()
	case s.send.IsSet():
		err = s.send.Err()
	default:
		err = s.wr.WriteFrame(fr)
	}
	s.mu.Unlock()

	return err
}

func (s *Stream) sendPacket(kind drpcwire.Kind, data []byte) error {
	err1 := s.wr.WritePacket(s.newPacket(kind, data))
	err2 := s.wr.Flush()
	return combineTwoErrors(err1, err2)
}

//
// raw read/write
//

func (s *Stream) RawWrite(kind drpcwire.Kind, data []byte) error {
	s.mu.Lock()
	pkt := s.newPacket(kind, data)
	s.mu.Unlock()

	err := drpcwire.SplitN(pkt, 0, s.pollWriteFn)
	if err != nil {
		return combineTwoErrors(err, s.SendError(err))
	}
	return nil
}

func (s *Stream) RawFlush() (err error) {
	s.mu.Lock()
	switch {
	case s.done.IsSet():
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
	if err := s.RawWrite(drpcwire.Kind_Message, data); err != nil {
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
	switch {
	case s.done.IsSet():
		err = s.done.Err()
	default:
		err = s.sendPacket(drpcwire.Kind_Error, drpcwire.MarshalError(err))
		s.done.Set(drpc.Error.New("stream terminated by sending error"))
	}
	s.mu.Unlock()

	return err
}

func (s *Stream) Close() error {
	var err error

	s.mu.Lock()
	switch {
	case s.done.IsSet():
	default:
		err = s.sendPacket(drpcwire.Kind_Close, nil)
		s.done.Set(drpc.Error.New("stream terminated by sending close"))
	}
	s.mu.Unlock()

	return err
}

func (s *Stream) SendCancel() error {
	var err error

	s.mu.Lock()
	switch {
	case s.done.IsSet():
		err = s.done.Err()
	default:
		err = s.sendPacket(drpcwire.Kind_Cancel, nil)
		s.done.Set(context.Canceled)
	}
	s.mu.Unlock()

	return err
}

func (s *Stream) CloseSend() error {
	var err error

	s.mu.Lock()
	switch {
	case s.done.IsSet():
		err = s.done.Err()
	case s.send.IsSet():
	default:
		err = s.sendPacket(drpcwire.Kind_CloseSend, nil)
		s.send.Set(drpc.Error.New("send closed"))
	}
	if s.closed {
		s.done.Set(drpc.Error.New("both sides closed sends"))
	}
	s.mu.Unlock()

	return err
}
