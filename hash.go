// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2025, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package client

import (
	"github.com/twmb/murmur3"
)

func hash(key []byte) uint64 {
	h1, _ := murmur3.SeedSum128(74, 74, key)
	return h1
}
