// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

// Package drpcpool is a simple connection pool for clients.
//
// It has the ability to maintain a cache of connections with a
// maximum size on both the total and per key basis. It also
// can expire cached connections if they have been inactive in
// the pool for long enough.
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
