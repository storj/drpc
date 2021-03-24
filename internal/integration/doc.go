// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

// Package integration holds integration tests for drpc.
package integration

//go:generate protoc --go_out=paths=source_relative:service/. --go-drpc_out=paths=source_relative:service/. service.proto
//go:generate protoc --gogo_out=paths=source_relative:gogoservice/. --go-drpc_out=paths=source_relative,protolib=github.com/gogo/protobuf:gogoservice/. service.proto
//go:generate protoc --go_out=paths=source_relative:customservice/. --go-drpc_out=paths=source_relative,protolib=storj.io/drpc/internal/integration/customencoding:customservice/. service.proto
