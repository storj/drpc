// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcmanager

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/zeebo/errs"

	"storj.io/drpc"
	"storj.io/drpc/drpcctx"
	"storj.io/drpc/drpcdebug"
	"storj.io/drpc/drpcsignal"
	"storj.io/drpc/drpcstream"
	"storj.io/drpc/drpcwire"
)

type Server interface {
	HandleRPC(stream *drpcstream.Stream, rpc string)
}

var managerClosed = drpc.Error.New("manager closed")

type Manager struct {
	tr  drpc.Transport
	srv Server
	wr  *drpcwire.Writer
	rd  *drpcwire.Reader

	sid     uint64
	sem     chan struct{}
	once    sync.Once         // ensures the shutdown procedure only happens once
	closing drpcsignal.Signal // closing is set when the manager is intending to close
	done    drpcsignal.Signal // done is set when the manager is fully closed
	reader  drpcsignal.Signal // reader is set when the reader goroutine has exited
	queue   chan drpcwire.Packet
}

func New(tr drpc.Transport, srv Server) *Manager {
	m := &Manager{
		tr:  tr,
		srv: srv,
		wr:  drpcwire.NewWriter(tr, 1024),
		rd:  drpcwire.NewReader(tr),

		sem:     make(chan struct{}, 1), // we only allow 1 concurrent stream at a time
		closing: drpcsignal.New(),
		done:    drpcsignal.New(),
		reader:  drpcsignal.New(),
		queue:   make(chan drpcwire.Packet),
	}

	go m.manageReader()

	return m
}

func (m *Manager) kind() string {
	if m.srv == nil {
		return "CLIENT"
	}
	return "SERVER"
}

func (m *Manager) DoneSig() *drpcsignal.Signal { return &m.done }

func (m *Manager) Close() (err error) {
	m.once.Do(func() {
		m.closing.Set(managerClosed) // signal our intent to close
		err = m.tr.Close()           // close the underlying transport
		<-m.reader.Signal()          // wait for the reader to exit
		m.sem <- struct{}{}          // acquire the semaphore to ensure no streams exist
		m.done.Set(managerClosed)    // set that we're now fully closed
	})
	return errs.Wrap(err)
}

