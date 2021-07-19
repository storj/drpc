// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

// Package twirpcompat holds compatibility tests for Twirp.
package twirpcompat

//go:generate protoc --go_out=paths=source_relative:. --go-drpc_out=paths=source_relative:. --twirp_out=paths=source_relative:. clientcompat.proto
