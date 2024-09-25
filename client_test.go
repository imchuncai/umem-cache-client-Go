// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2024, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package client

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"math/rand"
	"testing"

	"github.com/imchuncai/umem-cache-client-Go/conn"
	"github.com/imchuncai/umem-cache-client-Go/umem_cache"
)

// set seed to 0 for random seed
var seed int64 = 47

const testMachine = "[::1]"

var threads = []Thread{
	{testMachine + ":47474", 0},
	{testMachine + ":47474", 1},
	{testMachine + ":47474", 2},
	{testMachine + ":47474", 3},
}

const N = 1000
const keySizeLimit = math.MaxUint8
const valSizeLimit = 1 << 20
const serverMemory = 2 << 30
const benchHotCasePercent = 20
const benchHotAccessPercent = 80

const benchCaseN = (serverMemory / valSizeLimit * 2) * 100 / benchHotCasePercent
const benchHotCaseN = (benchCaseN * benchHotCasePercent / 100)
const benchColdCaseN = benchCaseN - benchHotCaseN

var globalVal [valSizeLimit * 2]byte
var globalSyncService testSyncService

type testSyncService struct {
	i uint32
}

func (s *testSyncService) GetService() (Server, error) {
	s.i++
	return Server{s.i, threads}, nil
}

func init() {
	if seed == 0 {
		seed = time.Now().Unix()
	}
	fmt.Printf("seed: %d\t"+"N: %d\n", seed, N)

	r := rand.New(rand.NewSource(seed))
	for i := 0; i < valSizeLimit*2; i++ {
		globalVal[i] = byte(r.Intn(math.MaxUint8 + 1))
	}
}

type TestCase struct {
	key string
	val []byte
}

func (tc TestCase) String() string {
	return fmt.Sprintf("key size: %d, value size: %d", len(tc.key), len(tc.val))
}

type BenchTestCase struct {
	TestCase
	i int
}

func (tc BenchTestCase) size() int {
	return len(tc.key) + len(tc.val)
}

func (tc BenchTestCase) isHot() bool {
	return tc.i < benchHotCaseN
}

type BenchHelper struct {
	mu        sync.Mutex
	r         *rand.Rand
	hotCases  []BenchTestCase
	coldCases []BenchTestCase
}

func benchRandValue(r *rand.Rand) []byte {
	// satisfy memcached
	const max = valSizeLimit - 100
	val := randVal(r)
	if len(val) > max {
		return val[:max]
	}
	return val
}

func newBenchHelper() *BenchHelper {
	var helper BenchHelper
	helper.r = rand.New(rand.NewSource(seed))

	helper.hotCases = make([]BenchTestCase, benchHotCaseN)
	for i := 0; i < benchHotCaseN; i++ {
		helper.hotCases[i].i = i
		helper.hotCases[i].key = strconv.Itoa(i)
		helper.hotCases[i].val = benchRandValue(helper.r)
	}

	helper.coldCases = make([]BenchTestCase, benchColdCaseN)
	for i := 0; i < benchColdCaseN; i++ {
		helper.coldCases[i].i = i + benchHotCaseN
		helper.coldCases[i].key = strconv.Itoa(i + benchHotCaseN)
		helper.coldCases[i].val = benchRandValue(helper.r)
	}
	return &helper
}

func (h *BenchHelper) randHot() BenchTestCase {
	i := h.r.Int31n(benchHotCaseN)
	return h.hotCases[i]
}

func (h *BenchHelper) randCold() BenchTestCase {
	i := h.r.Int31n(benchColdCaseN)
	return h.coldCases[i]
}

