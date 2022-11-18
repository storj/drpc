// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcpool

import (
	"context"
	"testing"
	"time"

	"github.com/zeebo/assert"

	"storj.io/drpc"
	"storj.io/drpc/drpctest"
)

func TestPoolReuse(t *testing.T) {
	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	pool := New(Options{
		Capacity:    2,
		KeyCapacity: 1,
	})
	defer func() { _ = pool.Close() }()

	count := 0
	dial := func(ctx context.Context, key interface{}) (Conn, error) {
		count++
		return new(callbackConn), nil
	}
	check := func(conn drpc.Conn, expected int) {
		t.Helper()
		_ = conn.Invoke(ctx, "", nil, nil, nil)
		assert.Equal(t, count, expected)
	}

	conn1 := pool.Get(ctx, "key1", dial)
	conn2 := pool.Get(ctx, "key2", dial)
	conn3 := pool.Get(ctx, "key3", dial)
	assert.Equal(t, count, 0) // lazily dial

	check(conn1, 1) // conn1's first invoke dials
	check(conn1, 1) // conn1 reuses the connection
	check(conn2, 2) // conn2's first invoke dials
	check(conn2, 2) // conn2 reuses the connection
	check(conn1, 2) // conn1 still reuses the connection
	check(conn3, 3) // conn3's first invoke dials
	check(conn1, 3) // conn1 has not been evicted because it was used most recently
	check(conn2, 4) // conn2 was evicted so it needs another dial
}

func TestPoolConcurrency(t *testing.T) {
	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	pool := New(Options{
		Capacity:    2,
		KeyCapacity: 1,
	})
	defer func() { _ = pool.Close() }()

	count := 0
	uc1 := new(callbackConn)
	uc2 := new(callbackConn)
	dial := func(ctx context.Context, key interface{}) (Conn, error) {
		count++
		return map[string]Conn{"key1": uc1, "key2": uc2}[key.(string)], nil
	}

	conn1 := pool.Get(ctx, "key1", dial)
	conn2 := pool.Get(ctx, "key2", dial)

	// ensure we can open multiple concurrent streams to the same destination by dialing more.
	stream1_1, err := conn1.NewStream(ctx, "", nil)
	assert.NoError(t, err)
	assert.Equal(t, count, 1)

	stream1_2, err := conn1.NewStream(ctx, "", nil)
	assert.NoError(t, err)
	assert.Equal(t, count, 2)

	stream1_3, err := conn1.NewStream(ctx, "", nil)
	assert.NoError(t, err)
	assert.Equal(t, count, 3)

	// ensure we can open multiple concurrent streams to other destinations.
	stream2_1, err := conn2.NewStream(ctx, "", nil)
	assert.NoError(t, err)
	assert.Equal(t, count, 4)

	// close the stream and wait for it to be replaced.
	_ = stream2_1.Close()
	<-stream2_1.Context().Done()

	// ensure that it was replaced and that making a new stream does not dial.
	stream2_2, err := conn2.NewStream(ctx, "", nil)
	assert.NoError(t, err)
	assert.Equal(t, count, 4)

	// close all of the concurrent streams and wait for them to be replaced.
	_ = stream2_2.Close()
	<-stream2_2.Context().Done()
	_ = stream1_1.Close()
	<-stream1_1.Context().Done()
	_ = stream1_2.Close()
	<-stream1_2.Context().Done()
	_ = stream1_3.Close()
	<-stream1_3.Context().Done()

	// ensure that it was replaced and that making a new stream does not dial.
	stream1_4, err := conn1.NewStream(ctx, "", nil)
	assert.NoError(t, err)
	assert.Equal(t, count, 4)

	// clean up.
	_ = stream1_4.Close()
	<-stream1_4.Context().Done()
}

// TestPool_Expiration checks that inserted entries expire eventually.
func TestPool_Expiration(t *testing.T) {
	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	closed := make(chan string, 1)
	pool := New(Options{Expiration: time.Nanosecond})
	defer func() { _ = pool.Close() }()

	useConn(ctx, pool, closed, "key")
	assert.Equal(t, <-closed, "key")
}

