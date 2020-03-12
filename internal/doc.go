// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

// Package integration holds integration tests for drpc.
package internal

//go:generate bash -c "go install storj.io/drpc/cmd/protoc-gen-drpc && protoc --drpc_out=plugins=drpc:. invoke.proto"
