// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcwire

import (
	"errors"
	"testing"

	"github.com/zeebo/assert"

	"storj.io/drpc/drpcerr"
)

func TestError(t *testing.T) {
	data := MarshalError(drpcerr.WithCode(errors.New("test"), 5))
	err := UnmarshalError(data)
	assert.Equal(t, drpcerr.Code(err), 5)
	assert.Equal(t, err.Error(), "test")
}
