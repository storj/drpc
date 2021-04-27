// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package drpchttp

import (
	"context"
	"testing"

	"github.com/zeebo/assert"

	"storj.io/drpc/drpcmetadata"
)

func TestBuildContext(t *testing.T) {
	{ // check all the edge cases
		ctx, err := buildContext(context.Background(), []string{
			"key1=val1",         // basic
			"key2_%3d=val2_%25", // encoded = and encoded %
			"key3",              // no equals
			"=val4",             // empty key
			"key5=",             // empty value
			"key6=val6",         // duplicate key
			"key6=val7",         // duplicate key
			"key8=foo=val8",     // multiple equals
			"key9=%3d%25%3D%25", // multiple escapes
		})
		assert.NoError(t, err)

		metadata, ok := drpcmetadata.Get(ctx)
		assert.That(t, ok)
		assert.DeepEqual(t, metadata, map[string]string{
			"key1":   "val1",
			"key2_=": "val2_%",
			"key3":   "",
			"":       "val4",
			"key5":   "",
			"key6":   "val7",
			"key8":   "foo=val8",
			"key9":   "=%=%",
		})
	}

	{ // no entries associates no metadata
		ctx, err := buildContext(context.Background(), nil)
		assert.NoError(t, err)
		_, ok := drpcmetadata.Get(ctx)
		assert.That(t, !ok)
	}

	// check error cases
	cases := []string{
		"key%",       // truncated escape in key
		"key=val%",   // truncated escape in value
		"key%z1=val", // invalid hex in key in first byte
		"key%1z=val", // invalid hex in key in second byte
		"key=val%z1", // invalid hex in value in first byte
		"key=val%1x", // invalid hex in value in second byte
	}
	for _, entry := range cases {
		_, err := buildContext(context.Background(), []string{entry})
		assert.Error(t, err)
	}
}
