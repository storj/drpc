// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package fuzz

import "storj.io/drpc/drpcwire"

func Fuzz(data []byte) int {
	drpcwire.ParseFrame(data)
	return 0
}
