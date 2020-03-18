// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcmetadata

import (
	"context"

	proto "github.com/gogo/protobuf/proto"
	"github.com/zeebo/errs"
	"storj.io/drpc/drpcmetadata/invoke"
)

// AddPairs attaches metadata onto a context and return the context.
func AddPairs(ctx context.Context, md map[string]string) context.Context {
	for key, val := range md {
		ctx = Add(ctx, key, val)
	}

	return ctx
}

// Encode generates byte form of the metadata and appends it onto the passed in buffer.
func Encode(buffer []byte, md map[string]string) ([]byte, error) {
	msg := invoke.InvokeMetadata{
		Data: md,
	}

	msgBytes, err := proto.Marshal(&msg)
	if err != nil {
		return buffer, errs.Wrap(err)
	}

	buffer = append(buffer, msgBytes...)

	return buffer, nil
}

// Decode translate byte form of metadata into metadata struct defined by protobuf.
func Decode(data []byte) (*invoke.InvokeMetadata, error) {
	msg := invoke.InvokeMetadata{}
	err := proto.Unmarshal(data, &msg)
	if err != nil {
		return nil, err
	}

	return &msg, nil
}

type metadataKey struct{}

// Add associates a key/value pair on the context.
func Add(ctx context.Context, key, value string) context.Context {
	metadata, ok := Get(ctx)
	if !ok {
		metadata = make(map[string]string)
		ctx = context.WithValue(ctx, metadataKey{}, metadata)
	}
	metadata[key] = value
	return ctx
}

// Get returns all key/value pairs on the given context.
func Get(ctx context.Context) (map[string]string, bool) {
	metadata, ok := ctx.Value(metadataKey{}).(map[string]string)
	return metadata, ok
}
