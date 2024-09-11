// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2024, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package conn

import (
	"sync"

	"github.com/imchuncai/umem-cache-client-Go/umem_cache"
)

type HalfDuplexConn struct {
	mu   sync.Mutex
	conn *umem_cache.Conn
}

func NewHalfDuplex(address string, threadID uint32, version uint32) (*HalfDuplexConn, error) {
	conn, err := umem_cache.Dial(address, threadID, version)
	if err != nil {
		return nil, err
	}
	return &HalfDuplexConn{conn: conn}, nil
}

func (c *HalfDuplexConn) Close() {
	c.conn.Close()
}

func (c *HalfDuplexConn) GetOrSet(key string, fallbackGet umem_cache.FallbackGetFunc, nonBlock bool) umem_cache.GetResp {
	c.mu.Lock()
	defer c.mu.Unlock()

	var resp umem_cache.GetResp
	if nonBlock {
		resp = c.conn.Get(key, umem_cache.GET_FLAG_SET_ON_MISS|umem_cache.GET_FLAG_NON_BLOCK)
	} else {
		resp = c.conn.Get(key, umem_cache.GET_FLAG_SET_ON_MISS)
	}
	if resp.Err != nil || (nonBlock && resp.WillBlock) || resp.Val != nil {
		return resp
	}
	return c.conn.GetSet(key, fallbackGet)
}

func (c *HalfDuplexConn) DelForSet(key string, fallbackGet umem_cache.FallbackGetFunc, nonBlock bool) umem_cache.DelResp {
	c.mu.Lock()
	defer c.mu.Unlock()

	var resp umem_cache.DelResp
	if nonBlock {
		resp = c.conn.Del(key, umem_cache.DEL_FLAG_SET|umem_cache.DEL_FLAG_NON_BLOCK)
	} else {
		resp = c.conn.Del(key, umem_cache.DEL_FLAG_SET)
	}
	if resp.Err != nil || (nonBlock && resp.WillBlock) {
		return resp
	}
	return c.conn.DelSet(key, fallbackGet)
}
