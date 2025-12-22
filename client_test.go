// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2025, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package client

import (
	"math"
	"net"
	"testing"

	"github.com/imchuncai/umem-cache-raft-client-Go/proto"
)

const (
	CLIENT_FUZZ_N       = 100
	CLIENT_PORT         = 10047
	CLIENT_KEY_MAX_SIZE = math.MaxUint8
)

func TestClient(t *testing.T) {
	param := InitTest(t)

	client, err := New(MachineAddress(CLIENT_PORT), param.Config)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	machine, err := RunMachine(CLIENT_PORT, param)
	if err != nil {
		t.Fatal(err)
	}
	defer machine.Stop()

	t.Run("TooManyConnections", testClientTooManyConnections(param))
	t.Run("Timeout", testClientTimeout(client))

	// Note: if we run basic test on t.Failed(), previous fail log will be wiped
	if !t.Failed() {
		pool := NewPool(CLIENT_KEY_MAX_SIZE, CLIENT_FUZZ_N)
		testBasic(t, client, pool)
	}
}

func testClientTooManyConnections(param TestParam) func(t *testing.T) {
	return func(t *testing.T) {
		deadline := DEADLINE()

		address := MachineAddress(CLIENT_PORT)
		_, err := net.ResolveTCPAddr("tcp6", address)
		if err != nil {
			t.Fatal(err)
		}

		conns := make([]*proto.CacheConn, 0, THREAD_MAX_CONN)
		defer func() {
			for _, conn := range conns {
				conn.Close()
			}
		}()

		for i := 0; i < THREAD_MAX_CONN; i++ {
			conn, err := proto.DialCache(deadline, address, 0, param.Config.TLSConfig)
			if err != nil {
				t.Fatal(err)
			}
			conns = append(conns, conn)
		}

		conn, err := proto.DialCache(deadline, address, 0, param.Config.TLSConfig)
		if err == nil {
			conn.Close()
		}
		if !ErrIsClosedByPeer(err) {
			t.Fatal(err)
		}
	}
}

func testClientTimeout(client *Client) func(t *testing.T) {
	return func(t *testing.T) {
		tc := Case{
			[]byte("hello"),
			[]byte("world"),
		}

		fallbackGet := func(key []byte) (val []byte, err error) {
			return tc.Val, nil
		}

		badFallbackGet := func(key []byte) ([]byte, error) {
			ch := make(chan struct{})
			var err error
			go func() {
				_, err = client.GetOrSet(tc.Key, fallbackGet)
				ch <- struct{}{}
			}()
			<-ch
			if err != nil {
				t.Fatal(err)
			}
			return tc.Val, nil
		}

		del(t, client, tc)
		_, err := client.GetOrSet(tc.Key, badFallbackGet)
		if err != nil {
			t.Fatal(err)
		}
		check(t, client, tc)
	}
}
