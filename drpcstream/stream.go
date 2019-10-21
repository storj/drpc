// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcstream

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/zeebo/errs"

	"storj.io/drpc"
	"storj.io/drpc/drpcdebug"
	"storj.io/drpc/drpcsignal"
	"storj.io/drpc/drpcwire"
)

type Stream struct {
	ctx    context.Context
	cancel func()

	id drpcwire.ID
	wr *drpcwire.Writer

	mu    sync.Mutex
	send  drpcsignal.Signal
	recv  drpcsignal.Signal
	term  drpcsignal.Signal
	queue chan drpcwire.Packet

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

		send:  drpcsignal.New(),
		recv:  drpcsignal.New(),
		term:  drpcsignal.New(),
		queue: make(chan drpcwire.Packet),
	}

	s.pollWriteFn = s.pollWrite

	return s
}

//
// accessors
//

func (s *Stream) Context() context.Context { return s.ctx }

func (s *Stream) Terminated() <-chan struct{} { return s.term.Signal() }

//
// packet handler
//

// HandlePacket advances the stream state machine by inspecting the packet. It returns
// any major errors that should terminate the transport the stream is operating on as
// well as a boolean indicating if the stream expects more packets.
func (s *Stream) HandlePacket(pkt drpcwire.Packet) (error, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	drpcdebug.Log(func() string { return fmt.Sprintf("STR[%p][%d]: %v", s, s.id.Stream, pkt) })

	if pkt.ID.Stream != s.id.Stream {
		return nil, true
	}

	switch pkt.Kind {
	case drpcwire.Kind_Invoke:
		err := drpc.ProtocolError.New("invoke on existing stream")
		s.terminate(err)
		return err, false

	case drpcwire.Kind_Message:
		if s.recv.IsSet() || s.term.IsSet() {
			return nil, true
		}

		// drop the mutex while we either send into the queue or we're told that
		// receiving is done. we don't handle any more packets until the message
		// is delivered, so the only way it can become set is from some of the
		// stream terminating calls, in which case, shutting down the stream is
		// racing with the message being received, so dropping it is valid.
		s.mu.Unlock()
		defer s.mu.Lock()

		select {
		case <-s.recv.Signal():
		case <-s.term.Signal():
		case s.queue <- pkt:
		}
		return nil, true

	case drpcwire.Kind_Error:
		err := drpcwire.UnmarshalError(pkt.Data)
		s.send.Set(io.EOF) // weird grpc :(
		s.terminate(err)
		return nil, false

	case drpcwire.Kind_Cancel:
		s.terminate(context.Canceled)
		return nil, false

	case drpcwire.Kind_Close:
		s.recv.Set(io.EOF)
		s.terminate(drpc.Error.New("remote closed the stream"))
		return nil, false

	case drpcwire.Kind_CloseSend:
		s.recv.Set(io.EOF)
		s.terminateIfBothClosed()
		return nil, false

	default:
		err := drpc.InternalError.New("unknown packet kind: %s", pkt.Kind)
		s.terminate(err)
		return err, false
	}
}

//
// helpers
//

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
	defer s.mu.Unlock()

	switch {
	case s.send.IsSet():
		return s.send.Err()
	case s.term.IsSet():
		return s.term.Err()
	default:
		return errs.Wrap(s.wr.WriteFrame(fr))
	}
}

func (s *Stream) sendPacket(kind drpcwire.Kind, data []byte) error {
	if err := s.wr.WritePacket(s.newPacket(kind, data)); err != nil {
		return errs.Wrap(err)
	}
	if err := s.wr.Flush(); err != nil {
		return errs.Wrap(err)
	}
	return nil
}

func (s *Stream) terminateIfBothClosed() {
	if s.send.IsSet() && s.recv.IsSet() {
		s.term.Set(drpc.Error.New("stream terminated by both issuing close send"))
		s.cancel()
	}
}

func (s *Stream) terminate(err error) {
	s.send.Set(err)
	s.recv.Set(err)
	s.term.Set(err)
	s.cancel()
}

//
// raw read/write
//

func (s *Stream) RawWrite(kind drpcwire.Kind, data []byte) error {
	s.mu.Lock()
	pkt := s.newPacket(kind, data)
	s.mu.Unlock()

	return drpcwire.SplitN(pkt, 0, s.pollWriteFn)
}

func (s *Stream) RawFlush() (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return errs.Wrap(s.wr.Flush())
}

func (s *Stream) RawRecv() ([]byte, error) {
	if s.recv.IsSet() {
		return nil, s.recv.Err()
	}

	select {
	case <-s.recv.Signal():
		return nil, s.recv.Err()
	case pkt := <-s.queue:
		return pkt.Data, nil
	}
}

//
// msg read/write
//

func (s *Stream) MsgSend(msg drpc.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return errs.Wrap(err)
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
	defer s.mu.Unlock()

	if s.term.IsSet() {
		return nil
	}

	s.send.Set(io.EOF) // weird grpc :(
	s.terminate(drpc.Error.New("stream terminated by sending error"))

	return s.sendPacket(drpcwire.Kind_Error, drpcwire.MarshalError(err))
}

func (s *Stream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.term.IsSet() {
		return nil
	}

	s.terminate(drpc.Error.New("stream terminated by sending close"))

	return s.sendPacket(drpcwire.Kind_Close, nil)
}

func (s *Stream) SendCancel(err error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.term.IsSet() {
		return nil
	}

	s.send.Set(io.EOF) // weird grpc :(
	s.recv.Set(err)
	s.terminate(drpc.Error.New("stream terminated by sending cancel"))

	return s.sendPacket(drpcwire.Kind_Cancel, nil)
}

func (s *Stream) CloseSend() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.send.IsSet() {
		return nil
	}
	if s.term.IsSet() {
		return nil
	}

	s.send.Set(drpc.Error.New("send closed"))
	s.terminateIfBothClosed()

	return s.sendPacket(drpcwire.Kind_CloseSend, nil)
}
