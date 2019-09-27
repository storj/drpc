// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcwire

import "fmt"

//go:generate stringer -type=Kind -trimprefix=Kind_ -output=packet_string.go

type Kind uint8

const (
	Kind_Reserved Kind = iota

	Kind_Invoke    // body is rpc name
	Kind_Message   // body is message data
	Kind_Error     // body is error data
	Kind_Cancel    // body must be empty
	Kind_Close     // body must be empty
	Kind_CloseSend // body must be empty

	Kind_Largest
)

//
// packet id
//

type ID struct {
	Stream  uint64
	Message uint64
}

func (i ID) Less(j ID) bool {
	return i.Stream < j.Stream || (i.Stream == j.Stream && i.Message < j.Message)
}

func (i ID) String() string { return fmt.Sprintf("<%d,%d>", i.Stream, i.Message) }

//
// data frame
//

type Frame struct {
	Data []byte
	ID   ID
	Kind Kind
	Done bool
}

func ParseFrame(buf []byte) (rem []byte, fr Frame, ok bool, err error) {
	var length uint64
	var control byte
	if len(buf) < 4 {
		goto bad
	}

	rem, control = buf[1:], buf[0]
	fr.Done = control&1 > 0
	fr.Kind = Kind(control >> 1)
	rem, fr.ID.Stream, ok, err = ReadVarint(rem)
	if !ok || err != nil {
		goto bad
	}
	rem, fr.ID.Message, ok, err = ReadVarint(rem)
	if !ok || err != nil {
		goto bad
	}
	rem, length, ok, err = ReadVarint(rem)
	if !ok || err != nil || length > uint64(len(rem)) {
		goto bad
	}
	rem, fr.Data = rem[length:], rem[:length]

	return rem, fr, true, nil
bad:
	return buf, fr, false, err
}

func AppendFrame(buf []byte, fr Frame) []byte {
	control := byte(fr.Kind << 1)
	if fr.Done {
		control |= 1
	}

	out := buf
	out = append(out, control)
	out = AppendVarint(out, fr.ID.Stream)
	out = AppendVarint(out, fr.ID.Message)
	out = AppendVarint(out, uint64(len(fr.Data)))
	out = append(out, fr.Data...)
	return out
}

//
// packet
//

type Packet struct {
	Data []byte
	ID   ID
	Kind Kind
}

func (p Packet) String() string {
	return fmt.Sprintf("<s:%d m:%d kind:%s data:%d>",
		p.ID.Stream, p.ID.Message, p.Kind, len(p.Data))
}
