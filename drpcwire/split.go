// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcwire

func SplitN(pkt Packet, n int, cb func(fr Frame) error) error {
	switch {
	case n == 0:
		n = 1024
	case n < 0:
		n = 0
	}

	for {
		fr := Frame{
			Data: pkt.Data,
			ID:   pkt.ID,
			Kind: pkt.Kind,
			Done: true,
		}
		if len(pkt.Data) > n && n > 0 {
			fr.Data, pkt.Data = pkt.Data[:n], pkt.Data[n:]
			fr.Done = false
		}
		if err := cb(fr); err != nil {
			return err
		}
		if fr.Done {
			return nil
		}
	}
}
