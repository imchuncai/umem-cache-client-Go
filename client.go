// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2025, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package client

import (
	"time"

	"github.com/imchuncai/umem-cache-raft-client-Go/proto"
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
	return &c.threads[hash(key)%uint64(len(c.threads))]
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

func (c *Client) Close() {
	for i := range c.threads {
		c.threads[i].Close()
	}
}
