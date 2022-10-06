// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcpool

import (
	"context"
	"sync"
	"testing"

	"github.com/zeebo/assert"

	"storj.io/drpc"
)

func TestPoolReuse(t *testing.T) {
	ctx := context.Background()

	p := New(Options{
		Capacity:    2,
		KeyCapacity: 1,
		Expiration:  0,
	})
	defer func() { _ = p.Close() }()

	count := 0
	dial := func(ctx context.Context, key interface{}) (drpc.Conn, error) {
		count++
		return noopConn{}, nil
	}
	check := func(c drpc.Conn, e int) {
		_ = c.Invoke(ctx, "", nil, nil, nil)
		assert.Equal(t, count, e)
	}

	c1 := p.Get(ctx, "key1", dial)
	c2 := p.Get(ctx, "key2", dial)
	c3 := p.Get(ctx, "key3", dial)
	assert.Equal(t, count, 0) // lazily dial

	check(c1, 1) // c1's first invoke dials
	check(c1, 1) // c1 reuses the connection
	check(c2, 2) // c2's first invoke dials
	check(c2, 2) // c2 reuses the connection
	check(c1, 2) // c1 still reuses the connection
	check(c3, 3) // c3's first invoke dials
	check(c1, 3) // c1 has not been evicted because it was used most recently
	check(c2, 4) // c2 was evicted so it needs another dial
}

type noopConn struct{ drpc.Conn }

func (noopConn) Close() error            { return nil }
func (noopConn) Closed() <-chan struct{} { return nil }

func (noopConn) Invoke(ctx context.Context, rpc string, enc drpc.Encoding, in, out drpc.Message) error {
	return nil
}

func TestPoolConcurrency(t *testing.T) {
	ctx := context.Background()

	p := New(Options{
		Capacity:    2,
		KeyCapacity: 1,
		Expiration:  0,
	})
	defer func() { _ = p.Close() }()

	count := 0
	uc1 := new(streamConn)
	uc2 := new(streamConn)
	dial := func(ctx context.Context, key interface{}) (drpc.Conn, error) {
		count++
		return map[string]drpc.Conn{"key1": uc1, "key2": uc2}[key.(string)], nil
	}

	c1 := p.Get(ctx, "key1", dial)
	c2 := p.Get(ctx, "key2", dial)

	// ensure we can open multiple concurrent streams to the same destination by dialing more.
	s1_0, err := c1.NewStream(ctx, "", nil)
	assert.NoError(t, err)
	assert.Equal(t, count, 1)

	s1_1, err := c1.NewStream(ctx, "", nil)
	assert.NoError(t, err)
	assert.Equal(t, count, 2)

	s1_2, err := c1.NewStream(ctx, "", nil)
	assert.NoError(t, err)
	assert.Equal(t, count, 3)

	// ensure we can open multiple concurrent streams to other destinations.
	s2_0, err := c2.NewStream(ctx, "", nil)
	assert.NoError(t, err)
	assert.Equal(t, count, 4)

	// close the stream and wait for it to be replaced.
	uc2.CloseStream(0)
	<-s2_0.Context().Done()

	// ensure that it was replaced and that making a new stream does not dial.
	s2_1, err := c2.NewStream(ctx, "", nil)
	assert.NoError(t, err)
	assert.Equal(t, count, 4)

	// close all of the concurrent streams and wait for them to be replaced.
	uc2.CloseStream(1)
	<-s2_1.Context().Done()
	uc1.CloseStream(0)
	<-s1_0.Context().Done()
	uc1.CloseStream(1)
	<-s1_1.Context().Done()
	uc1.CloseStream(2)
	<-s1_2.Context().Done()

	// ensure that it was replaced and that making a new stream does not dial.
	s1_3, err := c1.NewStream(ctx, "", nil)
	assert.NoError(t, err)
	assert.Equal(t, count, 4)

	// clean up.
	uc1.CloseStream(3)
	<-s1_3.Context().Done()
}

type streamConn struct {
	drpc.Conn
	mu      sync.Mutex
	streams []*streamConnStream
}

func (*streamConn) Close() error            { return nil }
func (*streamConn) Closed() <-chan struct{} { return nil }

func (s *streamConn) NewStream(ctx context.Context, rpc string, enc drpc.Encoding) (drpc.Stream, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx, cancel := context.WithCancel(ctx)
	s.streams = append(s.streams, &streamConnStream{ctx: ctx, cancel: cancel})
	return s.streams[len(s.streams)-1], nil
}

func (s *streamConn) CloseStream(n int) { s.streams[n].cancel() }

type streamConnStream struct {
	drpc.Stream
	ctx    context.Context
	cancel func()
}

func (s *streamConnStream) Context() context.Context { return s.ctx }
