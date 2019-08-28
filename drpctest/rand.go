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

func RandID() drpcwire.ID {
	mu.Lock()
	if rand.Intn(100) == 0 {
		streamID++
		messageID = 1
	} else {
		messageID++
	}
	id := drpcwire.ID{
		Stream:  streamID,
		Message: messageID,
	}
	mu.Unlock()
	return id
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

func RandKind() drpcwire.Kind {
	return drpcwire.Kind(rand.Intn(int(drpcwire.Kind_Largest)-1) + 1)
}

var payloadSize = map[drpcwire.Kind]func() int{
	drpcwire.Kind_Invoke:    func() int { return rand.Intn(1023) + 1 },
	drpcwire.Kind_Message:   func() int { return rand.Intn(1023) + 1 },
	drpcwire.Kind_Error:     func() int { return rand.Intn(1023) + 1 },
	drpcwire.Kind_Cancel:    func() int { return 0 },
	drpcwire.Kind_Close:     func() int { return 0 },
	drpcwire.Kind_CloseSend: func() int { return 0 },
}

func RandFrame() drpcwire.Frame {
	kind := RandKind()
	return drpcwire.Frame{
		Data: RandBytes(payloadSize[kind]()),
		ID:   RandID(),
		Kind: kind,
		Done: RandBool(),
	}
}

func RandPacket() drpcwire.Packet {
	kind := RandKind()
	return drpcwire.Packet{
		Data: RandBytes(10 * payloadSize[kind]()),
		ID:   RandID(),
		Kind: kind,
	}
}
