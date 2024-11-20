// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2024, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package client

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"unsafe"
)

func stringKey(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

type BenchTestCase struct {
	TestCase
	i   int
	hot bool
}

func (tc BenchTestCase) size() int {
	return len(tc.key) + len(tc.val)
}

type BenchHelper struct {
	mu        sync.Mutex
	r         *rand.Rand
	hotCases  []BenchTestCase
	coldCases []BenchTestCase
}

func newRandSizeBenchHelper() *BenchHelper {
	caseN := benchKVSize / ((benchKeySizeLimit + valSizeLimit) / 2)
	hotCaseN := caseN * benchHotCasePercent / 100
	coldCaseN := caseN - hotCaseN

	var helper BenchHelper
	helper.r = rand.New(rand.NewSource(seed))

	helper.hotCases = make([]BenchTestCase, hotCaseN)
	for i := 0; i < hotCaseN; i++ {
		helper.hotCases[i].i = i
		helper.hotCases[i].hot = true
		helper.hotCases[i].TestCase = kvPoll.randV(i)
	}

	helper.coldCases = make([]BenchTestCase, coldCaseN)
	for i := 0; i < coldCaseN; i++ {
		helper.coldCases[i].i = i + coldCaseN
		helper.coldCases[i].hot = false
		helper.coldCases[i].TestCase = kvPoll.randV(i + coldCaseN)
	}

	if !benchRandSize {
		n := (benchKeySizeLimit + valSizeLimit) / 2
		for i := range helper.hotCases {
			helper.hotCases[i].val = kvPoll.randN(n)
		}
		for i := range helper.coldCases {
			helper.coldCases[i].val = kvPoll.randN(n)
		}
	}
	return &helper
}

func (h *BenchHelper) randHot() BenchTestCase {
	i := h.r.Intn(len(h.hotCases))
	return h.hotCases[i]
}

func (h *BenchHelper) randCold() BenchTestCase {
	i := h.r.Intn(len(h.coldCases))
	return h.coldCases[i]
}

func (h *BenchHelper) randCase() BenchTestCase {
	h.mu.Lock()
	defer h.mu.Unlock()

	i := h.r.Intn(100)
	if i < benchHotAccessPercent {
		return h.randHot()
	}
	return h.randCold()
}

func (h *BenchHelper) hotBytes() int {
	hotBytes := 0
	for _, tc := range h.hotCases {
		hotBytes += tc.size()
	}
	return hotBytes
}

func (h *BenchHelper) coldBytes() int {
	coldBytes := 0
	for _, tc := range h.coldCases {
		coldBytes += tc.size()
	}
	return coldBytes
}

func percent(i, n int) float64 {
	return float64(i) / float64(n) * 100
}

type GetOrSetFunc func(key []byte, i int, f func() []byte) error

func performance(b *testing.B, getOrSet GetOrSetFunc) {
	helper := newRandSizeBenchHelper()

	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			tc := helper.randCase()
			fallbackGet := func() []byte {
				return tc.val
			}
			err := getOrSet(tc.key, tc.i, fallbackGet)
			if err != nil {
				b.Fatalf("got error: %v", err)
			}
		}
	})

	var miss, hotMiss, hotGet int32
	b.RunParallel(func(p *testing.PB) {
		var __miss, __hotMiss, __hotGet int32
		for p.Next() {
			tc := helper.randCase()
			if tc.hot {
				__hotGet++
			}
			fallbackVal := func() []byte {
				__miss++
				if tc.hot {
					__hotMiss++
				}
				return tc.val
			}
			err := getOrSet(tc.key, tc.i, fallbackVal)
			if err != nil {
				b.Fatalf("got error: %v", err)
			}
		}

		atomic.AddInt32(&hotGet, __hotGet)
		atomic.AddInt32(&miss, __miss)
		atomic.AddInt32(&hotMiss, __hotMiss)
	})

	if b.N == 1 {
		// benchmark is called twice, drop the first
		return
	}

	hotBytes := helper.hotBytes()
	coldBytes := helper.coldBytes()
	totalBytes := hotBytes + coldBytes

	var hotCachedN, hotCachedBytes int
	for _, tc := range helper.hotCases {
		hit := true
		fallbackNil := func() []byte {
			hit = false
			return nil
		}
		err := getOrSet(tc.key, tc.i, fallbackNil)
		if err != nil {
			b.Fatalf("got error: %v", err)
		}
		if hit {
			hotCachedN++
			hotCachedBytes += tc.size()
		}
	}
	var coldCachedN, coldCachedBytes int
	for _, tc := range helper.coldCases {
		hit := true
		fallbackNil := func() []byte {
			hit = false
			return nil
		}
		err := getOrSet(tc.key, tc.i, fallbackNil)
		if err != nil {
			b.Fatalf("got error: %v", err)
		}
		if hit {
			coldCachedN++
			coldCachedBytes += tc.size()
		}
	}
	cached := hotCachedBytes + coldCachedBytes

	b.Logf("\n=======================================================\n"+
		" case:%8d"+"    hot:%8d(%d%%)"+"    hot_access: %d%% \n"+
		"  get:%8d"+"    hit:%8d"+"    hit_rate: %.2f%% \n"+
		"  hot:%8d"+"    hit:%8d"+"    hit_rate: %.2f%% \n"+
		" cached:%5dm(%5dm %3.0f%%  -  %5dm %3.0f%%)\n"+
		"    hot:%5dm(%5dm %3.0f%%)%8d(%5d %3.0f%%)      \n"+
		"   cold:%5dm(%5dm %3.0f%%)%8d(%5d %3.0f%%)      \n"+
		"=======================================================\n",
		len(helper.hotCases)+len(helper.coldCases), len(helper.hotCases),
		benchHotCasePercent, benchHotAccessPercent,
		b.N, int32(b.N)-miss, percent(int(int32(b.N)-miss), int(b.N)),
		hotGet, hotGet-hotMiss, percent(int(hotGet-hotMiss), int(hotGet)),
		cached>>20, totalBytes>>20, percent(cached, totalBytes),
		serverMemory>>20, percent(cached, serverMemory),
		hotCachedBytes>>20, hotBytes>>20, percent(hotCachedBytes, hotBytes),
		hotCachedN, len(helper.hotCases), percent(hotCachedN, len(helper.hotCases)),
		coldCachedBytes>>20, coldBytes>>20, percent(coldCachedBytes, coldBytes),
		coldCachedN, len(helper.coldCases), percent(coldCachedN, len(helper.coldCases)))
}

func performanceFast(b *testing.B, getOrSet GetOrSetFunc) {
	helper := newRandSizeBenchHelper()
	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			tc := helper.randCase()
			fallbackGet := func() []byte {
				return tc.val
			}
			err := getOrSet(tc.key, tc.i, fallbackGet)
			if err != nil {
				b.Fatalf("got error: %v", err)
			}
		}
	})
}
