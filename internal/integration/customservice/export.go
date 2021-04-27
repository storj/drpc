// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

// Package service is a testing service for the integration tests.
package service

import (
	"google.golang.org/protobuf/proto"

	"storj.io/drpc"
)

// Equal returns true if the two messages are equal.
func Equal(a, b drpc.Message) bool { return proto.Equal(a.(proto.Message), b.(proto.Message)) }

// Encoding is the drpc.Encoding used for this service.
var Encoding drpcEncoding_File_service_proto
