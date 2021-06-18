// +build gofuzzbeta

// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcwire

import "testing"

func FuzzParseFrame(f *testing.F) {
	f.Add(AppendFrame(nil, Frame{}))
	f.Add(AppendFrame(nil, Frame{
		Data:    []byte("foo"),
		ID:      ID{1<<64 - 1, 1<<64 - 1},
		Kind:    7,
		Done:    true,
		Control: true,
	}))

	f.Fuzz(func(t *testing.T, data []byte) {
		_, _, _, _ = ParseFrame(data)
	})
}
