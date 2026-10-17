// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2025-2026, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package proto

import (
	"encoding/binary"
	"net/netip"
)

const _MACHINE_BIN_SIZE = 16 + 2 + 2 + 4 + 8 + 8

type Machine struct {
	Addr      netip.AddrPort
	ID        uint32
	Stability uint64
	Version   uint64
}

func (m *Machine) Available() bool {
	return m.Stability&1 == 1
}

func (m *Machine) append(dest []byte) []byte {
	ipv6 := m.Addr.Addr().As16()
	dest = append(dest, ipv6[:]...)
	dest = binary.BigEndian.AppendUint16(dest, m.Addr.Port())
	dest = append(dest, 0, 0)
	dest = binary.LittleEndian.AppendUint32(dest, m.ID)
	dest = binary.LittleEndian.AppendUint64(dest, m.Stability)
	return binary.LittleEndian.AppendUint64(dest, m.Version)
}

func newMachine(bin []byte) Machine {
	ip := netip.AddrFrom16([16]byte(bin[:16]))
	port := binary.BigEndian.Uint16(bin[16:])
	addr := netip.AddrPortFrom(ip, port)
	return Machine{
		addr,
		binary.LittleEndian.Uint32(bin[20:]),
		binary.LittleEndian.Uint64(bin[24:]),
		binary.LittleEndian.Uint64(bin[32:]),
	}
}

func NewInitialMachine(addr netip.AddrPort) Machine {
	return Machine{Addr: addr}
}
