// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2025-2026, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package client

import (
	"crypto/tls"
	"fmt"
	"math/bits"
	"sync"
	"time"

	"github.com/imchuncai/umem-cache-client-Go/proto"
	"github.com/twmb/murmur3"
)

type Client struct {
	timeout time.Duration

	threads []thread
}

func threadNR(deadline time.Time, address string, config *tls.Config) (uint32, error) {
	conn, err := proto.DialCache(deadline, address, 0, config)
	if err != nil {
		return 0, fmt.Errorf("dial: %s thread 0 failed: %w", address, err)
	}
	defer conn.Close()

	stats := conn.Stats()
	if stats.Error != nil {
		return 0, fmt.Errorf("request stats failed: %w", stats.Error)
	}
	return stats.ConfigThreadNR, nil
}

func New(address string, config Config) (*Client, error) {
	err := config.check()
	if err != nil {
		return nil, err
	}

	deadline := time.Now().Add(config.Timeout)
	threadNR, err := threadNR(deadline, address, config.TLSConfig)
	if err != nil {
		return nil, fmt.Errorf("get threadNR failed: %w", err)
	}

	return &Client{
		timeout: config.Timeout,
		threads: newThreads(address, threadNR, config)}, nil
}

func (c *Client) dispatch(key []byte) *thread {
	h1, _ := murmur3.SeedSum128(74, 74, key)
	hi, _ := bits.Mul64(h1, uint64(len(c.threads)))
	return &c.threads[hi]
}

func (c *Client) deadline() time.Time {
	return time.Now().Add(c.timeout)
}

// Note: get is guaranteed called at most once
func (c *Client) GetOrSet(key []byte, get proto.FallbackGetFunc) ([]byte, error) {
	return c.dispatch(key).GetOrSet(c.deadline(), key, get)
}

func (c *Client) Del(key []byte) error {
	return c.dispatch(key).Del(c.deadline(), key)
}

func (c *Client) Stats() []proto.Stats {
	deadline := c.deadline()
	stats := make([]proto.Stats, len(c.threads))
	var wg sync.WaitGroup
	wg.Add(len(c.threads))
	for i := range c.threads {
		go func() {
			stats[i] = c.threads[i].Stats(deadline)
			wg.Done()
		}()
	}
	wg.Wait()
	return stats
}

func (c *Client) Close() {
	for i := range c.threads {
		c.threads[i].Close()
	}
}
