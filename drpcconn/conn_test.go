// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcconn

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/zeebo/assert"

	"storj.io/drpc"
	"storj.io/drpc/drpcctx"
	"storj.io/drpc/drpcwire"
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

func TestConn_InvokeFlushesSendClose(t *testing.T) {
	ctx := drpcctx.NewTracker(context.Background())
	defer ctx.Wait()
	defer ctx.Cancel()

	pc, ps := net.Pipe()
	defer func() { assert.NoError(t, pc.Close()) }()
	defer func() { assert.NoError(t, ps.Close()) }()

	invokeDone := make(chan struct{})

	ctx.Run(func(ctx context.Context) {
		wr := drpcwire.NewWriter(ps, 64)
		rd := drpcwire.NewReader(ps)

		_, _ = rd.ReadPacket()    // Invoke
		_, _ = rd.ReadPacket()    // Message
		pkt, _ := rd.ReadPacket() // CloseSend

		_ = wr.WritePacket(drpcwire.Packet{
			Data: []byte("qux"),
			ID:   drpcwire.ID{Stream: pkt.ID.Stream, Message: 1},
			Kind: drpcwire.KindMessage,
		})
		_ = wr.Flush()

		_, _ = rd.ReadPacket() // Close
		<-invokeDone           // wait for invoke to return

		// ensure that any later packets are dropped by writing one
		// before closing the transport.
		for i := 0; i < 5; i++ {
			_ = wr.WritePacket(drpcwire.Packet{
				ID:   drpcwire.ID{Stream: pkt.ID.Stream, Message: 2},
				Kind: drpcwire.KindCloseSend,
			})
			_ = wr.Flush()
		}

		_ = ps.Close()
	})

	conn := New(pc)

	in, out := "baz", ""
	assert.NoError(t, conn.Invoke(ctx, "/com.example.Foo/Bar", testEncoding{}, &in, &out))
	assert.True(t, out == "qux")

	invokeDone <- struct{}{} // signal invoke has returned

	// we should eventually notice the transport is closed
	select {
	case <-conn.Closed():
	case <-time.After(1 * time.Second):
		t.Fatal("took too long for conn to be closed")
	}
}
