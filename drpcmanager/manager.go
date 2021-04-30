// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcmanager

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/zeebo/errs"

	"storj.io/drpc"
	"storj.io/drpc/drpcctx"
	"storj.io/drpc/drpcdebug"
	"storj.io/drpc/drpcmetadata"
	"storj.io/drpc/drpcsignal"
	"storj.io/drpc/drpcstream"
	"storj.io/drpc/drpcwire"
)

var managerClosed = errs.New("manager closed")

// Options controls configuration settings for a manager.
type Options struct {
	// WriterBufferSize controls the size of the buffer that we will fill before
	// flushing. Normal writes to streams typically issue a flush explicitly.
	WriterBufferSize int

	// Stream are passed to any streams the manager creates.
	Stream drpcstream.Options

	// InactivityTimeout is the amount of time the manager will wait when creating
	// a NewServerStream. It only includes the time it is reading packets from the
	// remote client. In other words, it only includes the time that the client
	// could delay before invoking an RPC. If zero or negative, no timeout is used.
	InactivityTimeout time.Duration
}

// Manager handles the logic of managing a transport for a drpc client or server.
// It ensures that the connection is always being read from, that it is closed
// in the case that the manager is and forwarding drpc protocol messages to the
// appropriate stream.
type Manager struct {
	tr   drpc.Transport
	wr   *drpcwire.Writer
	rd   *drpcwire.Reader
	opts Options

	once sync.Once

	sid     uint64
	sem     drpcsignal.Chan
	term    drpcsignal.Signal // set when the manager should start terminating
	read    drpcsignal.Signal // set after the goroutine reading from the transport is done
	tport   drpcsignal.Signal // set after the transport has been closed
	queue   chan drpcwire.Packet
	pktdone drpcsignal.Chan
	prev    *drpcstream.Stream
}

// New returns a new Manager for the transport.
func New(tr drpc.Transport) *Manager {
	return NewWithOptions(tr, Options{})
}

// NewWithOptions returns a new manager for the transport. It uses the provided
// options to manage details of how it uses it.
func NewWithOptions(tr drpc.Transport, opts Options) *Manager {
	m := &Manager{
		tr:   tr,
		wr:   drpcwire.NewWriter(tr, opts.WriterBufferSize),
		rd:   drpcwire.NewReader(tr),
		opts: opts,

		queue: make(chan drpcwire.Packet),
	}

	// this semaphore controls the number of concurrent streams. it MUST be 1.
	m.sem.Make(1)

	// a buffer of size 1 allows the consumer of the packet to signal it is done
	// without having to coordinate with the sender of the packet.
	m.pktdone.Make(1)

	go m.manageTransport()
	go m.manageReader()

	return m
}

//
// helpers
//

// poll checks if a channel is immediately ready.
func poll(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

// poll checks if the context is canceled or the manager is terminated.
func (m *Manager) poll(ctx context.Context) error {
	switch {
	case poll(ctx.Done()):
		return ctx.Err()

	case poll(m.term.Signal()):
		return m.term.Err()

	default:
		return nil
	}
}

// acquireSemaphore attempts to acquire the semaphore protecting streams. If the
// context is canceled or the manager is terminated, it returns an error.
func (m *Manager) acquireSemaphore(ctx context.Context) error {
	if err := m.poll(ctx); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()

	case <-m.term.Signal():
		return m.term.Err()

	case m.sem.Get() <- struct{}{}:
		if err := m.waitForPreviousStream(ctx); err != nil {
			m.sem.Recv()
			return err
		}
		return nil
	}
}

// waitForPreviousStream will, if there was a previous stream, ensure it is Closed and
// then wait until it is in the Finished state, where it will no longer make any
// reads or writes on the transport. It exits early if the context is canceled or
// the manager is terminated.
func (m *Manager) waitForPreviousStream(ctx context.Context) (err error) {
	if m.prev == nil {
		return nil
	}

	if err := m.prev.Close(); err != nil {
		return err
	}

	select {
	case <-m.prev.Finished():
		return nil

	case <-m.term.Signal():
		return m.term.Err()

	case <-ctx.Done():
		return ctx.Err()
	}
}

//
// exported interface
//

// Closed returns if the manager has been closed.
func (m *Manager) Closed() bool {
	return m.term.IsSet()
}

// Close closes the transport the manager is using.
func (m *Manager) Close() error {
	// when closing, we set the manager terminated signal, wait for the goroutine
	// managing the transport to notice and close it, acquire the semaphore to ensure
	// there are streams running, then wait for the goroutine reading packets to be done.
	// we protect it with a once to ensure both that we only do this once, and that
	// concurrent calls are sure that it has fully executed.

	m.once.Do(func() {
		m.term.Set(managerClosed)
		m.tport.Wait()
		m.sem.Send()
		m.read.Wait()
	})

	return m.tport.Err()
}

// newStream creates a stream value with the appropriate configuration for this manager.
func (m *Manager) newStream(ctx context.Context, sid uint64) *drpcstream.Stream {
	ctx = drpcctx.WithTransport(ctx, m.tr)
	stream := drpcstream.NewWithOptions(ctx, sid, m.wr.Reset(), m.opts.Stream)
	m.prev = stream
	go m.manageStream(ctx, stream)
	return stream
}

// NewClientStream starts a stream on the managed transport for use by a client.
func (m *Manager) NewClientStream(ctx context.Context) (stream *drpcstream.Stream, err error) {
	if err := m.acquireSemaphore(ctx); err != nil {
		return nil, err
	}

	m.sid++

	return m.newStream(ctx, m.sid), nil
}

