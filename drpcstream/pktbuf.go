// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcstream

import (
	"sync"
)

type packetBuffer struct {
	err  error
	mu   sync.Mutex
	data chan []byte
}

func newPacketBuffer() packetBuffer {
	return packetBuffer{
		data: make(chan []byte),
	}
}

func (pb *packetBuffer) Close(err error) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	if pb.err == nil {
		pb.err = err
		close(pb.data)
	}
}

func (pb *packetBuffer) Put(data []byte) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	pb.data <- data
	<-pb.data
}

func (pb *packetBuffer) Get() ([]byte, error) {
	data, ok := <-pb.data
	if !ok {
		return nil, pb.err
	}
	return data, nil
}

func (pb *packetBuffer) Done() {
	pb.data <- nil
}
