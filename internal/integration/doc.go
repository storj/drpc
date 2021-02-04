// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

// Package integration holds integration tests for drpc.
package integration

//go:generate go install storj.io/drpc/cmd/protoc-gen-drpc
//go:generate protoc --drpc_out=plugins=drpc:. service.proto