func (m *Manager) acquireSemaphore(ctx context.Context) (err error) {
	select {
	case <-m.done.Signal():
		return m.done.Err()
	case <-m.closing.Signal():
		return m.closing.Err()
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	select {
	case <-m.done.Signal():
		return m.done.Err()
	case <-m.closing.Signal():
		return m.closing.Err()
	case <-ctx.Done():
		return ctx.Err()
	case m.sem <- struct{}{}:
		return nil
	}
}

func (m *Manager) NewStream(ctx context.Context, sid uint64) (*drpcstream.Stream, error) {
	if err := m.acquireSemaphore(ctx); err != nil {
		return nil, errs.Wrap(err)
	}

	if sid == 0 {
		m.sid++
		sid = m.sid
	}

	stream := drpcstream.New(drpcctx.WithTransport(ctx, m.tr), sid, m.wr)
	drpcdebug.Log(func() string { return fmt.Sprintf("RPC[%s][%p][%d]: allocate stream\n", m.kind(), stream, sid) })
	go m.manageStream(ctx, stream)

	return stream, nil
}

//
// manage reading from the transport
//

func (m *Manager) manageReader() {
	for m.doManageReader() {
	}
	m.reader.Set(nil)
}

func (m *Manager) doManageReader() bool {
	pkt, err := m.rd.ReadPacket()
	if err != nil {
		m.done.Set(errs.Wrap(err))
		return false
	}

	drpcdebug.Log(func() string { return fmt.Sprintf("RPC[%s]: pkt: %s\n", m.kind(), pkt) })

	if pkt.Kind == drpcwire.Kind_Invoke {
		if m.srv == nil {
			m.done.Set(drpc.ProtocolError.New("invoke sent to client"))
			return false
		}

		stream, err := m.NewStream(context.Background(), pkt.ID.Stream)
		if err != nil {
			m.done.Set(errs.Wrap(err))
			return false
		}
		drpcdebug.Log(func() string {
			return fmt.Sprintf("RPC[%s][%p][%d]: invoke %q\n", m.kind(), stream, pkt.ID.Stream, pkt.Data)
		})
		go m.srv.HandleRPC(stream, string(pkt.Data))

		return true
	}

	select {
	case <-m.done.Signal():
		return false
	case <-m.closing.Signal():
		return false
	case m.queue <- pkt:
		return true

	// In the case that the producer has sent a message and there's no stream to consume it
	// we need to drop it on the floor to continue reading packets. This isn't an error because
	// a producer may be async sending messages and the consumer (i.e. us) may async close
	// from them and they already have messages in flight. By acquring the semaphore, we know
	// that there's no stream, so we immediately release the semaphore and drop the packet.
	case m.sem <- struct{}{}:
		<-m.sem
		return true
	}
}

//
// manage sending packets into the stream
//

func (m *Manager) manageStream(ctx context.Context, stream *drpcstream.Stream) {
	for {
		var err error
		var ok bool

		select {
		case <-stream.DoneSig().Signal():
			err, ok = stream.DoneSig().Err(), false

		case <-ctx.Done():
			err, ok = ctx.Err(), false

		case <-m.done.Signal():
			err, ok = m.done.Err(), false

		case pkt := <-m.queue:
			err, ok = m.handlePacket(ctx, stream, pkt)
		}

		switch {
		case ok:
			continue

		case
			err == context.Canceled,
			err == context.DeadlineExceeded:
			_ = stream.SendCancel(err)

		case err != nil:
			_ = stream.SendError(err)
		}

		// We can be sure the queue will not be used for this stream anymore.
		stream.CloseQueue()

		<-m.sem
		return
	}
}

func (m *Manager) handlePacket(ctx context.Context, stream *drpcstream.Stream, pkt drpcwire.Packet) (err error, ok bool) {
	drpcdebug.Log(func() string { return fmt.Sprintf("RPC[%s][%p][%d]: recv %s\n", m.kind(), stream, stream.ID(), pkt) })

	// Ignore packets for the wrong stream. This is to avoid races where the remote
	// is streaming in messages to us and we async close on them.
	if pkt.ID.Stream != stream.ID() {
		return nil, true
	}

	switch pkt.Kind {
	case drpcwire.Kind_Error:
		stream.ReadErrSig().Set(drpcwire.UnmarshalError(pkt.Data))
		stream.CloseQueue()
		return nil, false

	case drpcwire.Kind_Cancel:
		return context.Canceled, false

	case drpcwire.Kind_Invoke:
		return drpc.ProtocolError.New("invalid invoke sent"), false

	case drpcwire.Kind_Close:
		return drpc.Error.New("remote closed stream"), false

	case drpcwire.Kind_CloseSend:
		stream.ReadErrSig().Set(io.EOF)
		stream.CloseQueue()
		return nil, true

	case drpcwire.Kind_Message:
		if stream.QueueClosed() {
			return drpc.ProtocolError.New("message send after read queue closed"), false
		}

		// We do the double select pattern so that if we're calling this strictly after
		// some closing event has happened, we are certain to take that case. If we
		// didn't do this, then it's possible that pseudo-randomly the stream send into
		// the queue would happen.

		select {
		case <-m.done.Signal():
			return m.done.Err(), false

		case <-ctx.Done():
			return ctx.Err(), false

		case <-stream.DoneSig().Signal():
			return stream.DoneSig().Err(), false

		default:
		}

		select {
		case <-m.done.Signal():
			return m.done.Err(), false

		case <-ctx.Done():
			return ctx.Err(), false

		case <-stream.DoneSig().Signal():
			return stream.DoneSig().Err(), false

		case stream.Queue() <- pkt:
			return nil, true
		}

	default:
		return drpc.ProtocolError.New("unknown packet kind"), false
	}
}
