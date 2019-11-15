// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

// Package grpccompat holds compatibility tests for grpc.
package grpccompat

//go:generate bash -c "go install storj.io/drpc/cmd/protoc-gen-drpc && protoc --drpc_out=plugins=drpc+grpc:. service.proto"
