// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcwire

import (
	"bufio"
	"io"
	"sync"

	"storj.io/drpc"
)

//
// Writer
//

type Writer struct {
	w    io.Writer
	size int
	mu   sync.Mutex
	buf  []byte
}

func NewWriter(w io.Writer, size int) *Writer {
	return &Writer{
		w:    w,
		size: size,
		buf:  make([]byte, 0, size),
	}
}

func (b *Writer) WritePacket(pkt Packet) (err error) {
	return b.WriteFrame(Frame{
		Data: pkt.Data,
		ID:   pkt.ID,
		Kind: pkt.Kind,
		Done: true,
	})
}

func (b *Writer) WriteFrame(fr Frame) (err error) {
	b.mu.Lock()
	b.buf = AppendFrame(b.buf, fr)
	if len(b.buf) >= b.size {
		_, err = b.w.Write(b.buf)
		b.buf = b.buf[:0]
	}
	b.mu.Unlock()
	return err
}

func (b *Writer) Flush() (err error) {
	b.mu.Lock()
	if len(b.buf) > 0 {
		_, err = b.w.Write(b.buf)
		b.buf = b.buf[:0]
	}
	b.mu.Unlock()
	return err
}

//
// Reader
//

func SplitFrame(data []byte, atEOF bool) (int, []byte, error) {
	rem, _, ok, err := ParseFrame(data)
	switch advance := len(data) - len(rem); {
	case err != nil:
		return 0, nil, err
	case len(data) > 0 && !ok && atEOF:
		return 0, nil, drpc.ProtocolError.New("truncated frame")
	case !ok:
		return 0, nil, nil
	case advance < 0, len(data) < advance:
		return 0, nil, drpc.InternalError.New("scanner issue with advance value")
	default:
		return advance, data[:advance], nil
	}
}

type Reader struct {
	buf *bufio.Scanner
}

func NewReader(r io.Reader) *Reader {
	buf := bufio.NewScanner(r)
	buf.Buffer(make([]byte, 4<<10), 1<<20)
	buf.Split(SplitFrame)
	return &Reader{buf: buf}
}

func (s *Reader) ReadPacket() (pkt Packet, err error) {
	for s.buf.Scan() {
		rem, fr, ok, err := ParseFrame(s.buf.Bytes())
		switch {
		case err != nil:
			return pkt, err
		case !ok, len(rem) > 0:
			return pkt, drpc.InternalError.New("problem with scanner")
		case fr.ID.Less(pkt.ID):
			return pkt, drpc.ProtocolError.New("id monotonicity violation")
		case pkt.ID.Less(fr.ID):
			pkt = Packet{
				Data: pkt.Data[:0],
				ID:   fr.ID,
				Kind: fr.Kind,
			}
		case fr.Kind != pkt.Kind:
			return pkt, drpc.ProtocolError.New("packet kind change")
		}

		pkt.Data = append(pkt.Data, fr.Data...)
		if fr.Done {
			return pkt, nil
		}
	}
	if err := s.buf.Err(); err != nil {
		return pkt, err
	}
	return pkt, io.EOF
}
