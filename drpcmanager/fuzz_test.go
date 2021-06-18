// +build gofuzzbeta

// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcmanager

import "testing"

func FuzzClient(f *testing.F) {
	f.Fuzz(func(t *testing.T, prog []byte) {
		runRandomized(t, prog, new(randClient))
	})
}

func FuzzServer(f *testing.F) {
	f.Fuzz(func(t *testing.T, prog []byte) {
		runRandomized(t, prog, new(randServer))
	})
}
