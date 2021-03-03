// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

// Package grpccompat holds compatibility tests for grpc.
package grpccompat

//go:generate protoc --go_out=. --go-grpc_out=. --go-drpc_out=. service.proto
