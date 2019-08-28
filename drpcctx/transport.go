// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcctx

import (
	"context"

	"storj.io/drpc"
)

type transportKey struct{}

func WithTransport(ctx context.Context, tr drpc.Transport) context.Context {
	return context.WithValue(ctx, transportKey{}, tr)
}

func Transport(ctx context.Context) (drpc.Transport, bool) {
	tr, ok := ctx.Value(transportKey{}).(drpc.Transport)
	return tr, ok
}
