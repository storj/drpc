// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

// Package integration holds integration tests for drpc.
package integration

//go:generate protoc --go_out=service/. --go-drpc_out=service/. service.proto
//go:generate protoc --gogo_out=gogoservice/. --go-drpc_out=protolib=github.com/gogo/protobuf:gogoservice/. service.proto
//go:generate protoc --go_out=customservice/. --go-drpc_out=protolib=storj.io/drpc/internal/integration/customencoding:customservice/. service.proto
