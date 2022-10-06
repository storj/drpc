// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcpool

import (
	"context"
	"sync"
	"time"

	"github.com/zeebo/errs"

	"storj.io/drpc"
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
	//
	// Implementation note: The cache is potentially quadratic in the
	// size of this parameter, so it is intended for small values, like
	// 5 or so.
	KeyCapacity int
}

type entry struct {
	key interface{}
	val drpc.Conn
	exp *time.Timer
}

// Pool is a connection pool with key type K. It maintains a cache of connections
// per key and ensures the total number of connections in the cache is bounded by
// configurable values. It does not limit the maximum concurrency of the number
// of connections either in total or per key.
type Pool struct {
	opts    Options
	mu      sync.Mutex
	entries map[interface{}][]*entry
	order   []*entry
}

// New constructs a new Pool with the provided Options.
func New(opts Options) *Pool {
	return &Pool{
		opts:    opts,
		entries: make(map[interface{}][]*entry),
	}
}

// Close evicts all entries from the Pool's cache, closing them and returning all
// of the combined errors from closing.
func (p *Pool) Close() (err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	var eg errs.Group
	for _, entries := range p.entries {
		for _, ent := range entries {
			eg.Add(p.closeEntry(ent))
		}
	}

	p.entries = make(map[interface{}][]*entry)
	p.order = nil

	return eg.Err()
}

// Get returns a new drpc.Conn that will use the provided dial function to create an
// underlying conn to be cached by the Pool when Conn methods are invoked. It will
// share any cached connections with other conns that use the same key.
func (p *Pool) Get(ctx context.Context, key interface{},
	dial func(ctx context.Context, key interface{}) (drpc.Conn, error)) drpc.Conn {
	return &poolConn{
		done: make(chan struct{}),
		key:  key,
		pool: p,
		dial: dial,
	}
}

//
// helpers
//

// closeEntry ensures the timer and connection are closed, returning any errors.
func (p *Pool) closeEntry(ent *entry) error {
	if ent.exp == nil || ent.exp.Stop() {
		return ent.val.Close()
	}
	return nil
}

// filterEntry is a helper to remove a specific entry from a slice of entries.
func filterEntry(entries []*entry, ent *entry) []*entry {
	for i := range entries {
		if entries[i] == ent {
			copy(entries[i:], entries[i+1:])
			return entries[:len(entries)-1]
		}
	}
	return entries
}

// filterEntryLocked removes the entry from the map, deleting the
// map key if necessary.
//
// It should only be called with the mutex held.
func (p *Pool) filterEntryLocked(ent *entry) {
	entries := p.entries[ent.key]
	if len(entries) <= 1 {
		delete(p.entries, ent.key)
	} else {
		p.entries[ent.key] = filterEntry(entries, ent)
	}
	p.order = filterEntry(p.order, ent)
}

// filterCacheKey removes any closed or expired conns from the list
// of entries for the key, deleting the key from the entries map if
// necessary.
func (p *Pool) filterCacheKey(key interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, ent := range p.entries[key] {
		if closed(ent.val.Closed()) {
			p.filterEntryLocked(ent)
		}
	}
}

// firstCacheKeyEntryLocked returns the newest put entry for the
// given cache key.
//
// It should only be called with the mutex held.
func (p *Pool) firstCacheKeyEntryLocked(key interface{}) *entry {
	entries := p.entries[key]
	if len(entries) == 0 {
		return nil
	}
	return entries[len(entries)-1]
}

// oldestEntryLocked returns the oldest put entry from the Cache or nil
// if one does not exist.
//
// It should only be called with the mutex held.
func (p *Pool) oldestEntryLocked() *entry {
	if len(p.order) == 0 {
		return nil
	}
	return p.order[0]
}

// take acquires a value from the cache if one exists. It returns
// nil if one does not.
func (p *Pool) take(key interface{}) drpc.Conn {
	p.mu.Lock()
	defer p.mu.Unlock()

	for {
		ent := p.firstCacheKeyEntryLocked(key)
		if ent == nil {
			return nil
		}
		p.filterEntryLocked(ent)

		if ent.exp != nil && !ent.exp.Stop() {
			continue
		} else if closed(ent.val.Closed()) {
			continue
		}

		return ent.val
	}
}

// put places the connection in to the cache with the provided key, ensuring
// that the size limits the Pool is configured with are respected.
func (p *Pool) put(key interface{}, val drpc.Conn) {
	if p.opts.Capacity < 0 || p.opts.KeyCapacity < 0 {
		_ = val.Close()
		return
	} else if closed(val.Closed()) {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	for {
		entries := p.entries[key]
		if p.opts.KeyCapacity == 0 || len(entries) < p.opts.KeyCapacity {
			break
		}

		ent := entries[0]
		_ = p.closeEntry(ent)
		p.filterEntryLocked(ent)
	}

	for {
		if p.opts.Capacity == 0 || len(p.order) < p.opts.Capacity {
			break
		}

		ent := p.oldestEntryLocked()
		_ = p.closeEntry(ent)
		p.filterEntryLocked(ent)
	}

	ent := &entry{key: key, val: val}
	p.entries[key] = append(p.entries[key], ent)
	p.order = append(p.order, ent)

	if p.opts.Expiration > 0 {
		ent.exp = time.AfterFunc(p.opts.Expiration, func() {
			_ = val.Close()
			p.filterCacheKey(key)
		})
	}
}
