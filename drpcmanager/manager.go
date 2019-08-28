// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcmanager

import (
	"context"
	"sync"

	"github.com/zeebo/errs"
	"storj.io/drpc"
	"storj.io/drpc/drpcsignal"
	"storj.io/drpc/drpcstream"
	"storj.io/drpc/drpcwire"
)

type Server interface {
	HandleRPC(stream *drpcstream.Stream, rpc string) error
}

var managerClosed = drpc.Error.New("manager closed")

type Manager struct {
	tr  drpc.Transport
	srv Server
	wr  *drpcwire.Writer
	rd  *drpcwire.Reader

	sid   uint64
	once  sync.Once
	sem   chan struct{}
	done  drpcsignal.Signal
	queue chan drpcwire.Packet
}

func New(tr drpc.Transport, srv Server) *Manager {
	m := &Manager{
		tr:  tr,
		srv: srv,
		wr:  drpcwire.NewWriter(tr, 1024),
		rd:  drpcwire.NewReader(tr),

		sem:   make(chan struct{}, 2),
		done:  drpcsignal.New(),
		queue: make(chan drpcwire.Packet),
	}

	m.sem <- struct{}{}
	go m.manageReader()

	return m
}

func (m *Manager) DoneSig() *drpcsignal.Signal { return &m.done }

func (m *Manager) Close() (err error) {
	m.once.Do(func() {
		err = m.tr.Close()
		m.sem <- struct{}{}
		m.sem <- struct{}{}
	})
	m.done.Set(managerClosed)
	return err
}

func (m *Manager) acquireSemaphore(ctx context.Context) (err error) {
	select {
	case <-m.done.Signal():
		return m.done.Err()
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	select {
	case <-m.done.Signal():
		return m.done.Err()
	case <-ctx.Done():
		return ctx.Err()
	case m.sem <- struct{}{}:
		return nil
	}
}

func (m *Manager) NewStream(ctx context.Context, sid uint64) (*drpcstream.Stream, error) {
	if err := m.acquireSemaphore(ctx); err != nil {
		return nil, err
	}

	if sid == 0 {
		m.sid++
		sid = m.sid
	}

	stream := drpcstream.New(ctx, sid, m.wr)
	go m.manageStream(ctx, stream)

	return stream, nil
}

//
// manage reading from the transport
//

func (m *Manager) manageReader() {
	for {
		err := m.doManageReader()
		if err != nil {
			m.done.Set(err)
			<-m.sem
			return
		}
	}
}

func (m *Manager) doManageReader() error {
	pkt, err := m.rd.ReadPacket()
	if err != nil {
		return err
	}

	if pkt.Kind == drpcwire.Kind_Invoke {
		if m.srv == nil {
			return drpc.ProtocolError.New("invoke sent to client")
		}

		stream, err := m.NewStream(context.Background(), pkt.ID.Stream)
		if err != nil {
			return err
		}
		go m.srv.HandleRPC(stream, string(pkt.Data))

		return nil
	}

	select {
	case <-m.done.Signal():
		return m.done.Err()
	case m.queue <- pkt:
		return nil
	}
}

//
// manage sending packets into the stream
//

func (m *Manager) manageStream(ctx context.Context, stream *drpcstream.Stream) {
	for {
		var err error

		select {
		case <-stream.DoneSig().Signal():
			err = stream.DoneSig().Err()

		case <-ctx.Done():
			err = ctx.Err()

		case <-m.done.Signal():
			err = m.done.Err()

		case pkt := <-m.queue:
			err = m.handlePacket(ctx, stream, pkt)
		}

		switch {
		case err == nil:
			continue
		case err == context.Canceled:
			_ = stream.SendCancel()
		case err != nil:
			_ = stream.SendError(err)
		}

		stream.CloseQueue()
		_ = stream.CloseSend()
		<-m.sem
		return
	}
}

func (m *Manager) handlePacket(ctx context.Context, stream *drpcstream.Stream, pkt drpcwire.Packet) error {
	if pkt.ID.Stream != stream.ID() {
		return drpc.ProtocolError.New("invalid stream id")
	}

	switch pkt.Kind {
	case drpcwire.Kind_Error:
		return errs.New("%s", pkt.Data)

	case drpcwire.Kind_Cancel:
		_ = stream.SendCancel()
		return context.Canceled

	case drpcwire.Kind_Invoke:
		return drpc.ProtocolError.New("invalid invoke sent")

	case drpcwire.Kind_Close:
		return drpc.Error.New("remote closed stream")

	case drpcwire.Kind_CloseSend:
		stream.CloseQueue()
		return nil

	case drpcwire.Kind_Message:
		if stream.QueueClosed() {
			return drpc.ProtocolError.New("message send after SendClose")
		}

		select {
		case <-m.done.Signal():
			return m.done.Err()

		case <-ctx.Done():
			_ = stream.SendCancel()
			return context.Canceled

		default:
		}

		select {
		case <-m.done.Signal():
			return m.done.Err()

		case <-ctx.Done():
			_ = stream.SendCancel()
			return context.Canceled

		case stream.Queue() <- pkt:
			return nil
		}

	default:
		return drpc.ProtocolError.New("unknown packet kind")
	}
}
