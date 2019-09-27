// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcsignal

import (
	"sync"
	"sync/atomic"
)

type Signal struct {
	set uint32
	mu  sync.Mutex
	sig chan struct{}
	err error
}

func New() Signal {
	return Signal{
		sig: make(chan struct{}),
	}
}

func (s *Signal) Signal() chan struct{} {
	return s.sig
}

func (s *Signal) Set(err error) (ok bool) {
	if atomic.LoadUint32(&s.set) != 0 {
		return false
	}
	return s.setSlow(err)
}

func (s *Signal) setSlow(err error) (ok bool) {
	s.mu.Lock()
	if s.set == 0 {
		s.err = err
		atomic.StoreUint32(&s.set, 1)
		close(s.sig)
		ok = true
	}
	s.mu.Unlock()
	return ok
}

func (s *Signal) Get() (error, bool) {
	if atomic.LoadUint32(&s.set) != 0 {
		return s.err, true
	}
	return nil, false
}

func (s *Signal) IsSet() bool {
	return atomic.LoadUint32(&s.set) != 0
}

func (s *Signal) Err() error {
	if atomic.LoadUint32(&s.set) != 0 {
		return s.err
	}
	return nil
}
