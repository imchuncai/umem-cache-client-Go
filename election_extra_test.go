// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2025, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package client

import (
	"testing"
)

func TestElectionWithUnstableLog(t *testing.T) {
	param := InitTest(t)

	t.Run("Adjust", testClusterAdjust(param))
	t.Run("Shrink", testClusterShrink(param))
	t.Run("Grow", testClusterGrow(param))
	t.Run("ChangeAvailable", testElectionOnChangeAvailable(param))
}

func testElectionOnChangeAvailable(param TestParam) func(t *testing.T) {
	return func(t *testing.T) {
		addresses := ADDRESSES_ADMIN8()
		machines, err := RunAndInitCluster(param, 8, addresses)
		if err != nil {
			t.Fatal(err)
		}
		defer machines.Stop()

		err = machines[3].Stop()
		if err != nil {
			t.Fatal(err)
		}

		availabilities := []bool{false, true, true, false, true, true, true, true}
		err = ClusterCheckAvailable(addresses, availabilities, param.Config.TLSConfig)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestElectionWithUnstableGrowLog(t *testing.T) {
	param := InitTest(t)

	t.Run("ChangeAvailable", testElectionOnGrowChangeAvailable(param))
	t.Run("Complete", testClusterGrow(param))
}

func testElectionOnGrowChangeAvailable(param TestParam) func(t *testing.T) {
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

		err = machines[3].Stop()
		if err != nil {
			t.Fatal(err)
		}

		availabilities := []bool{false, true, true, false, true, true, true, true}
		err = ClusterCheckAvailable(to, availabilities, param.Config.TLSConfig)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestVoteWithLog0(t *testing.T) {
	param := InitTest(t)

	from := ADDRESSES_ADMIN4()
	to := ADDRESSES_ADMIN8()

	machines, err := RunAndInitCluster(param, 8, from)
	if err != nil {
		t.Fatal(err)
	}
	defer machines.Stop()

	deadline := DEADLINE()

	// make sure only machine[0] can win election
	err = machines[1].Stop()
	if err != nil {
		t.Fatal(err)
	}

	err = AdminChangeCluster(deadline, from, to, param.Config.TLSConfig)
	if err != nil {
		t.Fatal(err)
	}

	for _, address := range to[2:] {
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
