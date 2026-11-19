// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2025-2026, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package client

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/imchuncai/umem-cache-client-Go/proto"
)

func __leader(deadline time.Time, address string, config *tls.Config) (netip.AddrPort, error) {
	conn, err := proto.Dial(deadline, address, config)
	if err != nil {
		return netip.AddrPort{}, fmt.Errorf("dial failed: %w", err)
	}
	defer conn.Close()

	return conn.Leader(deadline)
}

func verifyLeader(deadline time.Time, addrs []netip.AddrPort, config *tls.Config) error {
	for i := 0; i < len(addrs); i++ {
		addr := addrs[i]
		leader, err := __leader(deadline, addr.String(), config)
		if err != nil {
			if deadlineExceeded(deadline) {
				return fmt.Errorf("request leader failed: %w", err)
			}

			i--
			continue
		}
		if leader != addrs[0] {
			return errors.New("cluster is not stable")
		}
	}
	return nil
}

func AdminInitCluster(deadline time.Time, addresses []string, config *tls.Config) error {
	addrs, err := proto.ResolveAddresses(addresses)
	if err != nil {
		return fmt.Errorf("resolve addresses failed: %w", err)
	}

	const wrapper = "init cluster failed: %w, you should restart all machines before retry"

	conn, err := proto.Dial(deadline, addresses[0], config)
	if err != nil {
		return fmt.Errorf("dial initial leader failed: %w", err)
	}
	defer conn.Close()

	err = conn.InitCluster(deadline, addrs)
	if err != nil {
		return fmt.Errorf(wrapper, err)
	}

	err = verifyLeader(deadline, addrs, config)
	if err != nil {
		return fmt.Errorf(wrapper, fmt.Errorf("verify leader failed: %w", err))
	}
	return nil
}

func adjustMachines(cluster Cluster, to []netip.AddrPort) []proto.Machine {
	n := len(cluster.Machines)
	machines := make([]proto.Machine, n)
	for i := 0; i < n; i++ {
		if cluster.Machines[i].Addr == to[i] {
			machines[i] = cluster.Machines[i]
		} else {
			machines[i] = proto.NewInitialMachine(to[i])
		}
	}
	return machines
}

func shrinkMachines(cluster Cluster, to []netip.AddrPort) ([]proto.Machine, error) {
	n := len(to)
	for i := 0; i < n; i++ {
		if cluster.Machines[i].Addr != to[i] {
			return nil, errors.New("bad addresses")
		}
	}
	return cluster.Machines[:n], nil
}

func growMachines(cluster Cluster, to []netip.AddrPort) ([]proto.Machine, error) {
	n := len(cluster.Machines)
	for i := 0; i < n; i++ {
		if cluster.Machines[i].Addr != to[i] {
			return nil, errors.New("bad addresses")
		}
	}

	machines := make([]proto.Machine, n*2)
	copy(machines, cluster.Machines)
	for i := n; i < n*2; i++ {
		machines[i] = proto.NewInitialMachine(to[i])
	}
	return machines, nil
}

// Note: we don't know wether the change is happened or not, caller should always check it.
func AdminChangeCluster(deadline time.Time, toAddresses []string, config *tls.Config) error {
	toAddrs, err := proto.ResolveAddresses(toAddresses)
	if err != nil {
		return fmt.Errorf("resolve to addresses failed: %w", err)
	}

	cluster, err := cluster(deadline, toAddresses, config)
	if err != nil {
		return err
	}

	var machines []proto.Machine
	switch len(toAddresses) {
	case len(cluster.Machines):
		machines = adjustMachines(cluster, toAddrs)
	case len(cluster.Machines) / 2:
		machines, err = shrinkMachines(cluster, toAddrs)
	case len(cluster.Machines) * 2:
		machines, err = growMachines(cluster, toAddrs)
	default:
		err = errors.New("bad addresses")
	}
	if err != nil {
		return err
	}

	conn, err := proto.Dial(deadline, cluster.Leader.String(), config)
	if err != nil {
		return fmt.Errorf("dial leader failed: %w", err)
	}
	defer conn.Close()

	err = conn.ChangeCluster(deadline, machines)
	if err != nil {
		return fmt.Errorf("change cluster failed: %w", err)
	}
	return nil
}
