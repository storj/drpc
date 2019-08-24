// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcwire_test

import (
	"testing"

	"github.com/zeebo/assert"

	"storj.io/drpc/drpctest"
	"storj.io/drpc/drpcwire"
)

func TestAppendParse(t *testing.T) {
	for i := 0; i < 1000; i++ {
		exp := drpctest.RandFrame()
		rem, got, ok, err := drpcwire.ParseFrame(drpcwire.AppendFrame(nil, exp))
		assert.NoError(t, err)
		assert.That(t, ok)
		assert.Equal(t, len(rem), 0)
		assert.DeepEqual(t, got, exp)
	}
}
