// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcwire

import (
	"bytes"
	"errors"
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

type partialWriter struct{ bytes.Buffer }

func (w *partialWriter) Write(p []byte) (int, error) {
	if w.Len() == 0 {
		w.Buffer.Write(p[:len(p)/2])
		return len(p) / 2, errors.New("partial")
	}
	return w.Buffer.Write(p)
}

// TestWriter_StickyError checks that nothing is written after a failed write,
// which could have left a partial frame that later bytes would complete.
func TestWriter_StickyError(t *testing.T) {
	var got partialWriter
	wr := NewWriter(&got, 0)

	assert.NoError(t, wr.WriteFrame(RandFrame()))
	assert.Error(t, wr.Flush())
	n := got.Len()

	assert.Error(t, wr.WriteFrame(RandFrame()))
	assert.Error(t, wr.Flush())
	assert.Equal(t, got.Len(), n)
}

// TestWriter_StickyErrorFull checks that the error from a write triggered by
// the buffer filling up is sticky too.
func TestWriter_StickyErrorFull(t *testing.T) {
	var got partialWriter
	wr := NewWriter(&got, 1)

	assert.Error(t, wr.WriteFrame(RandFrame()))
	n := got.Len()

	assert.Error(t, wr.WriteFrame(RandFrame()))
	assert.Error(t, wr.Flush())
	assert.Equal(t, got.Len(), n)
}
