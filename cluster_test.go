// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2025, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package client

import (
	"crypto/tls"
	"fmt"
	"math"
	"net"
	"os"
	"testing"
	"time"

	"github.com/imchuncai/umem-cache-raft-client-Go/proto"
)

const (
	CLUSTER_PORT_FROM      = 10047
	CLUSTER_FUZZ_N         = 10
	CLUSTER_KEY_SIZE_LIMIT = math.MaxUint8 - 8
)

func RunAndInitCluster(param TestParam, n int, addresses []string) (Machines, error) {
	ports := make([]int, n)
	for i := range n {
		ports[i] = CLUSTER_PORT_FROM + 2*i
	}
	deadline := DEADLINE()

	machines, err := RunMachines(ports, param)
	if err != nil {
		return nil, fmt.Errorf("run test cluster failed: %w", err)
	}

	err = AdminInitCluster(deadline, addresses, param.Config.TLSConfig)
	if err != nil {
		machines.Stop()
		return nil, fmt.Errorf("init cluster failed: %w", err)
	}
	return machines, nil
}

func ADDRESSES4() []string {
	addresses := make([]string, 4)
	for i := range 4 {
		addresses[i] = MachineAddress(CLUSTER_PORT_FROM + 2*i)
	}
	return addresses
}

func ADDRESSES8() []string {
	addresses := make([]string, 8)
	for i := range 8 {
		addresses[i] = MachineAddress(CLUSTER_PORT_FROM + 2*i)
	}
	return addresses
}

func ADDRESSES_ADMIN4() []string {
	addresses := make([]string, 4)
	for i := range 4 {
		addresses[i] = MachineAddress(CLUSTER_PORT_FROM + 2*i + 1)
	}
	return addresses
}

func ADDRESSES_ADMIN8() []string {
	addresses := make([]string, 8)
	for i := range 8 {
		addresses[i] = MachineAddress(CLUSTER_PORT_FROM + 2*i + 1)
	}
	return addresses
}

func TestCluster(t *testing.T) {
	param := InitTest(t)

	testClusterBasic(t, param)
	t.Run("ChangeAvailableTail", testClusterChangeAvailable(param))
	t.Run("Adjust", testClusterAdjust(param))
	t.Run("LeaderStepDown", testClusterLeaderStepDown(param))
	t.Run("Shrink", testClusterShrink(param))
	t.Run("Grow", testClusterGrow(param))
	t.Run("Election", testClusterElection(param))
}

func testClusterBasic(t *testing.T, param TestParam) {
	machines, err := RunAndInitCluster(param, 4, ADDRESSES_ADMIN4())
	if err != nil {
		t.Fatal(err)
	}
	defer machines.Stop()

	cluster, err := NewCluster(ADDRESSES4(), param.Config)
	if err != nil {
		t.Fatal(fmt.Errorf("new cluster failed: %w", err))
	}
	defer cluster.Close()

	pool := NewPool(CLUSTER_KEY_SIZE_LIMIT, CLUSTER_FUZZ_N)
	testBasic(t, cluster, pool)

	t.Run("NotAdmin", testClusterNotAdmin(param))
	t.Run("AdjustDuplicate", testClusterAdjustDuplicate(param))
	t.Run("GrowDuplicate", testClusterGrowDuplicate(param))
	t.Run("GrowDuplicate2", testClusterGrowDuplicate2(param))
}

func testClusterNotAdmin(param TestParam) func(t *testing.T) {
	return func(t *testing.T) {
		deadline := DEADLINE()

		err := AdminInitCluster(deadline, ADDRESSES4(), param.Config.TLSConfig)
		if !ErrIsClosedByPeer(err) {
			t.Fatal(err)
		}
	}
}

func ClusterCheckAvailable(addresses []string, availabilities []bool, config *tls.Config) error {
	_, err := proto.ResolveAddresses(ADDRESSES_ADMIN4())
	if err != nil {
		return fmt.Errorf("resolve addresses failed: %w", err)
	}

	deadline := DEADLINE()

loop:
	for {
		nap()
		_, cluster, err := AdminLeaderCluster(deadline, addresses, config)
		if err != nil {
			return fmt.Errorf("request leader cluster failed: %w", err)
		}
		if cluster.Type.Stable() && len(cluster.Machines) == len(addresses) {
			for i, m := range cluster.Machines {
				if m.Available() != availabilities[i] {
					continue loop
				}
			}
			return nil
		}
	}
}

