// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcwire

import (
	"bytes"
	"testing"

	"github.com/zeebo/assert"
)

func TestWriter(t *testing.T) {
	run := func(size int) func(t *testing.T) {
		return func(t *testing.T) {
			var exp []byte
			var got bytes.Buffer

			wr := NewWriter(&got, size)
			for i := 0; i < 1000; i++ {
				fr := RandFrame()
				exp = AppendFrame(exp, fr)
				assert.NoError(t, wr.WriteFrame(fr))
			}
			assert.NoError(t, wr.Flush())
			assert.That(t, bytes.Equal(exp, got.Bytes()))
		}
	}

	t.Run("Size 0B", run(0))
	t.Run("Size 1MB", run(1024*1024))
}
