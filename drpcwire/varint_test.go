// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcwire

import (
	"testing"

	"github.com/zeebo/assert"
)

func TestVarint(t *testing.T) {
	t.Run("Round Trip", func(t *testing.T) {
		var vals = []uint64{
			0, 1, 2,
			1<<7 - 1, 1 << 7, 1<<7 + 1,
			1<<14 - 1, 1 << 14, 1<<14 + 1,
			1<<21 - 1, 1 << 21, 1<<21 + 1,
			1<<28 - 1, 1 << 28, 1<<28 + 1,
			1<<35 - 1, 1 << 35, 1<<35 + 1,
			1<<42 - 1, 1 << 42, 1<<42 + 1,
			1<<49 - 1, 1 << 49, 1<<49 + 1,
			1<<56 - 1, 1 << 56, 1<<56 + 1,
			1<<63 - 1, 1 << 63, 1<<63 + 1,
			1<<64 - 1,
		}
		for i := 0; i < 64; i++ {
			// val has i+1 lower bits set
			vals = append(vals, (uint64(1)<<uint(i+1))-1)
		}

		for _, val := range vals {
			// the encoding should be related to the number of bits set
			buf := AppendVarint(nil, val)
			assert.Equal(t, varintSize(val), len(buf))

			// it should decode to the same value
			gotBuf, gotVal, ok, err := ReadVarint(buf)
			assert.NoError(t, err)
			assert.That(t, ok)
			assert.Equal(t, 0, len(gotBuf))
			assert.Equal(t, val, gotVal)
		}
	})

	t.Run("Round Trip Fuzz", func(t *testing.T) {
		for i := 0; i < 10000; i++ {
			val := RandUint64()
			buf := AppendVarint(nil, val)
			gotBuf, gotVal, ok, err := ReadVarint(buf)
			assert.NoError(t, err)
			assert.That(t, ok)
			assert.Equal(t, 0, len(gotBuf))
			assert.Equal(t, val, gotVal)
		}
	})
}
