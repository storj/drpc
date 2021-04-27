// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package drpchttp

import (
	"encoding/json"

	"storj.io/drpc"
)

// JSONMarshal looks for a JSONMarshal method on the encoding and calls that if it
// exists. Otherwise, it does a normal message marshal before doing a JSON marshal.
func JSONMarshal(msg drpc.Message, enc drpc.Encoding) ([]byte, error) {
	if enc, ok := enc.(interface {
		JSONMarshal(msg drpc.Message) ([]byte, error)
	}); ok {
		return enc.JSONMarshal(msg)
	}

	// fallback to normal Marshal + JSON Marshal
	buf, err := enc.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return json.Marshal(buf)
}

// JSONUnmarshal looks for a JSONUnmarshal method on the encoding and calls that
// if it exists. Otherwise, it JSON unmarshals the buf before doing a normal
// message unmarshal.
func JSONUnmarshal(buf []byte, msg drpc.Message, enc drpc.Encoding) error {
	if enc, ok := enc.(interface {
		JSONUnmarshal(buf []byte, msg drpc.Message) error
	}); ok {
		return enc.JSONUnmarshal(buf, msg)
	}

	// fallback to JSON Unmarshal + normal Unmarshal
	var data []byte
	if err := json.Unmarshal(buf, &data); err != nil {
		return err
	}
	return enc.Unmarshal(data, msg)
}
