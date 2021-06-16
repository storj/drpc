// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcstream

import (
	"sync"
	"sync/atomic"
)

type inspectMutex struct {
	held uint32
	mu   sync.Mutex
}

func (m *inspectMutex) Lock() {
	m.mu.Lock()
	atomic.StoreUint32(&m.held, 1)
}

func (m *inspectMutex) Unlock() {
	atomic.StoreUint32(&m.held, 0)
	m.mu.Unlock()
}

func (m *inspectMutex) Unlocked() bool {
	return atomic.LoadUint32(&m.held) == 0
}
