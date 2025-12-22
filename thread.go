// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2025, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package client

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/imchuncai/umem-cache-raft-client-Go/proto"
	"golang.org/x/sync/semaphore"
)

type thread struct {
	route   string
	id      uint32
	config  *tls.Config
	tickets *semaphore.Weighted // nil for no limit

	mu        sync.Mutex
	closed    bool
	idleConns []*proto.CacheConn
}

func newThreads(route string, config Config) []thread {
	threads := make([]thread, config.ThreadNR)
	for i := range threads {
		threads[i].init(route, uint32(i), config)
	}
	return threads
}

func (t *thread) init(route string, id uint32, config Config) {
	t.route = route
	t.id = id
	t.config = config.TLSConfig
	if config.MaxConnsPerThread > 0 {
		t.idleConns = make([]*proto.CacheConn, 0, config.MaxConnsPerThread)
		t.tickets = semaphore.NewWeighted(int64(config.MaxConnsPerThread))
	}
}

func (t *thread) acquireTicket(deadline time.Time) error {
	if t.tickets == nil {
		return nil
	}

	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	return t.tickets.Acquire(ctx, 1)
}

func (t *thread) releaseTicket() {
	if t.tickets != nil {
		t.tickets.Release(1)
	}
}

func (t *thread) dispatch() (*proto.CacheConn, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil, errors.New("thread is close")
	}
	if len(t.idleConns) == 0 {
		return nil, nil
	}

	last := len(t.idleConns) - 1
	conn := t.idleConns[last]
	t.idleConns = t.idleConns[:last]
	return conn, nil
}

func (t *thread) _return(c *proto.CacheConn) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		c.Close()
	} else {
		t.idleConns = append(t.idleConns, c)
	}
}

func (t *thread) do(deadline time.Time, f func(conn *proto.CacheConn) error) error {
	err := t.acquireTicket(deadline)
	if err != nil {
		return err
	}
	defer t.releaseTicket()

	conn, err := t.dispatch()
	if err != nil {
		return err
	}
	if conn == nil {
		conn, err = proto.DialCache(deadline, t.route, t.id, t.config)
		if err != nil {
			return fmt.Errorf("dial cache: %s %d failed: %w", t.route, t.id, err)
		}
	}

	err = f(conn)
	if err != nil {
		conn.Close()
	} else {
		t._return(conn)
	}
	return err
}

func (t *thread) GetOrSet(deadline time.Time, key []byte, get proto.FallbackGetFunc) (val []byte, err error) {
	err = t.do(deadline, func(conn *proto.CacheConn) error {
		val, err = conn.GetOrSet(deadline, key, get)
		return err
	})
	if val != nil {
		return val, nil
	}
	return
}

func (t *thread) Del(deadline time.Time, key []byte) error {
	return t.do(deadline, func(conn *proto.CacheConn) error {
		return conn.Del(deadline, key)
	})
}

func (t *thread) Close() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.closed = true
	for _, conn := range t.idleConns {
		conn.Close()
	}
}
