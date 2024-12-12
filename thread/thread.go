// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2024, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package thread

import (
	"errors"
	"sync"
	"time"

	"github.com/imchuncai/umem-cache-client-Go/umem_cache"
)

type Thread struct {
	address string
	id      uint32
	version uint32

	mu    sync.Mutex
	conns []*umem_cache.Conn
}

func New(address string, id uint32, version uint32, timeout time.Duration) *Thread {
	return &Thread{
		address: address,
		id:      id,
		version: version,
		conns:   []*umem_cache.Conn{},
	}
}

func (t *Thread) Close() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.__closed() {
		for _, conn := range t.conns {
			conn.Close()
		}
		t.conns = nil
	}
}

func (t *Thread) __closed() bool {
	return t.conns == nil
}

// Note: caller should close the conn or return it back
func (t *Thread) Dispatch(timeout time.Duration) (*umem_cache.Conn, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.__closed() {
		return nil, errors.New("thread is closed")
	}

	if len(t.conns) == 0 {
		return umem_cache.Dial(t.address, t.id, t.version, timeout)
	}

	conn := t.conns[len(t.conns)-1]
	t.conns = t.conns[:len(t.conns)-1]
	return conn, nil
}

func (t *Thread) Return(conn *umem_cache.Conn) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.__closed() {
		conn.Close()
	} else {
		t.conns = append(t.conns, conn)
	}
}
