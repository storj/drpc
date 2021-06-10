// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcconn

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/zeebo/assert"

	"storj.io/drpc"
	"storj.io/drpc/drpcctx"
	"storj.io/drpc/drpcwire"
)

var (
	errNotImplemented = errors.New("not implemented")
	errClosed         = errors.New("use of closed network connection")
)

// Dummy encoding, which assumes the drpc.Message is a *string.
type testEncoding struct{}

func (testEncoding) Marshal(msg drpc.Message) ([]byte, error) {
	return []byte(*msg.(*string)), nil
}

func (testEncoding) Unmarshal(buf []byte, msg drpc.Message) error {
	*msg.(*string) = string(buf)
	return nil
}

func (testEncoding) JSONMarshal(msg drpc.Message) ([]byte, error) {
	return nil, errNotImplemented
}

func (testEncoding) JSONUnmarshal(buf []byte, msg drpc.Message) error {
	return errNotImplemented
}

// Mock transport to simulate a remote endpoint by reading and writing frames
// from a go routine. Implements drpc.Transport.
type testTransport struct {
	readQueue      chan []byte
	readBuf        []byte
	writeQueue     chan []byte
	remoteReadBuf  []byte
	isRemoteClosed bool
	isClosed       bool
}

func newTestTransport() *testTransport {
	return &testTransport{
		readQueue:     make(chan []byte),
		writeQueue:    make(chan []byte),
		remoteReadBuf: make([]byte, 0),
	}
}

// Read the next frame sent over the connection.
func (t *testTransport) remoteReadFrame(ctx context.Context) (drpcwire.Frame, error) {
	for {
		// Attempt to parse a frame.
		rem, fr, ok, err := drpcwire.ParseFrame(t.remoteReadBuf)
		if err != nil {
			return fr, err
		}
		if ok {
			// Frame is complete. Store the remaining bytes for later use.
			t.remoteReadBuf = rem
			return fr, nil
		}
		// Frame is incomplete. Wait for more bytes.
		select {
		case <-ctx.Done():
			return fr, ctx.Err()
		case p, ok := <-t.writeQueue:
			if !ok {
				return fr, io.EOF
			}
			t.remoteReadBuf = append(t.remoteReadBuf, p...)
		}
	}
}

// Inject a frame as if it were sent by the remote.
func (t *testTransport) remoteWriteFrame(ctx context.Context, fr drpcwire.Frame) error {
	if t.isRemoteClosed {
		return errClosed
	}
	bytes := drpcwire.AppendFrame(nil, fr)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case t.readQueue <- bytes:
		return nil
	}
}

// Simulate a close of the connection by the remote.
func (t *testTransport) remoteClose() {
	if !t.isRemoteClosed {
		t.isRemoteClosed = true
		close(t.readQueue)
	}
}

// Implement io.Reader.
func (t *testTransport) Read(p []byte) (int, error) {
	var temp []byte
	if t.readBuf != nil {
		temp = t.readBuf
		t.readBuf = nil
	} else {
		var ok bool
		temp, ok = <-t.readQueue
		if !ok {
			return 0, io.EOF
		}
	}
	readlen := copy(p, temp)
	if readlen < len(temp) {
		t.readBuf = temp[readlen:]
	}
	return readlen, nil
}

// Implement io.Writer.
func (t *testTransport) Write(p []byte) (int, error) {
	if t.isClosed {
		return 0, errClosed
	}
	// Write must not hold on to the buffer, so make a copy.
	buf := make([]byte, len(p))
	writelen := copy(buf, p)
	t.writeQueue <- buf
	return writelen, nil
}

// Implement io.Closer.
func (t *testTransport) Close() error {
	if !t.isClosed {
		t.isClosed = true
		close(t.writeQueue)
	}
	return nil
}

func runTestTransport(ctx context.Context, transport *testTransport) {
	// Close the transport if an error is encountered.
	defer transport.remoteClose()
	// Read any frames sent through the connection.
	for {
		fr, err := transport.remoteReadFrame(ctx)
		if err != nil {
			return
		}
		if fr.Kind == drpcwire.KindCloseSend {
			// Request was fully sent. Write the response:
			// - Message to transmit response data
			// - CloseSend to indicate that the response is done
			err = transport.remoteWriteFrame(ctx, drpcwire.Frame{
				Data:    []byte("qux"),
				ID:      drpcwire.ID{Stream: fr.ID.Stream, Message: 1},
				Kind:    drpcwire.KindMessage,
				Done:    true,
				Control: false,
			})
			if err != nil {
				return
			}
			err = transport.remoteWriteFrame(ctx, drpcwire.Frame{
				Data:    []byte{},
				ID:      drpcwire.ID{Stream: fr.ID.Stream, Message: 2},
				Kind:    drpcwire.KindCloseSend,
				Done:    true,
				Control: false,
			})
			if err != nil {
				return
			}
			transport.remoteClose()
		}
	}
}

func TestConn_InvokeFlushesSendClose(t *testing.T) {
	ctx := drpcctx.NewTracker(context.Background())
	defer ctx.Wait()
	defer ctx.Cancel()

	transport := newTestTransport()
	ctx.Run(func(ctx context.Context) {
		runTestTransport(ctx, transport)
	})

	conn := New(transport)
	in := "baz"
	var out string
	assert.NoError(t, conn.Invoke(ctx, "/com.example.Foo/Bar", testEncoding{}, &in, &out))

	assert.True(t, out == "qux")

	// If "Invoke()" returns without processing the "CloseSend" message from the
	// remote, the manager does not signal that the connection was closed,
	// before another stream is opened.
	// Wait with a short timeout to prevent the test from blocking for a long
	// time in case it fails.
	closeDetected := false
	select {
	case <-conn.Closed():
		closeDetected = true
	case <-time.After(1 * time.Second):
	}
	assert.True(t, closeDetected)
}
