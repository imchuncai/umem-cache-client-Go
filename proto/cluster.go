// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2025, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package proto

import (
	"fmt"
	"net"
)

type ClusterType byte

func (t ClusterType) Normal() bool {
	return t == 0
}

func (t ClusterType) Stable() bool {
	return t <= 1
}

var clusterTypes = [...]string{
	0: "normal",
	1: "grow-transform",
	2: "adjust",
	3: "grow",
	4: "change-available",
	5: "grow-transform-change-available",
	6: "shrink",
	7: "invalid-type",
	8: "grow-transform-complete",
}

func (t ClusterType) String() string {
	if int(t) >= len(clusterTypes) {
		return "invalid-type"
	}
	return clusterTypes[t]
}

type Cluster struct {
	Type     ClusterType
	Version  uint64
	Machines []Machine
}

func (c Cluster) Addrs() []*net.TCPAddr {
	addrs := make([]*net.TCPAddr, len(c.Machines))
	for i, m := range c.Machines {
		addrs[i] = m.Addr
	}
	return addrs
}

func (c Cluster) Match(addrs []*net.TCPAddr) error {
	if len(c.Machines) != len(addrs) {
		return fmt.Errorf("cluster size not match: want: %d got: %d", len(addrs), len(c.Machines))
	}
	for i, m := range c.Machines {
		if !m.Match(addrs[i]) {
			return fmt.Errorf("%dth address not match: want %s got: %s", i, addrs[i], m.Addr)
		}
	}
	return nil
}
