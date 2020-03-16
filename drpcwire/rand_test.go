// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcwire

import (
	"math"
	"math/rand"
	"sync"
)

var (
	mu        sync.Mutex
	streamID  uint64 = 1
	messageID uint64 = 1
)

func RandID() ID {
	mu.Lock()
	if rand.Intn(100) == 0 {
		streamID++
		messageID = 1
	} else {
		messageID++
	}
	id := ID{
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

func RandKind() Kind {
	for {
		kind := Kind(rand.Intn(7))
		if _, ok := payloadSize[kind]; ok {
			return kind
		}
	}
}

var payloadSize = map[Kind]func() int{
	KindInvoke:         func() int { return rand.Intn(1023) + 1 },
	KindMessage:        func() int { return rand.Intn(1023) + 1 },
	KindError:          func() int { return rand.Intn(1023) + 1 },
	KindClose:          func() int { return 0 },
	KindCloseSend:      func() int { return 0 },
	KindInvokeMetadata: func() int { return rand.Intn(1023) + 1 },
}

func RandFrame() Frame {
	kind := RandKind()
	return Frame{
		Data: RandBytes(payloadSize[kind]()),
		ID:   RandID(),
		Kind: kind,
		Done: RandBool(),
	}
}

func RandPacket() Packet {
	kind := RandKind()
	return Packet{
		Data: RandBytes(10 * payloadSize[kind]()),
		ID:   RandID(),
		Kind: kind,
	}
}
