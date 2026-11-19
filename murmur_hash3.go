// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2026, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package client

// #include "murmur_hash3.h"
import "C"
import (
	"encoding/binary"
	"unsafe"
)

func rotl64(x uint64, r int8) uint64 {
	return (x << r) | (x >> (64 - r))
}

func getblock64(p []byte, i int) uint64 {
	return binary.LittleEndian.Uint64(p[i<<3:])
}

func fmix64(k uint64) uint64 {
	k ^= k >> 33
	k *= 0xff51afd7ed558ccd
	k ^= k >> 33
	k *= 0xc4ceb9fe1a85ec53
	k ^= k >> 33

	return k
}

func murmurHash3_x64_128(key []byte, seed uint32) (h1 uint64, h2 uint64) {
	nblocks := len(key) / 16

	h1 = uint64(seed)
	h2 = uint64(seed)

	const c1 = 0x87c37b91114253d5
	const c2 = 0x4cf5ad432745937f

	//----------
	// body

	for i := 0; i < nblocks; i++ {
		k1 := getblock64(key, i*2+0)
		k2 := getblock64(key, i*2+1)

		k1 *= c1
		k1 = rotl64(k1, 31)
		k1 *= c2
		h1 ^= k1

		h1 = rotl64(h1, 27)
		h1 += h2
		h1 = h1*5 + 0x52dce729

		k2 *= c2
		k2 = rotl64(k2, 33)
		k2 *= c1
		h2 ^= k2

		h2 = rotl64(h2, 31)
		h2 += h1
		h2 = h2*5 + 0x38495ab5
	}

	//----------
	// tail

	tail := key[nblocks*16:]

	var k1, k2 uint64

	switch len(key) & 15 {
	case 15:
		k2 ^= uint64(tail[14]) << 48
		fallthrough
	case 14:
		k2 ^= uint64(tail[13]) << 40
		fallthrough
	case 13:
		k2 ^= uint64(tail[12]) << 32
		fallthrough
	case 12:
		k2 ^= uint64(tail[11]) << 24
		fallthrough
	case 11:
		k2 ^= uint64(tail[10]) << 16
		fallthrough
	case 10:
		k2 ^= uint64(tail[9]) << 8
		fallthrough
	case 9:
		k2 ^= uint64(tail[8]) << 0
		k2 *= c2
		k2 = rotl64(k2, 33)
		k2 *= c1
		h2 ^= k2
		fallthrough
	case 8:
		k1 ^= uint64(tail[7]) << 56
		fallthrough
	case 7:
		k1 ^= uint64(tail[6]) << 48
		fallthrough
	case 6:
		k1 ^= uint64(tail[5]) << 40
		fallthrough
	case 5:
		k1 ^= uint64(tail[4]) << 32
		fallthrough
	case 4:
		k1 ^= uint64(tail[3]) << 24
		fallthrough
	case 3:
		k1 ^= uint64(tail[2]) << 16
		fallthrough
	case 2:
		k1 ^= uint64(tail[1]) << 8
		fallthrough
	case 1:
		k1 ^= uint64(tail[0]) << 0
		k1 *= c1
		k1 = rotl64(k1, 31)
		k1 *= c2
		h1 ^= k1
	}

	//----------
	// finalization

	h1 ^= uint64(len(key))
	h2 ^= uint64(len(key))

	h1 += h2
	h2 += h1

	h1 = fmix64(h1)
	h2 = fmix64(h2)

	h1 += h2
	h2 += h1
	return
}

func murmurHash3_x64_128_C(key []byte, seed uint32) (h1, h2 uint64) {
	var ret [2]uint64
	var p unsafe.Pointer
	if len(key) == 0 {
		p = nil
	} else {
		p = unsafe.Pointer(&key[0])
	}
	C.MurmurHash3_x64_128(p, C.int(len(key)), C.uint32_t(seed), unsafe.Pointer(&ret[0]))
	return ret[0], ret[1]
}
