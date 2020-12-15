// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcmetadata

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/zeebo/assert"

	"storj.io/drpc/drpcmetadata/invoke"
)

func TestEncode(t *testing.T) {
	t.Run("Empty Metadata", func(t *testing.T) {
		var metadata map[string]string
		buf, err := Encode(nil, metadata)
		assert.Nil(t, buf)
		assert.NoError(t, err)
	})

	t.Run("With Metadata", func(t *testing.T) {
		data, err := Encode(nil, map[string]string{
			"test1": "a",
			"test2": "b",
		})
		assert.NoError(t, err)
		assert.That(t, len(data) > 0)
	})
}

func TestDecode(t *testing.T) {
	t.Run("Empty Metadata", func(t *testing.T) {
		metadata, err := Decode(nil)
		assert.NoError(t, err)
		assert.Nil(t, metadata)
	})

	t.Run("With Metadata", func(t *testing.T) {
		msg := invoke.Metadata{
			Data: map[string]string{"test": "a"},
		}
		data, err := proto.Marshal(&msg)
		assert.NoError(t, err)
		metadata, err := Decode(data)
		assert.NoError(t, err)
		assert.DeepEqual(t, msg.Data, metadata)
	})
}
