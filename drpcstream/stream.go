// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcstream

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/zeebo/errs"

	"storj.io/drpc"
	"storj.io/drpc/drpcdebug"
	"storj.io/drpc/drpcenc"
	"storj.io/drpc/drpcsignal"
	"storj.io/drpc/drpcwire"
)

// Options controls configuration settings for a stream.
type Options struct {
	// SplitSize controls the default size we split packets into frames.
	SplitSize int
}

// Stream represents an rpc actively happening on a transport.
type Stream struct {
	ctx  streamCtx
	opts Options

	write chMutex
	read  chMutex

	id drpcwire.ID
	wr *drpcwire.Writer

	mu   sync.Mutex // protects state transitions
	sigs struct {
		send   drpcsignal.Signal // set when done sending messages
		recv   drpcsignal.Signal // set when done receiving messages
		term   drpcsignal.Signal // set when the stream is terminating and no new ops should begin
		fin    drpcsignal.Signal // set when the stream is finished and all ops are complete
		cancel drpcsignal.Signal // set when externally canceled
	}
	queue chan drpcwire.Packet
	wbuf  []byte
}

var _ drpc.Stream = (*Stream)(nil)

// New returns a new stream bound to the context with the given stream id and will
// use the writer to write messages on. It is important use monotonically increasing
// stream ids within a single transport.
func New(ctx context.Context, sid uint64, wr *drpcwire.Writer) *Stream {
	return NewWithOptions(ctx, sid, wr, Options{})
}

// NewWithOptions returns a new stream bound to the context with the given stream id
// and will use the writer to write messages on. It is important use monotonically increasing
// stream ids within a single transport. The options are used to control details of how
// the Stream operates.
func NewWithOptions(ctx context.Context, sid uint64, wr *drpcwire.Writer, opts Options) *Stream {
	return &Stream{
		ctx:  streamCtx{Context: ctx},
		opts: opts,

		wr: wr,

		id:    drpcwire.ID{Stream: sid},
		queue: make(chan drpcwire.Packet),
	}
}

//
// accessors
//

// streamCtx avoids having to allocate a Done channel until it is requested.
type streamCtx struct {
	context.Context
	ch drpcsignal.Chan
}

// Done returns the stored channel instead of the parent Done channel.
func (s *streamCtx) Done() <-chan struct{} { return s.ch.Get() }

// Context returns the context associated with the stream. It is closed when
// the Stream will no longer issue any writes or reads.
func (s *Stream) Context() context.Context { return &s.ctx }

// Terminated returns a channel that is closed when the stream has been terminated.
func (s *Stream) Terminated() <-chan struct{} { return s.sigs.term.Signal() }

// Finished returns a channel that is closed when the stream is fully finished
// and will no longer issue any writes or reads.
func (s *Stream) Finished() <-chan struct{} { return s.sigs.fin.Signal() }

// IsFinished returns true if the stream is fully finished and will no longer
// issue any writes or reads.
func (s *Stream) IsFinished() bool { return s.sigs.fin.IsSet() }

//
// packet handler
//

