// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2024, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package mix_conn

import (
	"github.com/imchuncai/umem-cache-client-Go/conn"
	"github.com/imchuncai/umem-cache-client-Go/umem_cache"
)

type MixConn struct {
	Address  string
	ThreadID uint32
	Version  uint32

	fullNB *conn.FullDuplexConn
	full   *conn.FullDuplexConn
	halfNB *conn.HalfDuplexConn
	half   *conn.HalfDuplexConn
}

func New(address string, threadID uint32, version uint32) (*MixConn, error) {
	c := MixConn{Address: address, ThreadID: threadID, Version: version}
	var err error
	c.fullNB, err = conn.NewFullDuplex(address, threadID, version)
	if err == nil {
		c.full, err = conn.NewFullDuplex(address, threadID, version)
	}
	if err == nil {
		c.halfNB, err = conn.NewHalfDuplex(address, threadID, version)
	}
	if err == nil {
		c.half, err = conn.NewHalfDuplex(address, threadID, version)
	}
	if err == nil {
		return &c, nil
	}

	if c.fullNB != nil {
		c.fullNB.Close()
	}
	if c.full != nil {
		c.full.Close()
	}
	if c.halfNB != nil {
		c.halfNB.Close()
	}
	if c.half != nil {
		c.half.Close()
	}
	return nil, err
}

func (c *MixConn) Get(key string) (val []byte, err error) {
	resp := c.fullNB.Get(key, true)
	if resp.Err == nil && resp.WillBlock {
		resp = c.full.Get(key, false)
	}
	return resp.Val, resp.Err
}

func (c *MixConn) GetOrSet(key string, fallbackGet umem_cache.FallbackGetFunc) (val []byte, err error) {
	val, err = c.Get(key)
	if err != nil || val != nil {
		return val, err
	}

	resp := c.halfNB.GetOrSet(key, fallbackGet, true)
	if resp.Err == nil && resp.WillBlock {
		resp = c.half.GetOrSet(key, fallbackGet, false)
	}
	return resp.Val, resp.Err
}

func (c *MixConn) Set(key string, val []byte) error {
	resp := c.fullNB.Set(key, val, true)
	if resp.Err == nil && resp.WillBlock {
		resp = c.full.Set(key, val, false)
	}
	return resp.Err
}

func (c *MixConn) Del(key string) error {
	resp := c.fullNB.Del(key, true)
	return resp.Err
}

func (s *MixConn) Close() {
	s.fullNB.Close()
	s.full.Close()
	s.halfNB.Close()
	s.half.Close()
}
