// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

// Package fuzz is used to fuzz drpcwire frame parsing.
package fuzz

import "storj.io/drpc/drpcwire"

// Fuzz takes in some data and attempts to parse it.
func Fuzz(data []byte) int {
	drpcwire.ParseFrame(data) //nolint
	return 0
}
