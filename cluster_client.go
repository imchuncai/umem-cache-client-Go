// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2026, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"math/bits"
	"math/rand/v2"
	"net/netip"
	"os"
	"time"

	"github.com/imchuncai/umem-cache-client-Go/proto"
)

type _ClusterClient struct {
	updating  <-chan struct{}
	authority *authority
	cluster   Cluster
	members   []member
}

func (c *_ClusterClient) Close() {
	c.authority.Close()
	for i := range c.members {
		c.members[i].Close()
	}
}

func __cluster(deadline time.Time, address netip.AddrPort, config *tls.Config) (Cluster, error) {
	conn, err := proto.Dial(deadline, address.String(), config)
	if err != nil {
		return Cluster{}, fmt.Errorf("dial:%s failed: %w", address, err)
	}
	defer conn.Close()

	cluster, err := conn.Cluster(deadline)
	if err != nil {
		return Cluster{}, fmt.Errorf("request cluster failed: %w", err)
	}
	return Cluster{cluster, address}, nil
}

func cluster(deadline time.Time, addresses []string, config *tls.Config) (Cluster, error) {
	for {
		cp := make([]string, len(addresses))
		copy(cp, addresses)
		rand.Shuffle(len(cp), func(a, b int) {
			cp[a], cp[b] = cp[b], cp[a]
		})
		var leader netip.AddrPort
		var err error
		for _, addr := range cp {
			leader, err = AdminLeader(deadline, addr, config)
			if err == nil {
				break
			}
			if errIsIOTimeout(err) {
				return Cluster{}, err
			}
		}

		cluster, err := __cluster(deadline, leader, config)
		if err == nil {
			return cluster, nil
		}
		if errIsIOTimeout(err) {
			return Cluster{}, err
		}
	}
}

func clusterAuthority(deadline time.Time, addresses []string, config *tls.Config) (Cluster, *proto.AuthorityConn, error) {
	for {
		cluster, err := cluster(deadline, addresses, config)
		if err != nil {
			return Cluster{}, nil, err
		}

		var authority *proto.AuthorityConn
		authority, err = proto.DialAuthority(deadline, cluster.Leader.String(), config)
		if err == nil {
			return cluster, authority, nil
		}
		if errIsIOTimeout(err) {
			return Cluster{}, nil, err
		}

		nap()
	}
}

func (c *_ClusterClient) Init(old *_ClusterClient, threadNR uint32, conf Config) {
	addrs := make([]string, len(old.members))
	for i := range old.members {
		addrs[i] = old.members[i].address
	}

	var cluster Cluster
	var authority *proto.AuthorityConn
	var err error
	for {
		cluster, authority, err = clusterAuthority(conf.deadline(), addrs, conf.TLSConfig)
		if err == nil {
			break
		}
	}

	c.cluster = cluster
	if cluster.Version == old.cluster.Version {
		c.members = old.members
		if cluster.Leader == old.cluster.Leader && !old.authority.Closed() {
			// Note: don't close old, it is in use
			c.authority = old.authority
			authority.Close()
		} else {
			c.authority = newAuthority(authority)
			old.authority.Close()
		}
	} else {
		c.members = newMembers(cluster.Machines, threadNR, conf)
		c.authority = newAuthority(authority)
		old.Close()
	}
}

func (c *_ClusterClient) dispatch(threadNR uint32, h1, h2 uint64) (member, uint64) {
	threadID, _ := bits.Mul64(h1, uint64(threadNR))
	return c.members[h2&uint64(len(c.members)-1)], threadID
}

func (c *_ClusterClient) GetOrSet(ctx context.Context, deadline time.Time, threadNR uint32, key []byte, h1, h2 uint64, get proto.FallbackGetFunc) ([]byte, error) {
	ch, err := c.authority.RequestPermission(deadline)
	if err != nil {
		return nil, err
	}

	member, threadID := c.dispatch(threadNR, h1, h2)

	var miss bool
	__get := func(key []byte) (val []byte, err error) {
		miss = true
		return get(key)
	}
	val, err := member.GetOrSet(deadline, threadID, key, __get)
	if err != nil {
		return nil, err
	}
	if miss {
		return val, nil
	}

	select {
	case <-ctx.Done():
		return nil, os.ErrDeadlineExceeded
	case v := <-ch:
		if v == c.cluster.Version {
			return val, nil
		}
		return nil, fmt.Errorf("cluster version changed from : %d to %d", c.cluster.Version, v)
	}
}

func (c *_ClusterClient) Del(ctx context.Context, deadline time.Time, threadNR uint32, key []byte, h1, h2 uint64) error {
	ch, err := c.authority.RequestPermission(deadline)
	if err != nil {
		return err
	}

	member, threadID := c.dispatch(threadNR, h1, h2)
	err = member.Del(deadline, threadID, key)
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return os.ErrDeadlineExceeded
	case v := <-ch:
		if v == c.cluster.Version {
			return nil
		}
		return fmt.Errorf("cluster version changed from : %d to %d", c.cluster.Version, v)
	}
}

func (c *_ClusterClient) Cluster(ctx context.Context, deadline time.Time) (Cluster, error) {
	ch, err := c.authority.RequestPermission(deadline)
	if err != nil {
		return Cluster{}, err
	}

	select {
	case <-ctx.Done():
		return Cluster{}, os.ErrDeadlineExceeded
	case v := <-ch:
		if v == c.cluster.Version {
			return c.cluster, nil
		}
		return Cluster{}, fmt.Errorf("cluster version changed from : %d to %d", c.cluster.Version, v)
	}
}
