// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcmetadata

import (
	"context"

	proto "github.com/gogo/protobuf/proto"
	"github.com/zeebo/errs"
	"storj.io/drpc/drpcmetadata/invoke"
)

// Metadata is a mapping from metadata key to value.
type Metadata map[string]string

// New generates a new Metadata instance with keys set to lowercase.
func New(data map[string]string) Metadata {
	md := Metadata{}
	for k, val := range data {
		md[key] = val
	}
	return md
}

// AddPairs attaches metadata onto a context and return the context.
func (md Metadata) AddPairs(ctx context.Context) context.Context {
	for key, val := range md {
		ctx = Add(ctx, key, val)
	}

	return ctx
}

// Encode generates byte form of the metadata and appends it onto the passed in buffer.
func (md Metadata) Encode(buffer []byte) ([]byte, error) {
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
func Decode(data []byte) (*ppb.InvokeMetadata, error) {
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
		metadata = make(Metadata)
		ctx = context.WithValue(ctx, metadataKey{}, metadata)
	}
	metadata[k] = value
	return ctx
}

// Get returns all key/value pairs on the given context.
func Get(ctx context.Context) (Metadata, bool) {
	metadata, ok := ctx.Value(metadataKey{}).(Metadata)
	return metadata, ok
}
