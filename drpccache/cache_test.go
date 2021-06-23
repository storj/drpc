// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package drpccache

import (
	"context"
	"testing"

	"github.com/zeebo/assert"
)

func TestWithContext(t *testing.T) {
	ctx := context.Background()
	assert.Nil(t, FromContext(ctx))

	cache := New()
	ctx = WithContext(ctx, cache)
	assert.Equal(t, cache, FromContext(ctx))
}

func TestLoad(t *testing.T) {
	cache := New()

	assert.Nil(t, cache.Load("key1"))
	assert.Nil(t, cache.Load("key2"))

	cache.Store("key1", "val1")

	assert.Equal(t, cache.Load("key1"), "val1")
	assert.Nil(t, cache.Load("key2"))

	cache.Store("key2", "val2")

	assert.Equal(t, cache.Load("key1"), "val1")
	assert.Equal(t, cache.Load("key2"), "val2")
}

func TestClear(t *testing.T) {
	cache := New()

	cache.Store("key1", "val1")
	cache.Store("key2", "val2")

	assert.Equal(t, cache.Load("key1"), "val1")
	assert.Equal(t, cache.Load("key2"), "val2")

	cache.Clear()

	assert.Nil(t, cache.Load("key1"))
	assert.Nil(t, cache.Load("key2"))
}

func TestLoadOrCreate(t *testing.T) {
	f := func(val interface{}) func() interface{} {
		return func() interface{} { return val }
	}

	cache := New()

	assert.Nil(t, cache.Load("key1"))
	assert.Nil(t, cache.Load("key2"))

	assert.Equal(t, cache.LoadOrCreate("key1", f("key1")), "key1")
	assert.Equal(t, cache.LoadOrCreate("key1", f("key2")), "key1")

	assert.Equal(t, cache.Load("key1"), "key1")
	assert.Nil(t, cache.Load("key2"))

	cache.LoadOrCreate("key1", func() interface{} { panic("called") })
}
