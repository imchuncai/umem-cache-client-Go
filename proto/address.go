// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2025-2026, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package proto

import (
	"errors"
	"fmt"
	"math"
	"net"
	"net/netip"
)

func powerOf2(n uint) bool {
	return n&(n-1) == 0
}

func ResolveAddresses(addresses []string) ([]netip.AddrPort, error) {
	n := uint(len(addresses))
	if n < 4 || n > math.MaxUint32 || !powerOf2(n) {
		return nil, errors.New("bad cluster machine size")
	}

	addr := make([]netip.AddrPort, len(addresses))
	for i := range addresses {
		tcpAddr, err := net.ResolveTCPAddr("tcp6", addresses[i])
		if err != nil {
			return nil, fmt.Errorf("resolve tcp6 address: %s failed: %w", addresses[i], err)
		}
		addr[i] = tcpAddr.AddrPort()
	}
	return addr, nil
}
