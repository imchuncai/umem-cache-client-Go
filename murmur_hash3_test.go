// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2026, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package client

import "testing"

func FuzzMurmurHash3_x64_128(f *testing.F) {
	f.Add(make([]byte, 255), uint32(74))
	f.Fuzz(func(t *testing.T, key []byte, seed uint32) {
		h1, h2 := murmurHash3_x64_128(key, seed)
		ch1, ch2 := murmurHash3_x64_128_C(key, seed)
		if h1 != ch1 || h2 != ch2 {
			t.Fatalf("key: %v seed: %d", key, seed)
		}
	})
}
