// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

// Package drpctest provides test related helpers.
package drpctest

import (
	"context"
	"runtime"
	"sync"
	"testing"
)

// Tracker keeps track of launched goroutines with a context.
type Tracker struct {
	context.Context
	tb     testing.TB
	cancel func()
	wg     sync.WaitGroup
}

// NewTracker creates a new tracker that inspects the provided TB to see if
// tests have failed in any of its launched goroutines.
func NewTracker(tb testing.TB) *Tracker {
	ctx, cancel := context.WithCancel(context.Background())
	return &Tracker{
		Context: ctx,
		tb:      tb,
		cancel:  cancel,
	}
}

// Close cancels the context and waits for all of the goroutines started by Run
// to finish.
func (t *Tracker) Close() {
	t.Cancel()
	t.Wait()
}

// Run starts a goroutine running the callback with the tracker as the context.
func (t *Tracker) Run(cb func(ctx context.Context)) {
	t.wg.Add(1)
	go t.track(cb)
}

// track is a helper to call done on the waitgroup after the callback returns.
func (t *Tracker) track(cb func(ctx context.Context)) {
	defer func() {
		if t.tb.Failed() {
			t.cancel()
		}
		t.wg.Done()
	}()
	cb(t)
}

// Cancel cancels the tracker's context.
func (t *Tracker) Cancel() { t.cancel() }

// Wait blocks until all callbacks started with Run have exited.
func (t *Tracker) Wait() {
	t.wg.Wait()
	if t.tb.Failed() {
		runtime.Goexit()
	}
}
