// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

// Package customencoding is a testing custom encoding for the integration tests.
package customencoding

import (
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"storj.io/drpc"
)

// Marshal returns the encoded form of msg.
func Marshal(msg drpc.Message) ([]byte, error) {
	return proto.Marshal(msg.(proto.Message))
}

// Unmarshal reads the encoded form of some Message into msg.
// The buf is expected to contain only a single complete Message.
func Unmarshal(buf []byte, msg drpc.Message) error {
	return proto.Unmarshal(buf, msg.(proto.Message))
}

// JSONMarshal returns the json encoded form of msg.
func JSONMarshal(msg drpc.Message) ([]byte, error) {
	return protojson.Marshal(msg.(proto.Message))
}

// JSONUnmarshal reads the json encoded form of some Message into msg.
// The buf is expected to contain only a single complete Message.
func JSONUnmarshal(buf []byte, msg drpc.Message) error {
	return protojson.Unmarshal(buf, msg.(proto.Message))
}
