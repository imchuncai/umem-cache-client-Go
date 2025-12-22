// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2025, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package client

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"os"
	"time"

	"github.com/imchuncai/umem-cache-client-Go/proto"
)

func nap() {
	time.Sleep(100 * time.Millisecond)
}

func verifyLeader(deadline time.Time, addrs []string, config *tls.Config) error {
	for i := 0; i < len(addrs); i++ {
		addr := addrs[i]
		leader, err := AdminLeader(deadline, addr, config)
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				return fmt.Errorf("request leader failed: %w", err)
			}

			i--
			nap()
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

	conn, err := proto.Dial(deadline, addresses[0], config, true)
	if err != nil {
		return fmt.Errorf("dial initial leader failed: %w", err)
	}
	defer conn.Close()

	err = conn.SetDeadline(deadline)
	if err != nil {
		return fmt.Errorf(wrapper, err)
	}

	err = conn.InitCluster(addrs)
	if err != nil {
		return fmt.Errorf(wrapper, err)
	}

	err = verifyLeader(deadline, addresses, config)
	if err != nil {
		return fmt.Errorf(wrapper, fmt.Errorf("verify leader failed: %w", err))
	}
	return nil
}

func adjustMachines(cluster proto.Cluster, to []*net.TCPAddr) []proto.Machine {
	n := len(cluster.Machines)
	machines := make([]proto.Machine, n)
	for i := 0; i < n; i++ {
		if cluster.Machines[i].Match(to[i]) {
			machines[i] = cluster.Machines[i]
		} else {
			machines[i] = proto.NewInitialMachine(to[i])
		}
	}
	return machines
}

func shrinkMachines(cluster proto.Cluster, to []*net.TCPAddr) ([]proto.Machine, error) {
	n := len(to)
	for i := 0; i < n; i++ {
		if !cluster.Machines[i].Match(to[i]) {
			return nil, errors.New("bad addresses")
		}
	}
	return cluster.Machines[:n], nil
}

func growMachines(cluster proto.Cluster, to []*net.TCPAddr) ([]proto.Machine, error) {
	n := len(cluster.Machines)
	for i := 0; i < n; i++ {
		if !cluster.Machines[i].Match(to[i]) {
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

// Note: change may not really happen, you can check by AdminClusterMatch()
func AdminChangeCluster(deadline time.Time, fromAddresses, toAddresses []string, config *tls.Config) error {
	_, err := proto.ResolveAddresses(fromAddresses)
	if err != nil {
		return fmt.Errorf("resolve from addresses failed: %w", err)
	}
	toAddrs, err := proto.ResolveAddresses(toAddresses)
	if err != nil {
		return fmt.Errorf("resolve to addresses failed: %w", err)
	}

	leader, cluster, err := AdminLeaderCluster(deadline, fromAddresses, config)
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

	conn, err := proto.Dial(deadline, leader, config, true)
	if err != nil {
		return fmt.Errorf("dial leader failed: %w", err)
	}
	defer conn.Close()

	err = conn.SetDeadline(deadline)
	if err != nil {
		return err
	}

	err = conn.ChangeCluster(machines)
	if err != nil {
		return fmt.Errorf("change cluster failed: %w", err)
	}
	return nil
}

func AdminClusterMatch(deadline time.Time, addresses []string, config *tls.Config) error {
	addrs, err := proto.ResolveAddresses(addresses)
	if err != nil {
		return fmt.Errorf("resolve addresses failed: %w", err)
	}

	for {
		_, cluster, err := AdminLeaderCluster(deadline, addresses, config)
		if err != nil {
			return err
		}

		if cluster.Type.Normal() {
			return cluster.Match(addrs)
		}
	}
}

func leaderRandom(deadline time.Time, addrs []string, config *tls.Config) (string, error) {
	cp := make([]string, len(addrs))
	copy(cp, addrs)

	rand.Shuffle(len(cp), func(a, b int) {
		cp[a], cp[b] = cp[b], cp[a]
	})

	for _, addr := range cp {
		leader, err := AdminLeader(deadline, addr, config)
		if err == nil || errors.Is(err, os.ErrDeadlineExceeded) {
			return leader, err
		}
	}
	return "", errors.New("lost leader")
}

func cluster(deadline time.Time, address string, config *tls.Config) (proto.Cluster, error) {
	conn, err := proto.Dial(deadline, address, config, true)
	if err != nil {
		return proto.Cluster{}, fmt.Errorf("dial:%s failed: %w", address, err)
	}
	defer conn.Close()

	err = conn.SetDeadline(deadline)
	if err != nil {
		return proto.Cluster{}, err
	}

	cluster, err := conn.Cluster()
	if err != nil {
		return proto.Cluster{}, fmt.Errorf("request cluster failed: %w", err)
	}
	return cluster, nil
}

func AdminLeaderCluster(deadline time.Time, addresses []string, config *tls.Config) (string, proto.Cluster, error) {
	do := func() (string, proto.Cluster, error) {
		leader, err := leaderRandom(deadline, addresses, config)
		if err != nil {
			return "", proto.Cluster{}, fmt.Errorf("request leader failed: %w", err)
		}

		cluster, err := cluster(deadline, leader, config)
		if err != nil {
			return "", proto.Cluster{}, err
		}

		return leader, cluster, nil
	}

	for {
		leader, cluster, err := do()
		if err == nil || errors.Is(err, os.ErrDeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
			return leader, cluster, err
		}
		nap()
	}
}

func AdminLeader(deadline time.Time, address string, config *tls.Config) (string, error) {
	conn, err := proto.Dial(deadline, address, config, true)
	if err != nil {
		return "", fmt.Errorf("dial failed: %w", err)
	}
	defer conn.Close()

	err = conn.SetDeadline(deadline)
	if err != nil {
		return "", err
	}
	return conn.Leader()
}

func AdminCluster(deadline time.Time, address string, config *tls.Config) (proto.Cluster, error) {
	_, err := net.ResolveTCPAddr("tcp6", address)
	if err != nil {
		return proto.Cluster{}, fmt.Errorf("resolve addresses failed: %w", err)
	}

	return cluster(deadline, address, config)
}
