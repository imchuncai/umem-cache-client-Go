// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2024, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package conn

import (
	"errors"
	"sync"

	"github.com/imchuncai/umem-cache-client-Go/umem_cache"
)

type FullDuplexConn struct {
	mu   sync.Mutex
	conn *umem_cache.Conn
	recv chan func()
}

func NewFullDuplex(address string, threadID uint32, version uint32) (*FullDuplexConn, error) {
	conn, err := umem_cache.Dial(address, threadID, version)
	if err != nil {
		return nil, err
	}

	c := &FullDuplexConn{conn: conn, recv: make(chan func(), 1024)}
	go func() {
		for f := range c.recv {
			f()
		}
	}()
	return c, nil
}

func (c *FullDuplexConn) isClosed() bool {
	return c.conn == nil
}

func (c *FullDuplexConn) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isClosed() {
		c.conn.Close()
		c.conn = nil
		close(c.recv)
	}
}

func (c *FullDuplexConn) addCMD(send func() error, recv func()) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isClosed() {
		return errors.New("connection is closed")
	}

	err := send()
	if err != nil {
		return err
	}

	c.recv <- recv
	return nil
}

func (c *FullDuplexConn) Get(key string, nonBlock bool) umem_cache.GetResp {
	channel := make(chan umem_cache.GetResp, 1)
	err := c.addCMD(
		func() error {
			if nonBlock {
				return c.conn.GetSend(key, umem_cache.GET_FLAG_NON_BLOCK)
			} else {
				return c.conn.GetSend(key, 0)
			}
		},
		func() { channel <- c.conn.GetRecv() },
	)
	if err != nil {
		return umem_cache.GetResp{Err: err}
	}
	return <-channel
}

func (c *FullDuplexConn) Set(key string, val []byte, nonBlock bool) umem_cache.SetResp {
	channel := make(chan umem_cache.SetResp, 1)
	err := c.addCMD(
		func() error {
			if nonBlock {
				return c.conn.SetSend(key, val, umem_cache.SET_FLAG_NON_BLOCK)
			} else {
				return c.conn.SetSend(key, val, 0)
			}
		},
		func() { channel <- c.conn.SetRecv() },
	)
	if err != nil {
		return umem_cache.SetResp{Err: err}
	}
	return <-channel
}

func (c *FullDuplexConn) Del(key string, nonBlock bool) umem_cache.DelResp {
	channel := make(chan umem_cache.DelResp, 1)
	err := c.addCMD(
		func() error {
			if nonBlock {
				return c.conn.DelSend(key, umem_cache.DEL_FLAG_NON_BLOCK)
			} else {
				return c.conn.DelSend(key, 0)
			}
		},
		func() { channel <- c.conn.DelRecv() },
	)
	if err != nil {
		return umem_cache.DelResp{Err: err}
	}
	return <-channel
}
