// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcwire

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/zeebo/assert"
)

func TestSplit(t *testing.T) {
	for i := 0; i < 1000; i++ {
		pkt, done, n := RandPacket(), false, rand.Intn(10)-1
		if size := rand.Intn(100); size < len(pkt.Data) {
			pkt.Data = pkt.Data[:size]
		}

		var buf []byte
		assert.NoError(t, SplitN(pkt, n, func(fr Frame) error {
			assert.That(t, !done)
			assert.That(t, len(fr.Data) <= n ||
				(n == -1 && len(pkt.Data) == len(fr.Data)) ||
				(n == 0 && len(fr.Data) <= 1024))
			assert.Equal(t, pkt.Kind, fr.Kind)
			done = fr.Done
			buf = append(buf, fr.Data...)
			return nil
		}))

		assert.That(t, done)
		assert.That(t, bytes.Equal(pkt.Data, buf))
	}
}
