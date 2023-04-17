// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcpool

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/zeebo/errs"

	"storj.io/drpc/drpcdebug"
)

// Options contains the options to configure a pool.
type Options struct {
	// Expiration will remove any values from the Pool after the
	// value passes. Zero means no expiration.
	Expiration time.Duration

	// Capacity is the maximum number of values the Pool can store.
	// Zero means unlimited. Negative means no values.
	Capacity int

	// KeyCapacity is like Capacity except it is per key. Zero means
	// the Pool holds unlimited for any single key. Negative means
	// no values for any single key.
	KeyCapacity int
}

// Pool is a connection pool with key type K. It maintains a cache of connections
// per key and ensures the total number of connections in the cache is bounded by
// configurable values. It does not limit the maximum concurrency of the number
// of connections either in total or per key.
type Pool[K comparable, V Conn] struct {
	opts    Options
	mu      sync.Mutex
	entries map[K]*list[K, V]
	order   list[K, V]
}

// New constructs a new Pool with the provided Options.
func New[K comparable, V Conn](opts Options) *Pool[K, V] {
	return &Pool[K, V]{
		opts:    opts,
		entries: make(map[K]*list[K, V]),
	}
}

func (p *Pool[K, V]) log(what string, cb func() string) {
	if drpcdebug.Enabled {
		drpcdebug.Log(func() (_, _, _ string) { return fmt.Sprintf("<pül %p>", p), what, cb() })
	}
}

// Close evicts all entries from the Pool's cache, closing them and returning all
// of the combined errors from closing.
func (p *Pool[K, V]) Close() (err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	var eg errs.Group
	for ent := p.order.head; ent != nil; ent = ent.global.next {
		eg.Add(p.closeEntry(ent))
	}

	p.entries = make(map[K]*list[K, V])
	p.order = list[K, V]{}

	return eg.Err()
}

// Get returns a new Conn that will use the provided dial function to create an
// underlying conn to be cached by the Pool when Conn methods are invoked. It will
// share any cached connections with other conns that use the same key.
func (p *Pool[K, V]) Get(ctx context.Context, key K,
	dial func(ctx context.Context, key K) (V, error)) Conn {
	return &poolConn[K, V]{
		key:  key,
		pool: p,
		dial: dial,
	}
}

//
// helpers
//

func (p *Pool[K, V]) removeEntry(ent *entry[K, V]) {
	p.mu.Lock()
	defer p.mu.Unlock()

	local := p.entries[ent.key]
	if local == nil {
		return
	}

	local.removeEntry(ent, (*entry[K, V]).localList)
	p.order.removeEntry(ent, (*entry[K, V]).globalList)

	if local.count == 0 {
		delete(p.entries, ent.key)
	}
}

// closeEntry ensures the timer and connection are closed, returning any errors.
func (p *Pool[K, V]) closeEntry(ent *entry[K, V]) error {
	p.log("CLOSE", ent.String)

	if ent.exp == nil || ent.exp.Stop() {
		return ent.val.Close()
	}
	return nil
}

// Take acquires a value from the cache if one exists. It returns
// the zero value for V and false if one does not.
func (p *Pool[K, V]) Take(key K) (V, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	local := p.entries[key]
	if local == nil {
		return *new(V), false
	}

	// N.B. this loop depends on the fact that removing an entry from
	// the list does not modify the entry's next pointer. a removed
	// entry still points into the list, but the things that it points
	// at no longer point at it.
	for ent := local.head; ent != nil; ent = ent.local.next {
		if !closed(ent.val.Unblocked()) {
			continue
		}

		local.removeEntry(ent, (*entry[K, V]).localList)
		p.order.removeEntry(ent, (*entry[K, V]).globalList)

		if ent.exp != nil && !ent.exp.Stop() {
			continue
		} else if closed(ent.val.Closed()) {
			continue
		}

		p.log("TAKEN", ent.String)
		return ent.val, true
	}

	return *new(V), false
}

// Put places the connection in to the cache with the provided key, ensuring
// that the size limits the Pool is configured with are respected.
func (p *Pool[K, V]) Put(key K, val V) {
	if p.opts.Capacity < 0 || p.opts.KeyCapacity < 0 {
		_ = val.Close()
		return
	} else if closed(val.Closed()) {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	local := p.entries[key]
	if local == nil {
		local = new(list[K, V])
		p.entries[key] = local
	}

	for p.opts.KeyCapacity != 0 && local.count >= p.opts.KeyCapacity {
		ent := local.head

		_ = p.closeEntry(ent)

		local.removeEntry(ent, (*entry[K, V]).localList)
		p.order.removeEntry(ent, (*entry[K, V]).globalList)
	}

	for p.opts.Capacity != 0 && p.order.count >= p.opts.Capacity {
		ent := p.order.head
		local := p.entries[ent.key]

		_ = p.closeEntry(ent)

		local.removeEntry(ent, (*entry[K, V]).localList)
		p.order.removeEntry(ent, (*entry[K, V]).globalList)

		if local.count == 0 {
			delete(p.entries, ent.key)
		}
	}

	ent := &entry[K, V]{key: key, val: val}
	local.appendEntry(ent, (*entry[K, V]).localList)
	p.order.appendEntry(ent, (*entry[K, V]).globalList)
	p.log("PUT", ent.String)

	if p.opts.Expiration > 0 {
		ent.exp = time.AfterFunc(p.opts.Expiration, func() {
			_ = val.Close()
			p.removeEntry(ent)
		})
	}
}
