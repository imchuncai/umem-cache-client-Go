// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2024, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package client

import (
	"errors"
	"fmt"
	"hash"
	"hash/fnv"
	"sync"
	"time"

	"github.com/imchuncai/umem-cache-client-Go/thread"
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
	timeout time.Duration

	mu          sync.Mutex
	syncService SyncService
	version     uint32
	hasher      hash.Hash32
	threads     []*thread.Thread
}

func closeThreads(threads []*thread.Thread) {
	for _, thread := range threads {
		if thread != nil {
			thread.Close()
		}
	}
}

func connect(s Server, timeout time.Duration) ([]*thread.Thread, error) {
	if len(s.Threads) == 0 {
		return nil, errors.New("empty server")
	}

	threads := make([]*thread.Thread, len(s.Threads))
	for i, t := range s.Threads {
		threads[i] = thread.New(t.Address, t.ThreadID, s.Version, timeout)
	}
	return threads, nil
}

// Note: must hold c.mu
func (c *Client) __reconnect(s Server) error {
	closeThreads(c.threads)

	if len(s.Threads) == 0 {
		return errors.New("no thread to connect")
	}

	threads, err := connect(s, c.timeout)
	if err != nil {
		return err
	}

	c.version = s.Version
	c.threads = threads
	return nil
}

func New(syncService SyncService, timeout time.Duration) (*Client, error) {
	c := Client{hasher: fnv.New32a(), syncService: syncService, timeout: timeout}
	server, err := syncService.GetService()
	if err != nil {
		return nil, fmt.Errorf("sync service error: %w", err)
	}
	err = c.__reconnect(server)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// Note: must hold c.mu
func (c *Client) __closed() bool {
	return c.threads == nil
}

func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.__closed() {
		closeThreads(c.threads)
		c.threads = nil
	}
}

func (c *Client) dispatch(key []byte) (*thread.Thread, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.__closed() {
		return nil, errors.New("client is closed")
	}

	c.hasher.Reset()
	c.hasher.Write([]byte(key))
	i := int(c.hasher.Sum32()) % len(c.threads)
	return c.threads[i], nil
}

func (c *Client) checkSyncService() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.__closed() {
		return errors.New("client is closed")
	}

	s, err := c.syncService.GetService()
	if err != nil {
		return fmt.Errorf("sync server failed: %w", err)
	}

	if s.Version < c.version {
		return fmt.Errorf("sync server invalid version: %d, current version: %d", s.Version, c.version)
	}

	if s.Version == c.version {
		return fmt.Errorf("unexpected failure")
	}

	err = c.__reconnect(s)
	if err != nil {
		return fmt.Errorf("reconnect failed: %w", err)
	}
	return nil
}

func (c *Client) do(key []byte, f func(conn *umem_cache.Conn) error) (err error, checkSyncService bool) {
	t, err := c.dispatch(key)
	if err != nil {
		return err, false
	}

	conn, err := t.Dispatch(c.timeout)
	if err != nil {
		return err, true
	}

	err = f(conn)
	if err == nil || errors.Is(err, umem_cache.ErrBadKeySize) ||
		errors.Is(err, umem_cache.ErrBadFallbackGet) {
		t.Return(conn)
		return err, false
	}

	conn.Close()
	return err, true
}

func (c *Client) secondChanceDo(key []byte, f func(conn *umem_cache.Conn) error) error {
	err, checkSyncService := c.do(key, f)
	if !checkSyncService {
		return err
	}

	err2 := c.checkSyncService()
	if err2 != nil {
		return fmt.Errorf("%w %w", err, err2)
	}

	err, _ = c.do(key, f)
	return err
}

func (c *Client) GetOrSet(key []byte, fallbackGet umem_cache.FallbackGetFunc) (val []byte, err error) {
	err = c.secondChanceDo(key, func(s *umem_cache.Conn) error {
		var err error
		val, err = s.GetOrSet(key, fallbackGet, c.timeout)
		return err
	})
	return
}

func (c *Client) Del(key []byte) error {
	return c.secondChanceDo(key, func(s *umem_cache.Conn) error {
		return s.Del(key, c.timeout)
	})
}
