// Copyright (C) 2024 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcstats

import (
	"sync/atomic"
)

// Stats keeps counters of read and written bytes.
type Stats struct {
	Read    uint64
	Written uint64
}

// AddRead atomically adds n bytes to the Read counter.
func (s *Stats) AddRead(n uint64) {
	if s != nil {
		atomic.AddUint64(&s.Read, n)
	}
}

// AddWritten atomically adds n bytes to the Written counter.
func (s *Stats) AddWritten(n uint64) {
	if s != nil {
		atomic.AddUint64(&s.Written, n)
	}
}

// AtomicClone returns a copy of the stats that is safe to use concurrently with Add methods.
func (s *Stats) AtomicClone() Stats {
	return Stats{
		Read:    atomic.LoadUint64(&s.Read),
		Written: atomic.LoadUint64(&s.Written),
	}
}
