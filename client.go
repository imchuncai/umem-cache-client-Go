// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2025-2026, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package client

import (
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

func New(address string, config Config) (*Client, error) {
	err := config.check()
	if err != nil {
		return nil, err
	}

	return &Client{
		timeout: config.Timeout,
		threads: newThreads(address, config)}, nil
}

func (c *Client) dispatch(key []byte) *thread {
	h1, _ := murmur3.SeedSum128(74, 74, key)
	hi, _ := bits.Mul64(h1, uint64(len(c.threads)))
	return &c.threads[hi]
}

func (c *Client) deadline() time.Time {
	return time.Now().Add(c.timeout)
}

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
