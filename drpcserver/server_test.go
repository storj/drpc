// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcserver

import (
	"context"
	"net"
	"testing"

	"github.com/zeebo/assert"

	"storj.io/drpc/drpcctx"
)

func TestServerTemporarySleep(t *testing.T) {
	ctx := drpcctx.NewTracker(context.Background())
	defer ctx.Wait()
	defer ctx.Cancel()

	calls := 0
	l := listener(func() (net.Conn, error) {
		calls++
		switch calls {
		case 1:
		case 2:
			ctx.Cancel()
		default:
			panic("spinning on temporary error")
		}

		return nil, new(temporaryError)
	})

	assert.NoError(t, New(nil).Serve(ctx, l))
}

type listener func() (net.Conn, error)

func (l listener) Accept() (net.Conn, error) { return l() }
func (l listener) Close() error              { return nil }
func (l listener) Addr() net.Addr            { return nil }

type temporaryError struct{}

func (temporaryError) Error() string   { return "temporary error" }
func (temporaryError) Timeout() bool   { return false }
func (temporaryError) Temporary() bool { return true }