// TestPool_Stale checks that the stale predicate is called on Take.
func TestPool_Stale(t *testing.T) {
	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	calls := 0
	pool := New(Options{})
	defer func() { _ = pool.Close() }()

	conn := pool.Get(ctx, "key", func(ctx context.Context, key interface{}) (Conn, error) {
		calls++
		return &callbackConn{ClosedFn: func() <-chan struct{} { return closedCh }}, nil
	})

	// an invoke should cause a dial
	invoke(ctx, conn)
	assert.Equal(t, calls, 1)

	// another invoke should cause another dial because the conn is considered closed
	invoke(ctx, conn)
	assert.Equal(t, calls, 2)
}

// TestPool_Capacity checks that total capacity limits are enforced.
func TestPool_Capacity(t *testing.T) {
	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	closed := make(chan string, 1)
	pool := New(Options{Capacity: 1})
	defer func() { _ = pool.Close() }()

	// using key0 should remain in the pool
	useConn(ctx, pool, closed, "key0")
	assert.Equal(t, len(closed), 0)

	// using key1 should evict key0
	useConn(ctx, pool, closed, "key1")
	assert.Equal(t, len(closed), 1)
	assert.Equal(t, <-closed, "key0")

	// close the pool and key1 should be closed
	_ = pool.Close()
	assert.Equal(t, len(closed), 1)
	assert.Equal(t, <-closed, "key1")
}

// TestPool_Capacity_Expiration checks that capacity limits are enforced
// even if expiration is set.
func TestPool_Capacity_Expiration(t *testing.T) {
	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	closed := make(chan string, 1)
	pool := New(Options{
		Capacity:   1,
		Expiration: time.Hour,
	})
	defer func() { _ = pool.Close() }()

	// using key0 should remain in the pool
	useConn(ctx, pool, closed, "key0")
	assert.Equal(t, len(closed), 0)

	// using key1 should evict key0
	useConn(ctx, pool, closed, "key1")
	assert.Equal(t, len(closed), 1)
	assert.Equal(t, <-closed, "key0")

	// close the pool and key1 should be closed
	_ = pool.Close()
	assert.Equal(t, len(closed), 1)
	assert.Equal(t, <-closed, "key1")
}

// TestPool_Capacity_Negative checks that negative capacities cache nothing.
func TestPool_Capacity_Negative(t *testing.T) {
	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	closed := make(chan string, 1)
	pool := New(Options{Capacity: -1})
	defer func() { _ = pool.Close() }()

	useConn(ctx, pool, closed, "key0")
	assert.Equal(t, len(closed), 1)
	assert.Equal(t, <-closed, "key0")
}

// TestPool_KeyCapacity checks that per-key capacity limits are enforced.
func TestPool_KeyCapacity(t *testing.T) {
	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	closed := make(chan string, 2)
	pool := New(Options{KeyCapacity: 1})
	defer func() { _ = pool.Close() }()

	useConn(ctx, pool, closed, "key0")
	assert.Equal(t, len(closed), 0)

	useConn(ctx, pool, closed, "key1")
	assert.Equal(t, len(closed), 0)

	// get two concurrent streams so that we force two underlying dials
	// causing one to be evicted when it is closed.
	conn := getConn(ctx, pool, closed, "key0")
	stream1, _ := conn.NewStream(ctx, "", nil)
	stream2, _ := conn.NewStream(ctx, "", nil)

	_ = stream1.Close()
	<-stream1.Context().Done()
	_ = stream2.Close()
	<-stream2.Context().Done()

	assert.Equal(t, len(closed), 1)
	assert.Equal(t, <-closed, "key0")
}

// TestPool_KeyCapacity_Negative checks that negative per-key capacities cache nothing.
func TestPool_KeyCapacity_Negative(t *testing.T) {
	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	closed := make(chan string, 1)
	pool := New(Options{KeyCapacity: -1})
	defer func() { _ = pool.Close() }()

	useConn(ctx, pool, closed, "key0")
	assert.Equal(t, len(closed), 1)
	assert.Equal(t, <-closed, "key0")
}

