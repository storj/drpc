// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

// Package drpcpool is a simple connection pool for clients.
//
// It has the ability to maintain a cache of connections with a
// maximum size on both the total and per key basis. It also
// can expire cached connections if they have been inactive in
// the pool for long enough.
//
// Implementation note: the cache has some methods that could
// potentially be quadratic in the worst case in the number of
// per cache key connections. Specifically, this worst case happens
// when there are many closed entries in the list of values. While
// we could do a single pass filtering closed entries, the logic is
// a bit harder to follow and ensure is correct. Instead we have a
// helper to remove a single entry from a list without knowing where
// it came from. Since we can possibly call that to remove every
// element from a list if they are all closed, it's quadratic in
// the maximum size of that list. Since this cache is intended to
// be used with small key capacities (like 5), the decision was made
// to accept that quadratic worst case for the benefit of having as
// simple an implementation as possible.
package drpcpool

// closed is a helper to check if a notification channel has been closed.
// It should not be called on channels that can have send operations
// performed on it.
func closed(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

// closedCh is an already closed channel.
var closedCh = func() chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}()
