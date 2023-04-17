// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcpool

import (
	"fmt"
	"time"
)

type entry[K comparable, V Conn] struct {
	key    K
	val    V
	exp    *time.Timer
	global node[K, V]
	local  node[K, V]
}

func (e *entry[K, V]) String() string {
	return fmt.Sprintf("<ent %p k:%v c:%v u:%v>",
		e, e.key, closed(e.val.Closed()), closed(e.val.Unblocked()))
}

type node[K comparable, V Conn] struct {
	next *entry[K, V]
	prev *entry[K, V]
}

type list[K comparable, V Conn] struct {
	head  *entry[K, V]
	tail  *entry[K, V]
	count int
}

func (e *entry[K, V]) globalList() *node[K, V] { return &e.global }
func (e *entry[K, V]) localList() *node[K, V]  { return &e.local }

func (l *list[K, V]) appendEntry(ent *entry[K, V], node func(*entry[K, V]) *node[K, V]) {
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

func (l *list[K, V]) removeEntry(ent *entry[K, V], node func(*entry[K, V]) *node[K, V]) {
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
