// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2025-2026, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package client

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"sync/atomic"
	"time"

	"github.com/imchuncai/umem-cache-client-Go/proto"
	"github.com/twmb/murmur3"
)

// annoying stupid DeadlineExceeded
func errIsIOTimeout(err error) bool {
	return errors.Is(err, os.ErrDeadlineExceeded) || errors.Is(err, context.DeadlineExceeded)
}

type Cluster struct {
	proto.Cluster
	Leader netip.AddrPort
}

type ClusterClient struct {
	threadNR uint32
	config   Config

	curr atomic.Pointer[_ClusterClient]
	next atomic.Pointer[_ClusterClient]
}

func NewClusterClient(addresses []string, config Config) (*ClusterClient, error) {
	if len(addresses) == 0 {
		return nil, errors.New("empty addresses")
	}

	err := config.check()
	if err != nil {
		return nil, err
	}

	deadline := config.deadline()
	cluster, authority, err := clusterAuthority(deadline, addresses, config.TLSConfig)
	if err != nil {
		return nil, err
	}

	var _threadNR uint32
	for _, m := range cluster.Machines {
		_threadNR, err = threadNR(deadline, m.Addr.String(), config.TLSConfig)
		if err == nil || errIsIOTimeout(err) {
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf("get threadNR failed: %w", err)
	}

	curr := &_ClusterClient{
		nil,
		newAuthority(authority),
		cluster,
		newMembers(cluster.Machines, _threadNR, config),
	}
	c := &ClusterClient{threadNR: _threadNR, config: config}
	c.curr.Store(curr)
	c.next.Store(curr)
	return c, nil
}

func (c *ClusterClient) rebuild(client *_ClusterClient) <-chan struct{} {
	ch := make(chan struct{})
	next := &_ClusterClient{updating: ch}
	if !c.next.CompareAndSwap(client, next) {
		return c.next.Load().updating
	}

	go func() {
		next.Init(client, c.threadNR, c.config)
		if !c.curr.CompareAndSwap(client, next) {
			next.Close()
		}
		close(ch)
	}()
	return ch
}

func (c *ClusterClient) do(f func(ctx context.Context, deadline time.Time, client *_ClusterClient) error) error {
	deadline := c.config.deadline()
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	for {
		client := c.curr.Load()
		if client == nil {
			return errors.New("cluster is closed")
		}

		err := f(ctx, deadline, client)
		if err == nil || errors.Is(err, proto.ErrClientSide) {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.rebuild(client):
		}
	}
}

// Note: get is guaranteed called at most once
func (c *ClusterClient) GetOrSet(key []byte, get proto.FallbackGetFunc) (val []byte, err error) {
	h1, h2 := murmur3.SeedSum128(74, 74, key)
	err = c.do(func(ctx context.Context, deadline time.Time, client *_ClusterClient) error {
		val, err = client.GetOrSet(ctx, deadline, c.threadNR, key, h1, h2, get)
		return err
	})
	return
}

func (c *ClusterClient) Del(key []byte) error {
	h1, h2 := murmur3.SeedSum128(74, 74, key)
	return c.do(func(ctx context.Context, deadline time.Time, client *_ClusterClient) error {
		return client.Del(ctx, deadline, c.threadNR, key, h1, h2)
	})
}

func (c *ClusterClient) Cluster() (cluster Cluster, err error) {
	err = c.do(func(ctx context.Context, deadline time.Time, client *_ClusterClient) error {
		cluster, err = client.Cluster(ctx, deadline)
		return err
	})
	return
}

func (c *ClusterClient) Close() {
	client := c.curr.Swap(nil)
	if client != nil {
		client.Close()
	}
}
