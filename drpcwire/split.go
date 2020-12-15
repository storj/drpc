// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcwire

import "context"

// SplitN splits the marshaled form of the Packet into a number of
// frames such that each frame is at most n bytes. It calls
// the callback with every such frame. If n is zero, a default of
// 1024 is used.
func SplitN(ctx context.Context, pkt Packet, n int, cb func(ctx context.Context, fr Frame) error) error {
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
		if err := cb(ctx, fr); err != nil {
			return err
		}
		if fr.Done {
			return nil
		}
	}
}
