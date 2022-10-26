// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcpool

import (
	"context"

	"storj.io/drpc"
)

type callbackConn struct {
	CloseFn     func() error
	ClosedFn    func() <-chan struct{}
	UnblockedFn func() <-chan struct{}
	InvokeFn    func(ctx context.Context, rpc string, enc drpc.Encoding, in, out drpc.Message) error
	NewStreamFn func(ctx context.Context, rpc string, enc drpc.Encoding) (drpc.Stream, error)
}

func (cb *callbackConn) Close() error {
	if cb.CloseFn != nil {
		return cb.CloseFn()
	}
	return nil
}

func (cb *callbackConn) Closed() <-chan struct{} {
	if cb.ClosedFn != nil {
		return cb.ClosedFn()
	}
	return nil
}

func (cb *callbackConn) Unblocked() <-chan struct{} {
	if cb.UnblockedFn != nil {
		return cb.UnblockedFn()
	}
	return closedCh
}

func (cb *callbackConn) Invoke(ctx context.Context, rpc string, enc drpc.Encoding, in, out drpc.Message) error {
	if cb.InvokeFn != nil {
		return cb.InvokeFn(ctx, rpc, enc, in, out)
	}
	return nil
}

func (cb *callbackConn) NewStream(ctx context.Context, rpc string, enc drpc.Encoding) (drpc.Stream, error) {
	if cb.NewStreamFn != nil {
		return cb.NewStreamFn(ctx, rpc, enc)
	}
	return newCallbackStream(ctx), nil
}

type callbackStream struct {
	drpc.Stream
	ctx    context.Context
	cancel func()
}

func newCallbackStream(ctx context.Context) *callbackStream {
	ctx, cancel := context.WithCancel(ctx)
	return &callbackStream{
		ctx:    ctx,
		cancel: cancel,
	}
}

func (cb *callbackStream) Close() error             { cb.cancel(); return nil }
func (cb *callbackStream) Context() context.Context { return cb.ctx }

// getConn is a helper to get a new conn from the pool that will send its key over
// the closed channel when it is closed.
func getConn(ctx context.Context, pool *Pool, closed chan string, key string) Conn {
	return pool.Get(ctx, key, func(ctx context.Context, _ interface{}) (Conn, error) {
		return &callbackConn{CloseFn: func() error { closed <- key; return nil }}, nil
	})
}

// useConn is a helper to get a new conn from the pool and use it, sending its
// key over the closed channel when it is closed.
func useConn(ctx context.Context, pool *Pool, closed chan string, key string) {
	conn := getConn(ctx, pool, closed, key)
	invoke(ctx, conn)
}

// invoke is a helper to invoke a dummy rpc on the conn.
func invoke(ctx context.Context, conn Conn) {
	_ = conn.Invoke(ctx, "", nil, nil, nil)
}
