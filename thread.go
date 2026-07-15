// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2025-2026, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package client

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/imchuncai/umem-cache-client-Go/proto"
)

type thread struct {
	route   string
	id      uint32
	config  *tls.Config
	tickets chan struct{} // nil for no limit

	mu        sync.Mutex
	idleConns []*proto.CacheConn // nil for closed
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
	t.idleConns = make([]*proto.CacheConn, 0, config.MaxConnsPerThread)
	if config.MaxConnsPerThread > 0 {
		t.tickets = make(chan struct{}, config.MaxConnsPerThread)
	}
}

func (t *thread) closed() bool {
	return t.idleConns == nil
}

func (t *thread) acquireTicket(deadline time.Time) error {
	if t.tickets == nil {
		return nil
	}

	select {
	case t.tickets <- struct{}{}:
		return nil
	default:
	}

	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	select {
	case t.tickets <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (t *thread) releaseTicket() {
	if t.tickets != nil {
		<-t.tickets
	}
}

func (t *thread) __dispatch() (*proto.CacheConn, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed() {
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

func (t *thread) dispatch(deadline time.Time) (*proto.CacheConn, error) {
	conn, err := t.__dispatch()
	if conn != nil && conn.SetDeadline(deadline) != nil {
		conn.Close()
		return nil, nil
	}
	return conn, err
}

func (t *thread) _return(c *proto.CacheConn) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed() {
		c.Close()
	} else {
		t.idleConns = append(t.idleConns, c)
	}
}

func (t *thread) __getOrSet(conn *proto.CacheConn, key []byte, get proto.FallbackGetFunc) (val []byte, err error) {
	val, err = conn.GetOrSet(key, get)
	if err == nil {
		t._return(conn)
	} else {
		conn.Close()
		if val != nil {
			err = nil
		}
	}
	return
}

func (t *thread) GetOrSet(deadline time.Time, key []byte, get proto.FallbackGetFunc) (val []byte, err error) {
	err = t.acquireTicket(deadline)
	if err != nil {
		return
	}
	defer t.releaseTicket()

	conn, err := t.dispatch(deadline)
	if err != nil {
		return nil, fmt.Errorf("dispatch failed: %w", err)
	}

	if conn != nil {
		val, err = t.__getOrSet(conn, key, get)
		if err == nil {
			return
		}
	}

	conn, err = proto.DialCache(deadline, t.route, t.id, t.config)
	if err != nil {
		return nil, fmt.Errorf("dial cache: %s %d failed: %w", t.route, t.id, err)
	}

	return t.__getOrSet(conn, key, get)
}

func (t *thread) __del(conn *proto.CacheConn, key []byte) error {
	err := conn.Del(key)
	if err == nil {
		t._return(conn)
	} else {
		conn.Close()
	}
	return err
}

func (t *thread) Del(deadline time.Time, key []byte) error {
	err := t.acquireTicket(deadline)
	if err != nil {
		return err
	}
	defer t.releaseTicket()

	conn, err := t.dispatch(deadline)
	if err != nil {
		return fmt.Errorf("dispatch failed: %w", err)
	}

	if conn != nil && t.__del(conn, key) == nil {
		return nil
	}

	conn, err = proto.DialCache(deadline, t.route, t.id, t.config)
	if err != nil {
		return fmt.Errorf("dial cache: %s %d failed: %w", t.route, t.id, err)
	}

	return t.__del(conn, key)
}

func (t *thread) Close() {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, conn := range t.idleConns {
		conn.Close()
	}
	t.idleConns = nil
}
