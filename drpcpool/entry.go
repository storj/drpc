// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcpool

import (
	"fmt"
	"time"
)

type entry struct {
	key    interface{}
	val    Conn
	exp    *time.Timer
	global node
	local  node
}

func (e *entry) String() string {
	return fmt.Sprintf("<ent %p k:%v c:%v u:%v>",
		e, e.key, closed(e.val.Closed()), closed(e.val.Unblocked()))
}

type node struct {
	next *entry
	prev *entry
}

type list struct {
	head  *entry
	tail  *entry
	count int
}

func (e *entry) globalList() *node { return &e.global }
func (e *entry) localList() *node  { return &e.local }

func (l *list) appendEntry(ent *entry, node func(*entry) *node) {
	if l.head == nil {
		l.head = ent
	}
	if l.tail != nil {
		node(l.tail).next = ent
		node(ent).prev = l.tail
	}
	l.tail = ent
	l.count++
}

func (l *list) removeEntry(ent *entry, node func(*entry) *node) {
	n := node(ent)
	if l.head == ent {
		l.head = n.next
	}
	if n.next != nil {
		node(n.next).prev = n.prev
	}
	if l.tail == ent {
		l.tail = n.prev
	}
	if n.prev != nil {
		node(n.prev).next = n.next
	}
	l.count--
}
