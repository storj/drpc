// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcwire

import (
	"testing"

	"github.com/zeebo/assert"
)

func TestAppendParse(t *testing.T) {
	for i := 0; i < 1000; i++ {
		exp := RandFrame()
		rem, got, ok, err := ParseFrame(AppendFrame(nil, exp))
		assert.NoError(t, err)
		assert.That(t, ok)
		assert.Equal(t, len(rem), 0)
		assert.DeepEqual(t, got, exp)
	}
}
