// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2025, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package client

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/imchuncai/umem-cache-raft-client-Go/proto"
)

var errClosed = errors.New("cluster is closed")

type Cluster struct {
	config Config

	mu       sync.RWMutex
	closed   bool
	updating bool

	version   uint64
	leader    string
	authority *authority
	members   []member
}

func errIsIOTimeout(err error) bool {
	return errors.Is(err, os.ErrDeadlineExceeded) || errors.Is(err, context.DeadlineExceeded)
}

func NewCluster(addresses []string, config Config) (*Cluster, error) {
	err := config.check()
	if err != nil {
		return nil, err
	}

	deadline := config.deadline()
	leader, cluster, authority, err := leaderClusterAuthority(deadline, addresses, config.TLSConfig)
	if err != nil {
		return nil, err
	}

	return &Cluster{
		config:    config,
		closed:    false,
		updating:  false,
		version:   cluster.Version,
		leader:    leader,
		authority: newAuthority(authority),
		members:   newMembers(cluster.Machines, config),
	}, nil
}

func leaderClusterAuthority(deadline time.Time, addresses []string, config *tls.Config) (string, proto.Cluster, *proto.AuthorityConn, error) {
	for {
		leader, cluster, err := AdminLeaderCluster(deadline, addresses, config)
		if err != nil {
			return "", proto.Cluster{}, nil, err
		}

		var authority *proto.AuthorityConn
		authority, err = proto.DialAuthority(deadline, leader, config)
		if err == nil {
			return leader, cluster, authority, nil
		}
		if errIsIOTimeout(err) {
			return "", proto.Cluster{}, nil, err
		}

		nap()
	}
}

func (c *Cluster) rebuild(version uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed || c.updating || version != c.version {
		return
	}

	c.updating = true
	go func() {
		defer func() {
			c.mu.Lock()
			defer c.mu.Unlock()

			c.updating = false
			if c.closed {
				c.__close()
			}
		}()

		addrs := make([]string, len(c.members))
		for i := range c.members {
			addrs[i] = c.members[i].address
		}

		leader, cluster, authority, _ := leaderClusterAuthority(time.Time{}, addrs, c.config.TLSConfig)

		// a cluster member is not working well, but cluster is not detected that yet.
		// we should not make the decision to rebuild authority.
		if cluster.Version == c.version && leader == c.leader && !c.authority.Closed() {
			return
		}

		c.leader = leader
		c.authority.Close()
		c.authority = newAuthority(authority)

		if c.version != cluster.Version {
			c.version = cluster.Version
			for i := range c.members {
				c.members[i].Close()
			}
			c.members = newMembers(cluster.Machines, c.config)
		}
	}()
}

func (c *Cluster) auth() (uint64, *authority, []member, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return 0, nil, nil, errClosed
	}

	if c.updating {
		return 0, nil, nil, fmt.Errorf("cluster is updating")
	}

	return c.version, c.authority, c.members, nil
}

func (c *Cluster) doOnce(ctx context.Context, deadline time.Time, hkey uint64, f func(m member, threadID int) error) (uint64, error) {
	version, auth, members, err := c.auth()
	if err != nil {
		return 0, err
	}

	ch, err := auth.RequestPermission(deadline)
	if err != nil {
		return version, err
	}

	threadI := hkey % uint64(len(members)*c.config.ThreadNR)
	m := members[threadI/uint64(c.config.ThreadNR)]
	threadID := int(threadI % uint64(c.config.ThreadNR))
	err = f(m, threadID)
	if err != nil {
		return version, err
	}

	select {
	case <-ctx.Done():
		return 0, os.ErrDeadlineExceeded
	case v2 := <-ch:
		if v2 == version {
			return 0, nil
		}
		return version, fmt.Errorf("cluster version changed from : %d to %d", version, v2)
	}
}

func (c *Cluster) do(deadline time.Time, hkey uint64, f func(m member, threadID int) error) error {
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	for ctx.Err() == nil {
		version, err := c.doOnce(ctx, deadline, hkey, f)
		if err == nil || errIsIOTimeout(err) || errors.Is(err, proto.ErrClient) || errors.Is(err, errClosed) {
			return err
		}

		nap()
		c.rebuild(version)
	}
	return ctx.Err()
}

func (c *Cluster) deadline() time.Time {
	return c.config.deadline()
}

func (c *Cluster) Del(key []byte) error {
	deadline := c.deadline()

	hkey := hash(key)
	return c.do(deadline, hkey, func(m member, threadID int) error {
		return m.Del(deadline, threadID, key)
	})
}

func (c *Cluster) GetOrSet(key []byte, fallbackGet proto.FallbackGetFunc) (val []byte, err error) {
	deadline := c.deadline()

	hkey := hash(key)
	err = c.do(deadline, hkey, func(m member, threadID int) error {
		val, err = m.GetOrSet(deadline, threadID, key, fallbackGet)
		return err
	})
	return
}

func (c *Cluster) __close() {
	c.authority.Close()
	for _, m := range c.members {
		m.Close()
	}
}

func (c *Cluster) Close() {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.closed {
		c.closed = true
		if !c.updating {
			c.__close()
		}
	}
}
