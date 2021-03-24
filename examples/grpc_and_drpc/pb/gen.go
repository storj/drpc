// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

// Package pb includes protobufs for this example.
package pb

//go:generate protoc --go_out=paths=source_relative:. --go-grpc_out=paths=source_relative:. --go-drpc_out=paths=source_relative:. sesamestreet.proto
