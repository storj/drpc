// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcstream

import (
	"sync"
)

type packetBuffer struct {
	mu   sync.Mutex
	cond sync.Cond
	data []byte
	set  bool
	err  error
}

func newPacketBuffer() *packetBuffer {
	pb := new(packetBuffer)
	pb.cond.L = &pb.mu
	return pb
}

func (p *packetBuffer) Close(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.err == nil {
		p.data = nil
		p.set = false
		p.err = err
		p.cond.Broadcast()
	}
}

func (p *packetBuffer) Put(data []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.err != nil {
		return
	}

	p.data = data
	p.set = true
	p.cond.Broadcast()

	for p.err == nil && p.set {
		p.cond.Wait()
	}
}

func (p *packetBuffer) Get() ([]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for p.err == nil && !p.set {
		p.cond.Wait()
	}

	return p.data, p.err
}

func (p *packetBuffer) Done() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.data = nil
	p.set = false
	p.cond.Broadcast()
}
