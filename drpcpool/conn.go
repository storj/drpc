// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcpool

import (
	"context"
	"sync"

	"github.com/zeebo/errs"

	"storj.io/drpc"
	"storj.io/drpc/drpcsignal"
)

// poolConn is a wrapper that asks a Pool for an underlying conn when necessary.
type poolConn struct {
	once sync.Once
	done chan struct{}
	key  interface{}
	pool *Pool
	dial func(context.Context, interface{}) (drpc.Conn, error)
}

// Close sets the poolConn to be in a closed state, inhibiting subsequent Invoke or NewStream
// calls.
func (p *poolConn) Close() error {
	p.once.Do(func() { close(p.done) })
	return nil
}

// Closed returns a channel that is closed after calls to Invoke and NewStream are inhibited.
func (p *poolConn) Closed() <-chan struct{} {
	return p.done
}

// Invoke grabs a temporary connection from the Pool, calls Invoke on that, and replaces the
// connection into the pool after.
func (p *poolConn) Invoke(ctx context.Context, rpc string, enc drpc.Encoding, in drpc.Message, out drpc.Message) (err error) {
	if closed(p.done) {
		return errs.New("connection closed")
	}

	conn := p.pool.take(p.key)
	if conn == nil {
		conn, err = p.dial(ctx, p.key)
		if err != nil {
			return err
		}
	}
	defer p.pool.put(p.key, conn)

	return conn.Invoke(ctx, rpc, enc, in, out)
}

// NewStream grabs a temporary connection from the Pool, calls NewStream on that, and returns
// that stream after setting up a goroutine to return the connection to the Pool after the
// stream is done. The stream is wrapped so that the returned stream's done channel is only
// closed after the underlying connection has been returned to the pool, allowing callers to
// be sure that a connection will be reused if possible.
func (p *poolConn) NewStream(ctx context.Context, rpc string, enc drpc.Encoding) (_ drpc.Stream, err error) {
	if closed(p.done) {
		return nil, errs.New("connection closed")
	}

	conn := p.pool.take(p.key)
	if conn == nil {
		conn, err = p.dial(ctx, p.key)
		if err != nil {
			return nil, err
		}
	}

	stream, err := conn.NewStream(ctx, rpc, enc)
	if err != nil {
		p.pool.put(p.key, conn)
		return nil, err
	}

	sw := &streamWrapper{Stream: stream}
	go p.monitorStream(stream, conn, &sw.ctx.sig)
	return sw, nil
}

func (p *poolConn) monitorStream(stream drpc.Stream, conn drpc.Conn, sig *drpcsignal.Signal) {
	<-stream.Context().Done()
	p.pool.put(p.key, conn)
	sig.Set(nil)
}

type streamWrapper struct {
	drpc.Stream
	ctx streamWrapperContext
}

type streamWrapperContext struct {
	context.Context
	sig drpcsignal.Signal
}

func (s *streamWrapper) Context() context.Context     { return &s.ctx }
func (s *streamWrapperContext) Done() <-chan struct{} { return s.sig.Signal() }
