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

	p := func(kind drpcwire.Kind, id uint64, data string) drpcwire.Packet {
		return drpcwire.Packet{
			Data: []byte(data),
			ID:   drpcwire.ID{Stream: 1, Message: id},
			Kind: kind,
		}
	}

	f := func(kind drpcwire.Kind, id uint64, data string, done bool) drpcwire.Frame {
		return drpcwire.Frame{
			Data: []byte(data),
			ID:   drpcwire.ID{Stream: 1, Message: id},
			Kind: kind,
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
		m(p(drpcwire.Kind_Message, 1, "hello world"),
			f(drpcwire.Kind_Message, 1, "hello", false),
			f(drpcwire.Kind_Message, 1, " ", false),
			f(drpcwire.Kind_Message, 1, "world", true)),

		m(p(drpcwire.Kind_Cancel, 2, ""),
			f(drpcwire.Kind_Message, 1, "hello", false),
			f(drpcwire.Kind_Message, 1, " ", false),
			f(drpcwire.Kind_Cancel, 2, "", true)),
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
