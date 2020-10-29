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
	// no error should still be nil
	assert.Nil(t, WithCode(nil, 5))

	// no wrapping should be ok
	assert.Equal(t, Code(WithCode(errors.New("test"), 5)), 5)

	// one layer of wrapping that can be unwrapped should be ok
	assert.Equal(t, Code(errs.Wrap(WithCode(errors.New("test"), 5))), 5)

	// not implementing any interface should be ok
	assert.Equal(t, Code(errors.New("foo")), 0)

	// cycles should be handled ok
	assert.Equal(t, Code(cycle{}), 0)

	// uncomparable should be ok
	assert.Equal(t, Code(uncomparable{}), 0)

	// opaque should remove the code
	assert.Equal(t, Code(opaque{WithCode(errors.New("test"), 5)}), 0)
}

type cycle struct{}

func (s cycle) Error() string { return "cycle" }
func (s cycle) Unwrap() error { return s }

type uncomparable struct{ _ [0]func() }

func (u uncomparable) Error() string { return "uncomparable" }
func (u uncomparable) Unwrap() error { return u }

type opaque struct{ error }