// NewServerStream starts a stream on the managed transport for use by a server. It does
// this by waiting for the client to issue an invoke message and returning the details.
func (m *Manager) NewServerStream(ctx context.Context) (stream *drpcstream.Stream, rpc string, err error) {
	if err := m.acquireSemaphore(ctx); err != nil {
		return nil, "", err
	}
	defer func() {
		if err != nil {
			m.sem.Recv()
		}
	}()

	var meta map[string]string
	var metaID uint64
	var timeoutCh <-chan time.Time

	// set up the timeout channel if necessary.
	if timeout := m.opts.InactivityTimeout; timeout > 0 {
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		timeoutCh = timer.C
	}

	for {
		select {
		case <-timeoutCh:
			return nil, "", context.DeadlineExceeded

		case <-ctx.Done():
			return nil, "", ctx.Err()

		case <-m.term.Signal():
			return nil, "", m.term.Err()

		case pkt := <-m.queue:
			switch pkt.Kind {
			// keep track of any metadata being sent before an invoke so that we can
			// include it if the stream id matches the eventual invoke.
			case drpcwire.KindInvokeMetadata:
				meta, err = drpcmetadata.Decode(pkt.Data)
				m.pktdone.Send()

				if err != nil {
					return nil, "", err
				}
				metaID = pkt.ID.Stream

			case drpcwire.KindInvoke:
				rpc = string(pkt.Data)
				m.pktdone.Send()

				if metaID == pkt.ID.Stream {
					ctx = drpcmetadata.AddPairs(ctx, meta)
				}

				return m.newStream(ctx, pkt.ID.Stream), rpc, nil

			default:
				// we ignore packets that aren't invokes because perhaps older streams have
				// messages in the queue sent concurrently with our notification to them
				// that the stream they were sent for is done.
				m.pktdone.Send()
			}
		}
	}
}

//
// manage transport
//

// manageTransport ensures that if the manager's term signal is ever set, then
// the underlying transport is closed and the error is recorded.
func (m *Manager) manageTransport() {
	m.term.Wait()
	drpcdebug.Log(func() string { return fmt.Sprintf("MAN[%p]: term: %v", m, m.term.Err()) })
	m.tport.Set(m.tr.Close())
}

//
// manage reader
//

// manageReader is always reading a packet and sending it into the queue of packets
// the manager has. It sets the read signal when it exits so that one can wait to
// ensure that no one is reading on the reader. It sets the term signal if there is
// any error reading packets.
func (m *Manager) manageReader() {
	defer m.read.Set(managerClosed)

	var buf []byte

	for {
		pkt, err := m.rd.ReadPacketUsing(buf[:0])
		if err != nil {
			if errors.Is(err, io.EOF) {
				m.term.Set(drpc.ClosedError.New("end of stream"))
			} else {
				m.term.Set(errs.Wrap(err))
			}
			return
		}

		drpcdebug.Log(func() string { return fmt.Sprintf("MAN[%p]: %v", m, pkt) })

		select {
		case <-m.term.Signal():
			return

		case m.queue <- pkt:
			m.pktdone.Recv()
			buf = pkt.Data
		}
	}
}

//
// manage stream
//

// manageStream watches the context and the stream and returns when the stream is
// finished, canceling the stream if the context is canceled.
func (m *Manager) manageStream(ctx context.Context, stream *drpcstream.Stream) {
	// create a wait group, launch the workers, and wait for them
	wg := new(sync.WaitGroup)
	wg.Add(2)
	go m.manageStreamPackets(ctx, wg, stream)
	go m.manageStreamContext(ctx, wg, stream)
	wg.Wait()

	// release semaphore
	m.sem.Recv()
}

// manageStreamPackets repeatedly reads from the queue of packets and asks the stream to
// handle them. If there is an error handling a packet, that is considered to
// be fatal to the manager, so we set term. HandlePacket also returns a bool to
// indicate that the stream requires no more packets, and so manageStream can
// just exit. It releases the semaphore whenever it exits.
func (m *Manager) manageStreamPackets(ctx context.Context, wg *sync.WaitGroup, stream *drpcstream.Stream) {
	defer wg.Done()

	for {
		select {
		case <-m.term.Signal():
			stream.Cancel(context.Canceled)
			return

		case <-stream.Terminated():
			return

		case pkt := <-m.queue:
			ok, err := stream.HandlePacket(pkt)
			m.pktdone.Send() // done reading the packet data

			if err != nil {
				m.term.Set(errs.Wrap(err))
				return
			} else if !ok {
				return
			}
		}
	}
}

// manageStreamContext ensures that if the stream context is canceled, we inform the stream and
// possibly abort the underlying transport if the stream isn't finished.
func (m *Manager) manageStreamContext(ctx context.Context, wg *sync.WaitGroup, stream *drpcstream.Stream) {
	defer wg.Done()

	select {
	case <-m.term.Signal():
		stream.Cancel(context.Canceled)
		return

	case <-stream.Terminated():
		return

	case <-ctx.Done():
		// If the stream isn't already finished, we have to terminate the transport
		// to do an active cancel. If it is already finished, there is no need.
		isFinished := stream.IsFinished()
		stream.Cancel(ctx.Err())
		if !isFinished {
			drpcdebug.Log(func() string { return fmt.Sprintf("MAN[%p][%p]: unfinished", m, stream) })
			m.term.Set(ctx.Err())
		}
	}
}
