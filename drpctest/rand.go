// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpctest

import (
	"math"
	"math/rand"
	"sync"

	"storj.io/drpc/drpcwire"
)

var (
	mu        sync.Mutex
	streamID  uint64 = 1
	messageID uint64 = 1
)

func RandPacketID() (sid, mid uint64) {
	mu.Lock()
	if rand.Intn(100) == 0 {
		streamID++
		messageID = 1
	} else {
		messageID++
	}
	sid, mid = streamID, messageID
	mu.Unlock()
	return sid, mid
}

func RandBytes(n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = byte(rand.Intn(256))
	}
	return out
}

func RandBool() bool {
	return rand.Intn(2) == 0
}

func RandUint64() uint64 {
	return uint64(rand.Int63n(math.MaxInt64))<<1 + uint64(rand.Intn(2))
}

func RandPacketKind() drpcwire.PacketKind {
	return drpcwire.PacketKind(rand.Intn(int(drpcwire.PacketKind_Largest)-1) + 1)
}

var payloadSize = map[drpcwire.PacketKind]func() int{
	drpcwire.PacketKind_Invoke:    func() int { return rand.Intn(1023) + 1 },
	drpcwire.PacketKind_Message:   func() int { return rand.Intn(1023) + 1 },
	drpcwire.PacketKind_Error:     func() int { return rand.Intn(1023) + 1 },
	drpcwire.PacketKind_Cancel:    func() int { return 0 },
	drpcwire.PacketKind_Close:     func() int { return 0 },
	drpcwire.PacketKind_CloseSend: func() int { return 0 },
}

func RandFrame() drpcwire.Frame {
	sid, mid := RandPacketID()
	kind := RandPacketKind()
	return drpcwire.Frame{
		StreamID:  sid,
		MessageID: mid,
		Done:      RandBool(),
		Kind:      kind,
		Data:      RandBytes(payloadSize[kind]()),
	}
}

func RandPacket() drpcwire.Packet {
	sid, mid := RandPacketID()
	kind := RandPacketKind()
	return drpcwire.Packet{
		StreamID:  sid,
		MessageID: mid,
		Kind:      kind,
		Data:      RandBytes(10 * payloadSize[kind]()),
	}
}
