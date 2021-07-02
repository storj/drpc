// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcmanager

import (
	"context"
	"errors"
	"io"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/zeebo/assert"

	"storj.io/drpc/drpcctx"
	"storj.io/drpc/drpcstream"
	"storj.io/drpc/drpcwire"
)

func TestRandomized_Client(t *testing.T) {
	runRandomized(t, randomBytes(time.Now().UnixNano(), 1024), new(randClient))
}

func TestRandomized_Server(t *testing.T) {
	runRandomized(t, randomBytes(time.Now().UnixNano(), 1024), new(randServer))
}

//
// client tests
//

type randClient struct {
	id     incID
	active bool
}

func (rc *randClient) newSteam(ctx context.Context, man *Manager) (*drpcstream.Stream, error) {
	stream, _, err := man.NewServerStream(ctx)
	return stream, err
}

func (rc *randClient) execute(t *testing.T, wr *drpcwire.Writer, op byte) {
	cmd, arg, done := parseOp(op)

	if !rc.active {
		assert.NoError(t, wr.WritePacket(drpcwire.Packet{
			Data: make([]byte, arg),
			ID:   rc.id.incMessage(),
			Kind: drpcwire.KindInvoke,
		}))
		rc.active = true
	}

	switch cmd {
	case 0: // new invoke
		if rc.active {
			assert.NoError(t, wr.WritePacket(drpcwire.Packet{
				ID:   rc.id.incMessage(),
				Kind: drpcwire.KindClose,
			}))
		}

		rc.id.incStream()

		for i := 0; i < arg; i++ {
			assert.NoError(t, wr.WriteFrame(drpcwire.Frame{
				ID:   rc.id.incMessage(),
				Kind: drpcwire.KindInvokeMetadata,
				Done: done,
			}))
		}

		assert.NoError(t, wr.WriteFrame(drpcwire.Frame{
			Data: make([]byte, arg),
			ID:   rc.id.incMessage(),
			Kind: drpcwire.KindInvoke,
			Done: done,
		}))
		rc.active = done

	case 1: // terminate (close send, close, error)
		kind := [...]drpcwire.Kind{
			drpcwire.KindCloseSend,
			drpcwire.KindClose,
			drpcwire.KindError,
		}[arg%3]

		assert.NoError(t, wr.WriteFrame(drpcwire.Frame{
			Data: make([]byte, 8),
			ID:   rc.id.incMessage(),
			Kind: kind,
			Done: done,
		}))

	case 2: // cause the remote side to close
		assert.NoError(t, wr.WritePacket(drpcwire.Packet{
			Data: []byte("remote-close"),
			ID:   rc.id.incMessage(),
			Kind: drpcwire.KindMessage,
		}))

	case 3, 4, 5, 6, 7: // send normal message
		assert.NoError(t, wr.WriteFrame(drpcwire.Frame{
			Data: make([]byte, arg),
			ID:   rc.id.incMessage(),
			Kind: drpcwire.KindMessage,
			Done: done,
		}))

	default:
		t.Fatalf("unknown command: %d", cmd)
	}
}

//
// server tests
//

type randServer struct {
	id incID
}

func (rs *randServer) newSteam(ctx context.Context, man *Manager) (*drpcstream.Stream, error) {
	return man.NewClientStream(ctx)
}

func (rs *randServer) execute(t *testing.T, wr *drpcwire.Writer, op byte) {
	cmd, arg, done := parseOp(op)

	switch cmd {
	case 0: // begin a new stream
		rs.id.incStream()

		assert.NoError(t, wr.WriteFrame(drpcwire.Frame{
			Data: make([]byte, arg),
			ID:   rs.id.incMessage(),
			Kind: drpcwire.KindMessage,
			Done: done,
		}))

	case 1: // terminate (close send, close, error)
		kind := [...]drpcwire.Kind{
			drpcwire.KindCloseSend,
			drpcwire.KindClose,
			drpcwire.KindError,
		}[arg%3]

		assert.NoError(t, wr.WriteFrame(drpcwire.Frame{
			Data: make([]byte, 8),
			ID:   rs.id.incMessage(),
			Kind: kind,
			Done: done,
		}))

	case 2: // cause the remote side to close
		assert.NoError(t, wr.WritePacket(drpcwire.Packet{
			Data: []byte("remote-close"),
			ID:   rs.id.incMessage(),
			Kind: drpcwire.KindMessage,
		}))

	case 3, 4, 5, 6, 7: // send random message
		assert.NoError(t, wr.WriteFrame(drpcwire.Frame{
			Data: make([]byte, arg),
			ID:   rs.id.incMessage(),
			Kind: drpcwire.KindMessage,
			Done: done,
		}))

	default:
		t.Fatalf("unknown command: %d", cmd)
	}
}

//
// test runner
//

type runner interface {
	newSteam(ctx context.Context, man *Manager) (*drpcstream.Stream, error)
	execute(t *testing.T, wr *drpcwire.Writer, op byte)
}

func runRandomized(t *testing.T, prog []byte, r runner) {
	ctx := drpcctx.NewTracker(context.Background())
	defer ctx.Wait()
	defer ctx.Cancel()

	pc, ps := net.Pipe()
	defer func() { _ = pc.Close() }()
	defer func() { _ = ps.Close() }()

	wr := drpcwire.NewWriter(pc, 0)
	man := New(ps)
	defer func() { _ = man.Close() }()

	errch := make(chan error, 1)
	ctx.Run(func(ctx context.Context) {
		errch <- func() error {
			for {
				stream, err := r.newSteam(ctx, man)
				if err != nil {
					return err
				}
				for {
					buf, err := stream.RawRecv()
					if expectedError(err) || string(buf) == "remote-close" {
						stream.Cancel(context.Canceled)
						break
					} else if err != nil {
						return err
					}
				}
			}
		}()
	})

	for _, op := range prog {
		r.execute(t, wr, op)
		assert.NoError(t, wr.Flush())
	}

	assert.NoError(t, man.Close())
	assert.Equal(t, (<-errch).Error(), "manager closed: Close called")
}

//
// helpers
//

func expectedError(err error) bool {
	return errors.Is(err, io.EOF) ||
		errors.Is(err, context.Canceled) ||
		(err != nil && err.Error() == "")
}

func parseOp(op byte) (cmd byte, arg int, done bool) {
	cmd, op = op&0b111, op>>3
	arg, op = int(op&0b1111), op>>4
	done = op&0b1 > 0
	return cmd, arg, done
}

func randomBytes(seed int64, n int) []byte {
	out := make([]byte, n)
	_, _ = rand.New(rand.NewSource(seed)).Read(out)
	return out
}

type incID drpcwire.ID

func (id *incID) incStream() { *id = incID{Stream: id.Stream + 1} }
func (id *incID) incMessage() drpcwire.ID {
	id.Message++
	return drpcwire.ID{Stream: id.Stream + 1, Message: id.Message}
}