func (h *BenchHelper) randCase() BenchTestCase {
	h.mu.Lock()
	defer h.mu.Unlock()

	i := h.r.Int31n(100)
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

func newTestClient(tb testing.TB) *Client {
	service, err := globalSyncService.GetService()
	if err != nil {
		tb.Fatalf("got error: %v", err)
	}

	client, err := New(service, &globalSyncService)
	if err != nil {
		tb.Fatalf("got error: %v", err)
	}
	return client
}

func randKeyN(r *rand.Rand, n int) string {
	i := r.Intn(valSizeLimit*2 - keySizeLimit)
	return string(globalVal[i : i+n])
}

func randKey(r *rand.Rand) string {
	n := r.Intn(keySizeLimit + 1)
	return randKeyN(r, n)
}

func randValN(r *rand.Rand, n int) []byte {
	i := r.Intn(valSizeLimit)
	return globalVal[i : i+n]
}

func randVal(r *rand.Rand) []byte {
	n := r.Intn(valSizeLimit + 1)
	return randValN(r, n)
}

func fuzz(f func(tc TestCase, randV []byte)) {
	r := rand.New(rand.NewSource(seed))
	keysN := []int{0, 255}
	valsN := []int{0, 47, 48, 49, 3575, 3576, 3577, valSizeLimit}
	for _, kn := range keysN {
		for _, vn := range valsN {
			key := randKeyN(r, kn)
			val := randValN(r, vn)
			val2 := randValN(r, vn)
			f(TestCase{key, val}, val2)
		}
	}

	for i := 0; i < N; i++ {
		key := randKey(r)
		val := randVal(r)
		val2 := randVal(r)
		f(TestCase{key, val}, val2)
	}
}

func get(tb testing.TB, client *Client, tc TestCase) {
	val, err := client.Get(tc.key)
	if err != nil {
		tb.Fatalf("got error: %v test case: %v", err, tc)
	}

	if val == nil {
		if tc.val != nil {
			tb.Fatalf("want %v, got: nil", tc)
		}
	} else if string(val) != string(tc.val) {
		tb.Fatalf("want: %v, got size: %d", tc, len(val))
	}
}

func set(tb testing.TB, client *Client, tc TestCase) {
	err := client.Set(tc.key, tc.val)
	if err != nil {
		tb.Fatalf("got error: %v, test case: %v", err, tc)
	}
}

func del(tb testing.TB, client *Client, key string) {
	err := client.Del(key)
	if err != nil {
		tb.Fatalf("got error: %v, key size: %d", err, len(key))
	}
}

var errIntentionally = errors.New("error intentionally")

func fallbackGet(tc TestCase) umem_cache.FallbackGetFunc {
	return func(key string) (val []byte, err error) {
		return tc.val, nil
	}
}

func badFallbackGet(key string) (val []byte, err error) {
	return nil, errIntentionally
}

func getOrSet(t *testing.T, client *Client, tc TestCase) {
	_, err := client.GetOrSet(tc.key, fallbackGet(tc))
	if err != nil {
		t.Fatalf("got error: %v test case: %v", err, tc)
	}
}

func badGetOrSet(t *testing.T, client *Client, tc TestCase) {
	_, err := client.GetOrSet(tc.key, badFallbackGet)
	if !errors.Is(err, errIntentionally) {
		t.Fatalf("want error: %v, got error: %v test case: %v",
			errIntentionally, err, tc)
	}
}

func TestGet(t *testing.T) {
	client := newTestClient(t)
	defer client.Close()

	fuzz(func(tc TestCase, _ []byte) {
		set(t, client, tc)
		get(t, client, tc)
	})
}

func TestDel(t *testing.T) {
	client := newTestClient(t)
	defer client.Close()

	fuzz(func(tc TestCase, _ []byte) {
		set(t, client, tc)
		del(t, client, tc.key)
		tc.val = nil
		get(t, client, tc)
	})
}

func TestGetOrSet(t *testing.T) {
	client := newTestClient(t)
	defer client.Close()

	fuzz(func(tc TestCase, randV []byte) {
		tc2 := TestCase{tc.key, randV}
		set(t, client, tc)
		getOrSet(t, client, tc2)
		get(t, client, tc)

		del(t, client, tc.key)
		getOrSet(t, client, tc2)
		get(t, client, tc2)
	})
}

func TestBadGetOrSet(t *testing.T) {
	client := newTestClient(t)
	defer client.Close()

	fuzz(func(tc TestCase, _ []byte) {
		del(t, client, tc.key)
		badGetOrSet(t, client, tc)
		tc.val = nil
		get(t, client, tc)
	})
}

func TestReconnect(t *testing.T) {
	client := newTestClient(t)
	defer client.Close()

	badVal := globalVal[:valSizeLimit+1]
	fuzz(func(tc TestCase, randV []byte) {
		err := client.Set(tc.key, badVal)
		if err == nil {
			t.Fatalf("want error got nil")
		}

		set(t, client, tc)
		get(t, client, tc)
	})
}

func TestDiscardValue(t *testing.T) {
	fuzz(func(tc TestCase, randV []byte) {
		s, err := globalSyncService.GetService()
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		thread := threads[0]
		c1, err := umem_cache.Dial(thread.Address, thread.ThreadID, s.Version)
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		defer c1.Close()

		resp1 := c1.Del(tc.key)
		if resp1.Err != nil {
			t.Fatalf("got error: %v", resp1.Err)
		}

		err = c1.GetSend(tc.key, umem_cache.GET_FLAG_SET_ON_MISS)
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		resp2 := c1.GetRecv()
		if resp1.Err != nil {
			t.Fatalf("got error: %v", resp1.Err)
		}
		// now key is locked for set

		c2, err := conn.NewFullDuplex(thread.Address, thread.ThreadID, s.Version)
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		defer c2.Close()

		dupResp := c2.Set(tc.key, randV, true)
		if dupResp.Err != nil {
			t.Fatalf("got error: %v", dupResp)
		}
		if !dupResp.WillBlock {
			t.Fatalf("want WillBlock: true, got: false")
		}

		// test the nil value
		resp3 := c1.GetSet(tc.key, func(key string) (val []byte, err error) {
			return nil, nil
		})
		if resp3.Err != nil {
			t.Fatalf("got error: %v", resp2.Err)
		}

		resp4 := c1.Get(tc.key, 0)
		if resp4.Err != nil {
			t.Fatalf("got error: %v", resp2.Err)
		}
		if resp4.Val != nil {
			t.Fatalf("want nil val, got %d", len(resp3.Val))
		}
	})
}

func percent(i, n int) float64 {
	return float64(i) / float64(n) * 100
}

type ServerCachedBytes func(b *testing.B, key string) int
type GetOrSetFunc func(key string, f umem_cache.FallbackGetFunc) error

func performance(b *testing.B, getOrSet GetOrSetFunc, serverCachedBytes ServerCachedBytes) {
	helper := newBenchHelper()

	visited := make([]int32, benchCaseN)
	var miss, hotMiss int32

	b.RunParallel(func(p *testing.PB) {
		_visited := make([]int32, benchCaseN)
		var _miss, __hotMiss int32
		for p.Next() {
			tc := helper.randCase()
			_visited[tc.i]++
			fallbackGet := func(string) ([]byte, error) {
				_miss++
				if tc.isHot() {
					__hotMiss++
				}
				return tc.val, nil
			}
			err := getOrSet(tc.key, fallbackGet)
			if err != nil {
				b.Fatalf("got error: %v", err)
			}
		}

		for i, n := range _visited {
			atomic.AddInt32(&visited[i], n)
		}
		atomic.AddInt32(&miss, _miss)
		atomic.AddInt32(&hotMiss, __hotMiss)
	})

	var get, getHot, hotInvolved int32
	for _, n := range visited[:benchHotCaseN] {
		if n > 0 {
			hotInvolved++
			get += n
			getHot += n
		}
	}
	var coldInvolved int32
	for _, n := range visited[benchHotCaseN:] {
		if n > 0 {
			coldInvolved++
			get += n
		}
	}

	if get <= 1 {
		// benchmark is called twice, drop the first
		return
	}

	var head string
	if get-getHot < benchColdCaseN {
		head = "\n==================small benchtime===================\n"
	} else {
		head = "\n====================================================\n"
	}

	// first get always miss, drop that
	get -= hotInvolved + coldInvolved
	miss -= hotInvolved + coldInvolved
	getHot -= hotInvolved
	hotMiss -= hotInvolved

	hotBytes := helper.hotBytes()
	coldBytes := helper.coldBytes()
	totalBytes := hotBytes + coldBytes

	var hotCachedN, hotCachedBytes int
	for _, tc := range helper.hotCases {
		if visited[tc.i] == 0 {
			continue
		}
		if n := serverCachedBytes(b, tc.key); n > 0 {
			hotCachedN++
			hotCachedBytes += n
		}
	}
	var coldCachedN, coldCachedBytes int
	for _, tc := range helper.coldCases {
		if visited[tc.i] == 0 {
			continue
		}
		if n := serverCachedBytes(b, tc.key); n > 0 {
			coldCachedN++
			coldCachedBytes += n
		}
	}
	cached := hotCachedBytes + coldCachedBytes

	b.Logf(head+
		" case:%6d"+"     hot:%6d(%d%%)"+"    hot_access: %d%% \n"+
		"  get:%6d"+"    miss:%6d"+"    hit_rate: %.0f%% \n"+
		"  hot:%6d"+"    miss:%6d"+"    hit_rate: %.0f%% \n"+
		" cached:%5dm(%5dm %3.0f%%  -  %5dm %3.0f%%)\n"+
		"    hot:%5dm(%5dm %3.0f%%)%8d(%5d %3.0f%%)      \n"+
		"   cold:%5dm(%5dm %3.0f%%)%8d(%5d %3.0f%%)      \n"+
		"====================================================\n",
		benchCaseN, benchHotCaseN, benchHotCasePercent, benchHotAccessPercent,
		get, miss, percent(int(get-miss), int(get)),
		getHot, hotMiss, percent(int(getHot-hotMiss), int(getHot)),
		cached>>20, totalBytes>>20, percent(cached, totalBytes),
		serverMemory>>20, percent(cached, serverMemory),
		hotCachedBytes>>20, hotBytes>>20, percent(hotCachedBytes, hotBytes),
		hotCachedN, benchHotCaseN, percent(hotCachedN, benchHotCaseN),
		coldCachedBytes>>20, coldBytes>>20, percent(coldCachedBytes, coldBytes),
		coldCachedN, benchColdCaseN, percent(coldCachedN, benchColdCaseN))
}

func BenchmarkPerformance(b *testing.B) {
	client := newTestClient(b)
	defer client.Close()

	performance(b,
		func(key string, fallbackGet umem_cache.FallbackGetFunc) error {
			_, err := client.GetOrSet(key, fallbackGet)
			return err
		},
		func(b *testing.B, key string) int {
			val, err := client.Get(key)
			if err != nil {
				b.Fatalf("got error: %v", err)
			}
			if val != nil {
				return len(key) + len(val)
			}
			return 0
		},
	)
}

func performanceFast(b *testing.B, getOrSet GetOrSetFunc) {
	helper := newBenchHelper()
	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			tc := helper.randCase()
			fallbackGet := func(string) ([]byte, error) {
				return tc.val, nil
			}
			err := getOrSet(tc.key, fallbackGet)
			if err != nil {
				b.Fatalf("got error: %v", err)
			}
		}
	})
}

func BenchmarkGetOrSet(b *testing.B) {
	client := newTestClient(b)
	defer client.Close()

	performanceFast(b, func(key string, fallbackGet umem_cache.FallbackGetFunc) error {
		_, err := client.GetOrSet(key, fallbackGet)
		return err
	})
}
