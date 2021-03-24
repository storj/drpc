// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

// Package grpccompat holds compatibility tests for grpc.
package grpccompat

//go:generate protoc --go_out=paths=source_relative:. --go-grpc_out=paths=source_relative:. --go-drpc_out=paths=source_relative:. service.proto
