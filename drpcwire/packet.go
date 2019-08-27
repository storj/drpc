// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcwire

import "fmt"

//go:generate stringer -type=PacketKind -trimprefix=PacketKind_ -output=packet_string.go

type PacketKind uint8

const (
	PacketKind_Reserved PacketKind = iota

	PacketKind_Invoke    // body is rpc name
	PacketKind_Message   // body is message data
	PacketKind_Error     // body is error data
	PacketKind_Cancel    // body must be empty
	PacketKind_Close     // body must be empty
	PacketKind_CloseSend // body must be empty

	PacketKind_Largest
)

type Packet struct {
	Data      []byte
	StreamID  uint64
	MessageID uint64
	Kind      PacketKind
}

func (p Packet) String() string {
	return fmt.Sprintf("<kind:%s data:%d>", p.Kind, len(p.Data))
}

type Frame struct {
	Data      []byte
	StreamID  uint64
	MessageID uint64
	Kind      PacketKind
	Done      bool
}

func ParseFrame(buf []byte) (rem []byte, fr Frame, ok bool, err error) {
	var length uint64
	var control byte
	if len(buf) < 3 {
		goto bad
	}

	rem, control = buf[1:], buf[0]
	fr.Done = control&1 > 0
	fr.Kind = PacketKind(control >> 1)
	rem, fr.StreamID, ok, err = ReadVarint(rem)
	if !ok || err != nil {
		goto bad
	}
	rem, fr.MessageID, ok, err = ReadVarint(rem)
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
	out = AppendVarint(out, fr.StreamID)
	out = AppendVarint(out, fr.MessageID)
	out = AppendVarint(out, uint64(len(fr.Data)))
	out = append(out, fr.Data...)
	return out

}
