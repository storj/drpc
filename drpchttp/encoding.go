// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package drpchttp

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"

	"github.com/zeebo/errs"

	"storj.io/drpc"
)

const maxSize = 4 << 20

type (
	marshalFunc   = func(msg drpc.Message, enc drpc.Encoding) ([]byte, error)
	unmarshalFunc = func(buf []byte, msg drpc.Message, enc drpc.Encoding) error
	writeFunc     = func(w io.Writer, buf []byte) error
	readFunc      = func(r io.Reader) ([]byte, error)
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

func protoMarshal(msg drpc.Message, enc drpc.Encoding) ([]byte, error) {
	return enc.Marshal(msg)
}

func protoUnmarshal(buf []byte, msg drpc.Message, enc drpc.Encoding) error {
	return enc.Unmarshal(buf, msg)
}

func normalWrite(w io.Writer, buf []byte) error {
	_, err := w.Write(buf)
	return err
}

func base64Write(wf writeFunc) writeFunc {
	return func(w io.Writer, buf []byte) error {
		tmp := make([]byte, base64.StdEncoding.EncodedLen(len(buf)))
		base64.StdEncoding.Encode(tmp, buf)
		return wf(w, tmp)
	}
}

func readExactly(r io.Reader, n uint64) ([]byte, error) {
	buf := make([]byte, n)
	_, err := io.ReadFull(r, buf)
	return buf, err
}

func grpcRead(r io.Reader) ([]byte, error) {
	if tmp, err := readExactly(r, 5); err != nil {
		return nil, err
	} else if size := binary.BigEndian.Uint32(tmp[1:5]); size > maxSize {
		return nil, errs.New("message too large")
	} else if data, err := readExactly(r, uint64(size)); errors.Is(err, io.EOF) {
		return nil, io.ErrUnexpectedEOF
	} else if err != nil {
		return nil, err
	} else {
		return data, nil
	}
}

func twirpRead(r io.Reader) ([]byte, error) {
	if data, err := io.ReadAll(io.LimitReader(r, maxSize)); err != nil {
		return nil, err
	} else if len(data) > maxSize {
		return nil, errs.New("message too large")
	} else {
		return data, nil
	}
}

func base64Read(rf readFunc) readFunc {
	return func(r io.Reader) ([]byte, error) {
		return rf(base64.NewDecoder(base64.StdEncoding, r))
	}
}