func TestPool_Blocked(t *testing.T) {
	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	closed := make(chan string, 2)
	unblocked := make(chan struct{})
	pool := New(Options{Capacity: 2})
	defer func() { _ = pool.Close() }()

	conn1Calls := 0
	conn1Dials := 0
	conn1 := pool.Get(ctx, "key", func(ctx context.Context, key interface{}) (Conn, error) {
		conn1Dials++
		return &callbackConn{
			CloseFn:     func() error { closed <- "conn1"; return nil },
			UnblockedFn: func() <-chan struct{} { return unblocked },
			InvokeFn: func(ctx context.Context, rpc string, enc drpc.Encoding, in, out drpc.Message) error {
				conn1Calls++
				return nil
			},
		}, nil
	})

	conn2Calls := 0
	conn2Dials := 0
	conn2 := pool.Get(ctx, "key", func(ctx context.Context, key interface{}) (Conn, error) {
		conn2Dials++
		return &callbackConn{
			CloseFn: func() error { closed <- "conn2"; return nil },
			InvokeFn: func(ctx context.Context, rpc string, enc drpc.Encoding, in, out drpc.Message) error {
				conn2Calls++
				return nil
			},
		}, nil
	})

	// place a blocked conn in the pool
	invoke(ctx, conn1)
	assert.Equal(t, conn1Calls, 1)
	assert.Equal(t, conn1Dials, 1)
	assert.Equal(t, conn2Calls, 0)
	assert.Equal(t, conn2Dials, 0)
	assert.Equal(t, len(closed), 0)

	// place another blocked conn in the pool
	invoke(ctx, conn1)
	assert.Equal(t, conn1Calls, 2)
	assert.Equal(t, conn1Dials, 2)
	assert.Equal(t, conn2Calls, 0)
	assert.Equal(t, conn2Dials, 0)
	assert.Equal(t, len(closed), 0)

	// invoking with conn2 should cause a conn1 to be evicted
	invoke(ctx, conn2)
	assert.Equal(t, conn1Calls, 2)
	assert.Equal(t, conn1Dials, 2)
	assert.Equal(t, conn2Calls, 1)
	assert.Equal(t, conn2Dials, 1)
	assert.Equal(t, len(closed), 1)
	assert.Equal(t, <-closed, "conn1")

	// unblock conn1
	close(unblocked)

	// since conn1 is the oldest, invoking with it should work
	invoke(ctx, conn1)
	assert.Equal(t, conn1Calls, 3)
	assert.Equal(t, conn1Dials, 2)
	assert.Equal(t, conn2Calls, 1)
	assert.Equal(t, conn2Dials, 1)
	assert.Equal(t, len(closed), 0)
}

func TestPool_MultipleCachedReuse(t *testing.T) {
	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	pool := New(Options{KeyCapacity: 2})
	defer func() { _ = pool.Close() }()

	closeStream := func(st drpc.Stream) { _ = st.Close(); <-st.Context().Done() }
	closedConns := make(map[int]bool)
	dials := 0
	conn := pool.Get(ctx, "key", func(ctx context.Context, key interface{}) (Conn, error) {
		d := dials
		dials++
		return &callbackConn{
			ClosedFn: func() <-chan struct{} {
				if closedConns[d] {
					return closedCh
				}
				return nil
			},
		}, nil
	})

	// start two concurrent streams
	st1, err := conn.NewStream(ctx, "rpc", nil)
	assert.NoError(t, err)
	defer closeStream(st1)

	st2, err := conn.NewStream(ctx, "rpc", nil)
	assert.NoError(t, err)
	defer closeStream(st2)

	// ensure we dialed twice
	assert.Equal(t, dials, 2)

	// put both the dialed connections back into the pool
	closeStream(st1)
	closeStream(st2)

	// cause the first connection to be considered dead
	closedConns[0] = true

	// start a new stream
	st3, err := conn.NewStream(ctx, "rpc", nil)
	assert.NoError(t, err)
	defer closeStream(st3)

	// the new stream should have reused the second connection
	assert.Equal(t, dials, 2)

	// start a new concurrent stream
	st4, err := conn.NewStream(ctx, "rpc", nil)
	assert.NoError(t, err)
	defer closeStream(st4)

	// there should have been no free streams left
	assert.Equal(t, dials, 3)
}

func BenchmarkPool(b *testing.B) {
	ctx := drpctest.NewTracker(b)
	defer ctx.Close()

	const capacity = 1000

	pool := New(Options{Capacity: capacity})
	uc := new(callbackConn)
	conn := pool.Get(ctx, "key", func(ctx context.Context, key interface{}) (Conn, error) { return uc, nil })

	var streams []drpc.Stream
	for i := 0; i < capacity; i++ {
		stream, _ := conn.NewStream(ctx, "", nil)
		streams = append(streams, stream)
	}
	for _, stream := range streams {
		_ = stream.Close()
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		invoke(ctx, conn)
	}
}
