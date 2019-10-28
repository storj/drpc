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

	writeMu chMutex
	id      drpcwire.ID
	wr      *drpcwire.Writer

	mu   sync.Mutex // protects state transitions
	sigs struct {
		send   drpcsignal.Signal // set when done sending messages
		recv   drpcsignal.Signal // set when done receiving messages
		term   drpcsignal.Signal // set when in terminated state
		finish drpcsignal.Signal // set when all writes are complete
		cancel drpcsignal.Signal // set when externally canceled and transport will be closed
	}
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

		wr: wr,

		id:    drpcwire.ID{Stream: sid},
		queue: make(chan drpcwire.Packet),
	}

	s.pollWriteFn = s.pollWrite

	return s
}

//
// accessors
//

func (s *Stream) Context() context.Context { return s.ctx }

func (s *Stream) Terminated() <-chan struct{} { return s.sigs.term.Signal() }

func (s *Stream) Finished() bool { return s.sigs.finish.IsSet() }

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
		if s.sigs.recv.IsSet() || s.sigs.term.IsSet() {
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
		case <-s.sigs.recv.Signal():
		case <-s.sigs.term.Signal():
		case s.queue <- pkt:
		}

		return nil, true

	case drpcwire.Kind_Error:
		err := drpcwire.UnmarshalError(pkt.Data)
		s.sigs.send.Set(io.EOF) // in this state, gRPC returns io.EOF on send.
		s.terminate(err)
		return nil, false

	case drpcwire.Kind_Close:
		s.sigs.recv.Set(io.EOF)
		s.terminate(drpc.Error.New("remote closed the stream"))
		return nil, false

	case drpcwire.Kind_CloseSend:
		s.sigs.recv.Set(io.EOF)
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

// checkFinished checks to see if the stream is terminated, and if so, sets the finished
// flag. This must be called every time right before we release the write mutex.
func (s *Stream) checkFinished() {
	if s.sigs.term.IsSet() {
		s.sigs.finish.Set(nil)
	}
}

// checkCancelError will replace the error with one from the cancel signal if it is
// set. This is to prevent errors from reads/writes to a transport after it has been
// asynchronously closed due to context cancelation.
func (s *Stream) checkCancelError(err error) error {
	if err, ok := s.sigs.cancel.Get(); ok {
		return err
	}
	return err
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
	if s.sigs.send.IsSet() {
		return s.sigs.send.Err()
	} else if s.sigs.term.IsSet() {
		return s.sigs.term.Err()
	} else {
		return s.checkCancelError(errs.Wrap(s.wr.WriteFrame(fr)))
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
	if s.sigs.send.IsSet() && s.sigs.recv.IsSet() {
		s.terminate(drpc.Error.New("stream terminated by both issuing close send"))
	}
}

func (s *Stream) terminate(err error) {
	s.sigs.send.Set(err)
	s.sigs.recv.Set(err)
	s.sigs.term.Set(err)
	s.cancel()

	// if we can acquire the write mutex, then checkFinished. if not, then we know
	// some other write is happening, and it will call checkFinished before it
	// releases the mutex.
	if s.writeMu.TryLock() {
		s.checkFinished()
		s.writeMu.Unlock()
	}
}

//
// raw read/write
//

func (s *Stream) RawWrite(kind drpcwire.Kind, data []byte) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	defer s.checkFinished()

	return drpcwire.SplitN(s.newPacket(kind, data), 0, s.pollWriteFn)
}

func (s *Stream) RawFlush() (err error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	defer s.checkFinished()

	return s.checkCancelError(errs.Wrap(s.wr.Flush()))
}

func (s *Stream) RawRecv() ([]byte, error) {
	if s.sigs.recv.IsSet() {
		return nil, s.sigs.recv.Err()
	}

	select {
	case <-s.sigs.recv.Signal():
		return nil, s.sigs.recv.Err()
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

func (s *Stream) SendError(serr error) error {
	s.mu.Lock()
	if s.sigs.term.IsSet() {
		s.mu.Unlock()
		return nil
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	defer s.checkFinished()

	s.sigs.send.Set(io.EOF) // in this state, gRPC returns io.EOF on send.
	s.terminate(drpc.Error.New("stream terminated by sending error"))
	s.mu.Unlock()

	return s.checkCancelError(s.sendPacket(drpcwire.Kind_Error, drpcwire.MarshalError(serr)))
}

func (s *Stream) Close() error {
	s.mu.Lock()
	if s.sigs.term.IsSet() {
		s.mu.Unlock()
		return nil
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	defer s.checkFinished()

	s.terminate(drpc.Error.New("stream terminated by sending close"))
	s.mu.Unlock()

	return s.checkCancelError(s.sendPacket(drpcwire.Kind_Close, nil))
}

func (s *Stream) CloseSend() error {
	s.mu.Lock()
	if s.sigs.send.IsSet() || s.sigs.term.IsSet() {
		s.mu.Unlock()
		return nil
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	defer s.checkFinished()

	s.sigs.send.Set(drpc.Error.New("send closed"))
	s.terminateIfBothClosed()
	s.mu.Unlock()

	return s.checkCancelError(s.sendPacket(drpcwire.Kind_CloseSend, nil))
}

// Cancel transitions the stream into a state where all writes to the transport will return
// the provided error, and terminates the stream.
func (s *Stream) Cancel(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.sigs.term.IsSet() {
		return
	}

	s.sigs.cancel.Set(err)
	s.sigs.send.Set(io.EOF) // in this state, gRPC returns io.EOF on send.
	s.terminate(err)
}
