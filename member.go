// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2025, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package client

import (
	"encoding/binary"
	"time"

	"github.com/imchuncai/umem-cache-raft-client-Go/proto"
)

type member struct {
	version uint64
	address string
	threads []thread
}

func newMembers(machines []proto.Machine, conf Config) []member {
	var route string
	for _, m := range machines {
		if m.Available() {
			route = m.Addr.String()
			break
		}
	}

	members := make([]member, len(machines))
	for i := len(machines) - 1; i >= 0; i-- {
		m := machines[i]
		if m.Available() {
			route = m.Addr.String()
		}
		members[i].init(m, route, conf)
	}
	return members
}

func (m *member) init(machine proto.Machine, route string, conf Config) {
	m.version = machine.Version
	m.address = machine.Addr.String()
	m.threads = newThreads(route, conf)
}

func (m *member) Close() {
	for i := range m.threads {
		m.threads[i].Close()
	}
}

func (m *member) realKey(key []byte) []byte {
	k := make([]byte, 8+len(key))
	binary.BigEndian.PutUint64(k, m.version)
	copy(k[8:], key)
	return k
}

func (m *member) GetOrSet(deadline time.Time, threadID int, key []byte, get proto.FallbackGetFunc) (val []byte, err error) {
	key = m.realKey(key)
	return m.threads[threadID].GetOrSet(deadline, key, get)
}

func (m *member) Del(deadline time.Time, threadID int, key []byte) error {
	key = m.realKey(key)
	return m.threads[threadID].Del(deadline, key)
}
