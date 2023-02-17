// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

//go:build !windows
// +build !windows

package drpcserver

import (
	"errors"
	"net"
)

// isTemporary checks if an error is temporary.
func isTemporary(err error) bool {
	var nErr net.Error
	if errors.As(err, &nErr) {
		//lint:ignore SA1019 while this is deprecated, there is no good replacement
		return nErr.Temporary()
	}

	return false
}