func testClusterChangeAvailable(param TestParam) func(t *testing.T) {
	return func(t *testing.T) {
		addresses := ADDRESSES_ADMIN4()
		machines, err := RunAndInitCluster(param, 4, addresses)
		if err != nil {
			t.Fatal(err)
		}
		defer machines.Stop()

		err = machines[3].Stop()
		if err != nil {
			t.Fatal(err)
		}

		availabilities := []bool{true, true, true, false}
		err = ClusterCheckAvailable(addresses, availabilities, param.Config.TLSConfig)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func _testClusterAdjust(t *testing.T, from []string, to []string, param TestParam) {
	machines, err := RunAndInitCluster(param, 8, from)
	if err != nil {
		t.Fatal(err)
	}
	defer machines.Stop()

	deadline := DEADLINE()

	err = AdminChangeCluster(deadline, from, to, param.Config.TLSConfig)
	if err != nil {
		t.Fatal(err)
	}

	for {
		err = AdminClusterMatch(deadline, to, param.Config.TLSConfig)
		if err == nil {
			return
		}
		if err == os.ErrDeadlineExceeded {
			t.Fatal(err)
		}
	}
}

func testClusterAdjust(param TestParam) func(t *testing.T) {
	return func(t *testing.T) {
		from := ADDRESSES_ADMIN4()
		to := ADDRESSES_ADMIN4()
		to[3] = ADDRESSES_ADMIN8()[4]
		_testClusterAdjust(t, from, to, param)
	}
}

func testClusterAdjustDuplicate(param TestParam) func(t *testing.T) {
	return func(t *testing.T) {
		deadline := DEADLINE()

		from := ADDRESSES_ADMIN4()
		to := ADDRESSES_ADMIN4()
		dup := ADDRESSES_ADMIN8()[4]
		to[1] = dup
		to[3] = dup

		err := AdminChangeCluster(deadline, from, to, param.Config.TLSConfig)
		if err != nil {
			t.Fatal(err)
		}

		err = AdminClusterMatch(deadline, from, param.Config.TLSConfig)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func testClusterLeaderStepDown(param TestParam) func(t *testing.T) {
	return func(t *testing.T) {
		from := ADDRESSES_ADMIN4()
		to := ADDRESSES_ADMIN4()
		to[0] = ADDRESSES_ADMIN8()[4]
		_testClusterAdjust(t, from, to, param)
	}
}

func testClusterShrink(param TestParam) func(t *testing.T) {
	return func(t *testing.T) {
		from := ADDRESSES_ADMIN8()
		to := ADDRESSES_ADMIN4()

		machines, err := RunAndInitCluster(param, 8, from)
		if err != nil {
			t.Fatal(err)
		}
		defer machines.Stop()

		deadline := DEADLINE()

		err = AdminChangeCluster(deadline, from, to, param.Config.TLSConfig)
		if err != nil {
			t.Fatal(err)
		}

		err = AdminClusterMatch(deadline, to, param.Config.TLSConfig)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func warmup(deadline time.Time, address string, config *tls.Config) error {
	_, err := net.ResolveTCPAddr("tcp6", address)
	if err != nil {
		return fmt.Errorf("resolve address failed: %w", err)
	}

	conn, err := proto.DialCache(deadline, address, 0, config)
	if err != nil {
		return fmt.Errorf("dial failed: %w", err)
	}
	defer conn.Close()

	small := Case{[]byte{0}, []byte{0}}
	_, err = conn.GetOrSet(deadline, small.Key, fallbackGet(small))
	if err != nil {
		return fmt.Errorf("get or set small failed: %w", err)
	}

	big := Case{[]byte{1}, make([]byte, SERVER_MEMORY/THREAD_NR)}
	val, _ := conn.GetOrSet(deadline, big.Key, fallbackGet(big))
	if val == nil {
		return fmt.Errorf("get or set big failed: %w", err)
	}
	return nil
}

func testClusterGrow(param TestParam) func(t *testing.T) {
	return func(t *testing.T) {
		from := ADDRESSES_ADMIN4()
		to := ADDRESSES_ADMIN8()

		machines, err := RunAndInitCluster(param, 8, from)
		if err != nil {
			t.Fatal(err)
		}
		defer machines.Stop()

		deadline := DEADLINE()

		err = AdminChangeCluster(deadline, from, to, param.Config.TLSConfig)
		if err != nil {
			t.Fatal(err)
		}

		for _, address := range to[1:] {
			err := warmup(deadline, address, param.Config.TLSConfig)
			if err != nil {
				t.Fatalf("warmup: %s failed: %v", address, err)
			}
		}

		err = AdminClusterMatch(deadline, to, param.Config.TLSConfig)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func _testClusterGrowDuplicate(t *testing.T, from []string, to []string, param TestParam) {
	deadline := DEADLINE()

	err := AdminChangeCluster(deadline, from, to, param.Config.TLSConfig)
	if err != nil {
		t.Fatal(err)
	}

	err = AdminClusterMatch(deadline, from, param.Config.TLSConfig)
	if err != nil {
		t.Fatal(err)
	}
}

func testClusterGrowDuplicate(param TestParam) func(t *testing.T) {
	return func(t *testing.T) {
		from := ADDRESSES_ADMIN4()
		to := ADDRESSES_ADMIN8()
		to[4] = from[0]

		_testClusterGrowDuplicate(t, from, to, param)
	}
}

func testClusterGrowDuplicate2(param TestParam) func(t *testing.T) {
	return func(t *testing.T) {
		from := ADDRESSES_ADMIN4()
		to := ADDRESSES_ADMIN8()
		to[7] = to[4]

		_testClusterGrowDuplicate(t, from, to, param)
	}
}

func testClusterElection(param TestParam) func(t *testing.T) {
	return func(t *testing.T) {
		addresses := ADDRESSES_ADMIN4()
		machines, err := RunAndInitCluster(param, 4, addresses)
		if err != nil {
			t.Fatal(err)
		}
		defer machines.Stop()

		err = machines[0].Stop()
		if err != nil {
			t.Fatal(err)
		}

		availabilities := []bool{false, true, true, true}
		err = ClusterCheckAvailable(addresses, availabilities, param.Config.TLSConfig)
		if err != nil {
			t.Fatal(err)
		}
	}
}
