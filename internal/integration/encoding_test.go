// Copyright (C) 2026 Storj Labs, Inc.
// See LICENSE for copying information.

package integration

import (
	"bytes"
	"testing"

	"github.com/zeebo/assert"

	"storj.io/drpc/drpcenc"
)

// TestMarshalAppend ensures the generated encoding appends to the buffer it is
// given rather than replacing it, so that callers can reuse buffers.
func TestMarshalAppend(t *testing.T) {
	msg := &In{In: 5, Data: data(1024)}

	exp, err := Encoding.Marshal(msg)
	assert.NoError(t, err)

	prefix := []byte("prefix")
	got, err := drpcenc.MarshalAppend(msg, Encoding, append([]byte(nil), prefix...))
	assert.NoError(t, err)
	assert.That(t, bytes.Equal(got[:len(prefix)], prefix))
	assert.That(t, bytes.Equal(got[len(prefix):], exp))

	// a grown buffer must be reusable for the next message without losing data
	buf := got[:0]
	got, err = drpcenc.MarshalAppend(msg, Encoding, buf)
	assert.NoError(t, err)
	assert.That(t, bytes.Equal(got, exp))
}

func BenchmarkMarshalAppend(b *testing.B) {
	msg := &In{In: 5, Data: data(64 << 10)}

	buf, err := drpcenc.MarshalAppend(msg, Encoding, nil)
	assert.NoError(b, err)

	b.SetBytes(int64(len(buf)))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf, err = drpcenc.MarshalAppend(msg, Encoding, buf[:0])
		if err != nil {
			b.Fatal(err)
		}
	}
}
