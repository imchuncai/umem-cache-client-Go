// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2024, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package client

import (
	"errors"
	"hash"
	"hash/fnv"
	"sync"

	mix_conn "github.com/imchuncai/umem-cache-client-Go/mix_conn"
	"github.com/imchuncai/umem-cache-client-Go/umem_cache"
)

type Thread struct {
	Address  string
	ThreadID uint32
}

type Server struct {
	Version uint32
	Threads []Thread
}

type SyncService interface {
	GetService() (Server, error)
}

type Client struct {
	mu          sync.Mutex
	syncService SyncService
	version     uint32
	hasher      hash.Hash32
	threads     []*mix_conn.MixConn
}

func close(threads []*mix_conn.MixConn) {
	for _, conn := range threads {
		conn.Close()
	}
}

func connect(s Server) ([]*mix_conn.MixConn, error) {
	if len(s.Threads) == 0 {
		return nil, errors.New("empty server")
	}

	conns := make([]*mix_conn.MixConn, len(s.Threads))
	for i, thread := range s.Threads {
		var err error
		conns[i], err = mix_conn.New(thread.Address, thread.ThreadID, s.Version)
		if err != nil {
			close(conns[:i])
			return nil, err
		}
	}
	return conns, nil
}

func (c *Client) __connect(s Server) error {
	conns, err := connect(s)
	if err != nil {
		return err
	}

	c.version = s.Version
	c.threads = conns
	return nil
}

// Note: we will establish 4 tcp connections to each thread
func New(s Server, syncService SyncService) (*Client, error) {
	if len(s.Threads) == 0 {
		return nil, errors.New("empty threads")
	}

	c := Client{hasher: fnv.New32a(), syncService: syncService}
	err := c.__connect(s)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Client) __closed() bool {
	return c.threads == nil
}

func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.__closed() {
		close(c.threads)
		c.threads = nil
	}
}

func (c *Client) __connIndex(key string) int {
	c.hasher.Reset()
	c.hasher.Write([]byte(key))
	return int(c.hasher.Sum32()) % len(c.threads)
}

func (c *Client) __dispatch(key string) *mix_conn.MixConn {
	i := c.__connIndex(key)
	return c.threads[i]
}

func (c *Client) dispatch(key string) (*mix_conn.MixConn, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.__closed() {
		return nil, errors.New("connection is closed")
	}
	return c.__dispatch(key), nil
}

func (c *Client) fixError(key string, conn *mix_conn.MixConn) *mix_conn.MixConn {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.__closed() {
		return nil
	}

	i := c.__connIndex(key)
	if conn != c.threads[i] {
		return c.threads[i]
	}

	s, err := c.syncService.GetService()
	if err != nil {
		return nil
	}

	if s.Version > c.version {
		close(c.threads)
		if c.__connect(s) != nil {
			return nil
		}
		return c.__dispatch(key)
	}

	c.threads[i].Close()
	new, err := mix_conn.New(conn.Address, conn.ThreadID, conn.Version)
	if err != nil {
		return nil
	}
	c.threads[i] = new
	return new
}

func (c *Client) secondChanceDo(key string, f func(conn *mix_conn.MixConn) error) error {
	conn, err := c.dispatch(key)
	if err != nil {
		return err
	}

	err = f(conn)
	if err == nil ||
		errors.Is(err, umem_cache.ErrBadKeySize) ||
		errors.Is(err, umem_cache.ErrBadFallbackGet) ||
		errors.Is(err, umem_cache.GetOrSetErrNoMem) ||
		errors.Is(err, umem_cache.SetErrNoMem) ||
		errors.Is(err, umem_cache.DelForSetErrNoMem) {
		return err
	}

	conn = c.fixError(key, conn)
	if conn == nil {
		return err
	}
	return f(conn)
}

func (c *Client) Get(key string) (val []byte, err error) {
	err = c.secondChanceDo(key, func(s *mix_conn.MixConn) error {
		var err error
		val, err = s.Get(key)
		return err
	})
	return
}

func (c *Client) GetOrSet(key string,
	fallbackGet umem_cache.FallbackGetFunc) (val []byte, err error) {
	err = c.secondChanceDo(key, func(s *mix_conn.MixConn) error {
		var err error
		val, err = s.GetOrSet(key, fallbackGet)
		return err
	})
	return
}

// use GetOrSet() instead is recommended
func (c *Client) Set(key string, val []byte) error {
	return c.secondChanceDo(key, func(s *mix_conn.MixConn) error {
		return s.Set(key, val)
	})
}

// use DelForSet() instead is recommended
func (c *Client) Del(key string) error {
	return c.secondChanceDo(key, func(s *mix_conn.MixConn) error {
		return s.Del(key)
	})
}

func (c *Client) DelForSet(key string, fallbackGet umem_cache.FallbackGetFunc) error {
	return c.secondChanceDo(key, func(s *mix_conn.MixConn) error {
		return s.DelForSet(key, fallbackGet)
	})
}
