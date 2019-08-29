// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcerr

import (
	"errors"
	"testing"

	"github.com/zeebo/assert"
	"github.com/zeebo/errs"
)

func TestCode(t *testing.T) {
	assert.Nil(t, WithCode(nil, 5))
	assert.Equal(t, Code(WithCode(errors.New("test"), 5)), 5)
	assert.Equal(t, Code(errs.Wrap(WithCode(errors.New("test"), 5))), 5)
}
