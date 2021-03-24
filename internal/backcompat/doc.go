// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

// Package backcompat holds backwards-compatibility tests for drpc.
package backcompat

// In order to generate oldservice, run `go get storj.io/drpc/cmd/protoc-gen-drpc@v0.0.17`
// and rename the resulting binary to be `protoc-gen-drpc17`. Then, execute the following
// command: protoc --drpc17_out=paths=source_relative,plugins=drpc:oldservicedefs/. servicedefs.proto

//go:generate protoc --go_out=paths=source_relative:newservicedefs/. --go-drpc_out=paths=source_relative:newservicedefs/. servicedefs.proto
