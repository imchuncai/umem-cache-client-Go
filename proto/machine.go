// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2025, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package proto

import (
	"encoding/binary"
	"net"
)

const _MACHINE_BIN_SIZE = 16 + 2 + 2 + 4 + 8 + 8

type Machine struct {
	Addr      *net.TCPAddr
	ID        uint32
	Stability uint64
	Version   uint64
}

func (m *Machine) Available() bool {
	return m.Stability&1 == 1
}

func (m *Machine) Match(addr *net.TCPAddr) bool {
	return AddrEqual(m.Addr, addr)
}

func (m *Machine) append(dest []byte) []byte {
	dest = append(dest, m.Addr.IP...)
	dest = binary.BigEndian.AppendUint16(dest, uint16(m.Addr.Port))
	dest = append(dest, 0, 0)
	dest = binary.BigEndian.AppendUint32(dest, m.ID)
	dest = binary.BigEndian.AppendUint64(dest, m.Stability)
	return binary.BigEndian.AppendUint64(dest, m.Version)
}

func newMachine(bin []byte) Machine {
	addr := new(net.TCPAddr)
	addr.IP = make([]byte, 16)
	copy(addr.IP, bin)
	addr.Port = int(binary.BigEndian.Uint16(bin[16:]))
	return Machine{
		addr,
		binary.BigEndian.Uint32(bin[20:]),
		binary.BigEndian.Uint64(bin[24:]),
		binary.BigEndian.Uint64(bin[32:]),
	}
}

func NewInitialMachine(addr *net.TCPAddr) Machine {
	return Machine{Addr: addr}
}