// HandlePacket advances the stream state machine by inspecting the packet. It returns
// any major errors that should terminate the transport the stream is operating on as
// well as a boolean indicating if the stream expects more packets.
func (s *Stream) HandlePacket(pkt drpcwire.Packet) (more bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	drpcdebug.Log(func() string { return fmt.Sprintf("STR[%p][%d]: %v", s, s.id.Stream, pkt) })

	if pkt.ID.Stream != s.id.Stream {
		return true, nil
	}

	switch pkt.Kind {
	case drpcwire.KindInvoke:
		err := drpc.ProtocolError.New("invoke on existing stream")
		s.terminate(err)
		return false, err

	case drpcwire.KindMessage:
		if s.sigs.recv.IsSet() || s.sigs.term.IsSet() {
			return true, nil
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

		return true, nil

	case drpcwire.KindError:
		err := drpcwire.UnmarshalError(pkt.Data)
		s.sigs.send.Set(io.EOF) // in this state, gRPC returns io.EOF on send.
		s.terminate(err)
		return false, nil

	case drpcwire.KindClose:
		s.sigs.recv.Set(io.EOF)
		s.terminate(drpc.Error.New("remote closed the stream"))
		return false, nil

	case drpcwire.KindCloseSend:
		s.sigs.recv.Set(io.EOF)
		s.terminateIfBothClosed()
		return false, nil

	default:
		err := drpc.InternalError.New("unknown packet kind: %s", pkt.Kind)
		s.terminate(err)
		return false, err
	}
}

//
// helpers
//

// checkFinished checks to see if the stream is terminated, and if so, sets the finished
// flag. This must be called after every read or write is complete, as well as when
// the stream becomes terminated.
func (s *Stream) checkFinished() {
	if s.sigs.term.IsSet() && s.write.Unlocked() && s.read.Unlocked() {
		if s.sigs.fin.Set(nil) {
			s.ctx.ch.Close()
		}
	}
}

// checkCancelError will replace the error with one from the cancel signal if it is
// set. This is to prevent errors from reads/writes to a transport after it has been
// asynchronously closed due to context cancelation.
func (s *Stream) checkCancelError(err error) error {
	if sigErr, ok := s.sigs.cancel.Get(); ok {
		return sigErr
	}
	return err
}

// newPackage bumps the internal message id and returns a packet. It must be called
// under a mutex.
func (s *Stream) newPacket(kind drpcwire.Kind, data []byte) drpcwire.Packet {
	s.id.Message++
	return drpcwire.Packet{
		Data: data,
		ID:   s.id,
		Kind: kind,
	}
}

// pollWrite checks for any conditions that should cause a write to not happen and
// then issues the write of the frame.
func (s *Stream) pollWrite(fr drpcwire.Frame) (err error) {
	switch {
	case s.sigs.send.IsSet():
		return s.sigs.send.Err()
	case s.sigs.term.IsSet():
		return s.sigs.term.Err()
	}

	return s.checkCancelError(errs.Wrap(s.wr.WriteFrame(fr)))
}

// sendPacket sends the packet in a single write and flushes. It does not check for
// any conditions to stop it from writing and is meant for internal stream use to
// do things like signal errors or closes to the remote side.
func (s *Stream) sendPacket(kind drpcwire.Kind, data []byte) (err error) {
	if err := s.wr.WritePacket(s.newPacket(kind, data)); err != nil {
		return errs.Wrap(err)
	}
	if err := s.wr.Flush(); err != nil {
		return errs.Wrap(err)
	}
	return nil
}

// terminateIfBothClosed is a helper to terminate the stream if both sides have
// issued a CloseSend.
func (s *Stream) terminateIfBothClosed() {
	if s.sigs.send.IsSet() && s.sigs.recv.IsSet() {
		s.terminate(termBothClosed)
	}
}

// terminate marks the stream as terminated with the given error. It also marks
// the stream as finished if no writes are happening at the time of the call.
func (s *Stream) terminate(err error) {
	s.sigs.send.Set(err)
	s.sigs.recv.Set(err)
	s.sigs.term.Set(err)
	s.checkFinished()
}

//
// raw read/write
//

// RawWrite sends the data bytes with the given kind.
func (s *Stream) RawWrite(kind drpcwire.Kind, data []byte) (err error) {
	defer s.checkFinished()
	s.write.Lock()
	defer s.write.Unlock()

	return s.rawWriteLocked(kind, data)
}

func (s *Stream) rawWriteLocked(kind drpcwire.Kind, data []byte) (err error) {
	return drpcwire.SplitN(s.newPacket(kind, data), s.opts.SplitSize, s.pollWrite)
}

// RawFlush flushes any buffers of data.
func (s *Stream) RawFlush() (err error) {
	defer s.checkFinished()
	s.write.Lock()
	defer s.write.Unlock()

	return s.rawFlushLocked()
}

func (s *Stream) rawFlushLocked() (err error) {
	return s.checkCancelError(errs.Wrap(s.wr.Flush()))
}

// RawRecv returns the raw bytes received for a message.
func (s *Stream) RawRecv() (data []byte, err error) {
	defer s.checkFinished()
	s.read.Lock()
	defer s.read.Unlock()

	if err, ok := s.sigs.recv.Get(); ok {
		return nil, err
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

// MsgSend marshals the message with the encoding, writes it, and flushes.
func (s *Stream) MsgSend(msg drpc.Message, enc drpc.Encoding) (err error) {
	defer s.checkFinished()
	s.write.Lock()
	defer s.write.Unlock()

	s.wbuf, err = drpcenc.MarshalAppend(msg, enc, s.wbuf[:0])
	if err != nil {
		return errs.Wrap(err)
	}
	if err := s.rawWriteLocked(drpcwire.KindMessage, s.wbuf); err != nil {
		return err
	}
	if err := s.rawFlushLocked(); err != nil {
		return err
	}

	return nil
}

// MsgRecv recives some message data and unmarshals it with enc into msg.
func (s *Stream) MsgRecv(msg drpc.Message, enc drpc.Encoding) (err error) {
	data, err := s.RawRecv()
	if err != nil {
		return err
	}
	return enc.Unmarshal(data, msg)
}

//
// terminal messages
//

var (
	sendClosed     = drpc.Error.New("send closed")
	termError      = drpc.Error.New("stream terminated by sending error")
	termClosed     = drpc.Error.New("stream terminated by sending close")
	termBothClosed = drpc.Error.New("stream terminated by both issuing close send")
)

// SendError terminates the stream and sends the error to the remote. It is a no-op if
// the stream is already terminated.
func (s *Stream) SendError(serr error) (err error) {
	drpcdebug.Log(func() string { return fmt.Sprintf("STR[%p][%d]: SendError(%v)", s, s.id.Stream, serr) })

	s.mu.Lock()
	if s.sigs.term.IsSet() {
		s.mu.Unlock()
		return nil
	}

	defer s.checkFinished()
	s.write.Lock()
	defer s.write.Unlock()

	s.sigs.send.Set(io.EOF) // in this state, gRPC returns io.EOF on send.
	s.terminate(termError)
	s.mu.Unlock()

	return s.checkCancelError(s.sendPacket(drpcwire.KindError, drpcwire.MarshalError(serr)))
}

// Close terminates the stream and sends that the stream has been closed to the remote.
// It is a no-op if the stream is already terminated.
func (s *Stream) Close() (err error) {
	drpcdebug.Log(func() string { return fmt.Sprintf("STR[%p][%d]: Close()", s, s.id.Stream) })

	s.mu.Lock()
	if s.sigs.term.IsSet() {
		s.mu.Unlock()
		return nil
	}

	defer s.checkFinished()
	s.write.Lock()
	defer s.write.Unlock()

	s.terminate(termClosed)
	s.mu.Unlock()

	return s.checkCancelError(s.sendPacket(drpcwire.KindClose, nil))
}

// CloseSend informs the remote that no more messages will be sent. If the remote has
// also already issued a CloseSend, the stream is terminated. It is a no-op if the
// stream already has sent a CloseSend or if it is terminated.
func (s *Stream) CloseSend() (err error) {
	drpcdebug.Log(func() string { return fmt.Sprintf("STR[%p][%d]: CloseSend()", s, s.id.Stream) })

	s.mu.Lock()
	if s.sigs.send.IsSet() || s.sigs.term.IsSet() {
		s.mu.Unlock()
		return nil
	}

	defer s.checkFinished()
	s.write.Lock()
	defer s.write.Unlock()

	s.sigs.send.Set(sendClosed)
	s.terminateIfBothClosed()
	s.mu.Unlock()

	return s.checkCancelError(s.sendPacket(drpcwire.KindCloseSend, nil))
}

// Cancel transitions the stream into a state where all writes to the transport will return
// the provided error, and terminates the stream. It is a no-op if the stream is already
// terminated.
func (s *Stream) Cancel(err error) {
	drpcdebug.Log(func() string { return fmt.Sprintf("STR[%p][%d]: Cancel(%v)", s, s.id.Stream, err) })

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.sigs.term.IsSet() {
		return
	}

	s.sigs.cancel.Set(err)
	s.sigs.send.Set(io.EOF) // in this state, gRPC returns io.EOF on send.
	s.terminate(err)
}
