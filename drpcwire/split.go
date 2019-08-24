// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcwire

func SplitN(pkt Packet, n int, cb func(fr Frame) error) error {
	switch {
	case n == 0:
		n = 1024
	case n < 0:
		n = len(pkt.Data)
	}

	for {
		fr := Frame{
			Kind: pkt.Kind,
			Data: pkt.Data,
			Done: true,
		}
		if len(fr.Data) > n {
			fr.Data, pkt.Data = fr.Data[:n], pkt.Data[n:]
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
