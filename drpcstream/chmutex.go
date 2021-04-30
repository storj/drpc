// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcstream

import (
	"sync"

	"storj.io/drpc/drpcsignal"
)

type chMutex struct {
	on sync.Once
	ch drpcsignal.Chan
}

func (m *chMutex) init() { m.ch.Make(1) }

func (m *chMutex) Lock() {
	m.on.Do(m.init)
	m.ch.Send()
}

func (m *chMutex) TryLock() bool {
	m.on.Do(m.init)
	select {
	case m.ch.Get() <- struct{}{}:
		return true
	default:
		return false
	}
}

func (m *chMutex) Unlock() {
	m.on.Do(m.init)
	m.ch.Recv()
}

func (m *chMutex) Unlocked() bool {
	m.on.Do(m.init)
	return !m.ch.Full()
}
