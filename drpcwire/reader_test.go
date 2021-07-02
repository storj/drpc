// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcwire

import (
	"bytes"
	"errors"
	"io"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/zeebo/assert"
)

func TestReader(t *testing.T) {
	type testCase struct {
		Packets []Packet
		Frames  []Frame
		Error   string
	}

	p := func(kind Kind, id uint64, data string) Packet {
		return Packet{
			Data: []byte(data),
			ID:   ID{Stream: 1, Message: id},
			Kind: kind,
		}
	}

	f := func(kind Kind, id uint64, data string, done, control bool) Frame {
		return Frame{
			Data:    []byte(data),
			ID:      ID{Stream: 1, Message: id},
			Kind:    kind,
			Done:    done,
			Control: control,
		}
	}

	m := func(pkt Packet, frames ...Frame) testCase {
		return testCase{
			Packets: []Packet{pkt},
			Frames:  frames,
		}
	}

	megaFrames := make([]Frame, 0, 10*1024)
	for i := 0; i < 10*1024; i++ {
		megaFrames = append(megaFrames, f(KindMessage, 1, strings.Repeat("X", 1024), false, false))
	}
	megaFrames = append(megaFrames, f(KindMessage, 1, "", true, false))

	cases := []testCase{
		m(p(KindMessage, 1, "hello world"),
			f(KindMessage, 1, "hello", false, false),
			f(KindMessage, 1, " ", false, false),
			f(KindMessage, 1, "world", true, false)),

		m(p(KindClose, 2, ""),
			f(KindMessage, 1, "hello", false, false),
			f(KindMessage, 1, " ", false, false),
			f(KindClose, 2, "", true, false)),

		{
			Packets: []Packet{
				p(KindClose, 2, ""),
			},
			Frames: []Frame{
				f(KindMessage, 1, "1", false, false),
				f(KindClose, 2, "", true, false),
				f(KindMessage, 1, "1", true, false),
			},
			Error: "id monotonicity violation",
		},

		{ // a single frame that's too large
			Packets: []Packet{},
			Frames:  []Frame{f(KindMessage, 1, strings.Repeat("X", 4<<20+22), true, false)},
			Error:   "data overflow",
		},

		{ // multiple frames that make too large a packet
			Packets: []Packet{},
			Frames:  megaFrames,
			Error:   "data overflow",
		},

		{ // Control bit is ignored
			Packets: []Packet{
				p(KindClose, 2, ""),
			},
			Frames: []Frame{
				f(KindMessage, 1, "1", false, false),
				f(KindClose, 2, "", true, false),
				f(KindMessage, 1, "1", true, true),
			},
		},

		{ // packet kind changes
			Frames: []Frame{
				f(KindMessage, 1, "", false, false),
				f(KindClose, 1, "", false, false),
			},
			Error: "packet kind change",
		},

		{ // id monotonicity from id reuse
			Packets: []Packet{
				p(KindMessage, 1, "1"),
			},
			Frames: []Frame{
				f(KindMessage, 1, "1", true, false),
				f(KindMessage, 1, "2", true, false),
			},
			Error: "id monotonicity violation",
		},

		{ // message id zero is not allowed
			Frames: []Frame{{ID: ID{Stream: 1, Message: 0}}},
			Error:  "id monotonicity violation",
		},

		{ // stream id zero is not allowed
			Frames: []Frame{{ID: ID{Stream: 0, Message: 1}}},
			Error:  "id monotonicity violation",
		},
	}

	for _, tc := range cases {
		var buf []byte
		for _, fr := range tc.Frames {
			buf = AppendFrame(buf, fr)
		}

		rd := NewReader(bytes.NewReader(buf))
		for _, expPkt := range tc.Packets {
			pkt, err := rd.ReadPacket()
			assert.NoError(t, err)
			assert.DeepEqual(t, expPkt, pkt)
		}

		_, err := rd.ReadPacket()
		assert.Error(t, err)
		if tc.Error != "" {
			assert.That(t, strings.Contains(err.Error(), tc.Error))
		} else {
			assert.Equal(t, err, io.EOF)
		}
	}
}

func TestReaderRandomized(t *testing.T) {
	seed := time.Now().UnixNano()
	t.Log("seed:", seed)
	rng := rand.New(rand.NewSource(seed))

	// create a function to get a predefined sequence of bytes
	bid := 0
	get := func(n int) []byte {
		out := make([]byte, n)
		for i := range out {
			out[i] = byte(bid)
			bid++
		}
		return out
	}

	// construct a random sequence of frames of different sizes
	// to attempt to capture any bugs from buffer management
	var buf []byte

	mid := uint64(1)
	done := false
	for i := 0; i < 1000; i++ {
		buf = AppendFrame(buf, Frame{
			ID:   ID{Stream: 1, Message: mid},
			Data: get(rng.Intn(8192)),
			Done: done,
		})

		if done {
			mid++
		}

		done = rng.Intn(10) == 0
	}

	// read all of the packets back which should have the
	// exact sequence of bytes, so we reset bid to generate
	// the sequence again.
	bid = 0
	r := NewReader(bytes.NewBuffer(buf))
	for i := 1; ; i++ {
		pkt, err := r.ReadPacket()
		if errors.Is(err, io.EOF) {
			break
		}
		assert.NoError(t, err)
		assert.Equal(t, pkt.ID.Message, i)
		assert.Equal(t, pkt.Data, get(len(pkt.Data)))
	}
}
