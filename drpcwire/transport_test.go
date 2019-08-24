// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcwire_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/zeebo/assert"
	"storj.io/drpc/drpctest"
	"storj.io/drpc/drpcwire"
)

func TestWriter(t *testing.T) {
	run := func(size int) func(t *testing.T) {
		return func(t *testing.T) {
			var exp []byte
			var got bytes.Buffer

			wr := drpcwire.NewWriter(&got, size)
			for i := 0; i < 1000; i++ {
				fr := drpctest.RandFrame()
				exp = drpcwire.AppendFrame(exp, fr)
				assert.NoError(t, wr.WriteFrame(fr))
			}
			assert.NoError(t, wr.Flush())
			assert.That(t, bytes.Equal(exp, got.Bytes()))
		}
	}

	t.Run("Size 0B", run(0))
	t.Run("Size 1MB", run(1024*1024))
}

func TestReader(t *testing.T) {
	type testCase struct {
		Packet drpcwire.Packet
		Frames []drpcwire.Frame
	}

	p := func(kind drpcwire.PacketKind, data string) drpcwire.Packet {
		return drpcwire.Packet{
			Kind: kind,
			Data: []byte(data),
		}
	}

	f := func(kind drpcwire.PacketKind, data string, done bool) drpcwire.Frame {
		return drpcwire.Frame{
			Kind: kind,
			Data: []byte(data),
			Done: done,
		}
	}

	m := func(pkt drpcwire.Packet, frames ...drpcwire.Frame) testCase {
		return testCase{
			Packet: pkt,
			Frames: frames,
		}
	}

	cases := []testCase{
		m(p(drpcwire.PacketKind_Message, "hello world"),
			f(drpcwire.PacketKind_Message, "hello", false),
			f(drpcwire.PacketKind_Message, " ", false),
			f(drpcwire.PacketKind_Message, "world", true)),

		m(p(drpcwire.PacketKind_Cancel, ""),
			f(drpcwire.PacketKind_Message, "hello", false),
			f(drpcwire.PacketKind_Message, " ", false),
			f(drpcwire.PacketKind_Cancel, "", true)),
	}

	for _, tc := range cases {
		var buf []byte
		for _, fr := range tc.Frames {
			buf = drpcwire.AppendFrame(buf, fr)
		}

		rd := drpcwire.NewReader(bytes.NewReader(buf))
		pkt, err := rd.ReadPacket()
		assert.NoError(t, err)
		assert.DeepEqual(t, tc.Packet, pkt)
		_, err = rd.ReadPacket()
		assert.Equal(t, err, io.EOF)
	}
}
