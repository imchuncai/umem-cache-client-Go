// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2024, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package client

import (
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"strconv"
	"sync"
	"time"

	"math/rand"
	"testing"

	"github.com/imchuncai/umem-cache-client-Go/umem_cache"
)

// set seed to 0 for random seed
var seed int64 = 47

const N = 1000
const machine = "[::1]"

var threads = []Thread{
	{machine + ":47474", 0},
	{machine + ":47474", 1},
	{machine + ":47474", 2},
	{machine + ":47474", 3},
}

const serverMemory = 100 << 20
const valSizeLimit = 1 << 20
const benchRandSize = true

// const serverMemory = 100 << 20
// const valSizeLimit = 1 << 10
// const benchRandSize = true

// const serverMemory = 2 << 30
// const valSizeLimit = 1 << 20
// const benchRandSize = true

// const serverMemory = 2 << 30
// const valSizeLimit = 1 << 20
// const benchRandSize = false

const keySizeLimit = math.MaxUint8
const benchKeySizeLimit = 200
const benchHotCasePercent = 20
const benchHotAccessPercent = 80
const benchKVSize = serverMemory * 4

var kvPoll KVPool

var unstableSyncService UnstableSyncService

type UnstableSyncService struct {
	i uint32
}

func (s *UnstableSyncService) GetService() (Server, error) {
	s.i++
	return Server{s.i, threads}, nil
}

var stableSyncService StableSyncService

type StableSyncService struct{}

func (s *StableSyncService) GetService() (Server, error) {
	return Server{unstableSyncService.i, threads}, nil
}

func init() {
	if seed == 0 {
		seed = time.Now().Unix()
	}
	fmt.Printf("seed: %d\t"+"N: %d\n", seed, N)
	kvPoll = newKVPool()
}

type KVPool struct {
	r     *rand.Rand
	pool  []byte
	apoll []byte
}

func newKVPool() KVPool {
	var p KVPool
	p.r = rand.New(rand.NewSource(seed))
	var n int
	if keySizeLimit >= valSizeLimit {
		n = keySizeLimit << 1
	} else {
		n = valSizeLimit << 1
	}
	p.pool = make([]byte, n)
	apollN := benchKeySizeLimit - len(strconv.Itoa(math.MaxInt))
	p.apoll = make([]byte, apollN)
	for i := 0; i < n; i++ {
		p.pool[i] = byte(p.r.Intn(math.MaxUint8 + 1))
	}
	for i := 0; i < apollN; i++ {
		p.apoll[i] = 'a'
	}
	return p
}

func (p KVPool) randN(n int) []byte {
	i := p.r.Intn(len(p.pool) - n)
	return p.pool[i : i+n]
}

func (p KVPool) rand(limit int) []byte {
	n := p.r.Intn(limit + 1)
	return p.randN(n)
}

func (p KVPool) randA() []byte {
	i := p.r.Intn(len(p.apoll) + 1)
	return p.apoll[:i]
}

func (p KVPool) randTestCase() TestCase {
	return TestCase{
		p.rand(keySizeLimit),
		p.rand(len(p.pool) >> 1),
	}
}

func (p KVPool) randV(i int) TestCase {
	return TestCase{
		append([]byte(strconv.Itoa(i)), p.randA()...),
		p.rand(len(p.pool) >> 1),
	}
}

type TestCase struct {
	key []byte
	val []byte
}

func (tc TestCase) String() string {
	return fmt.Sprintf("key size: %d, value size: %d", len(tc.key), len(tc.val))
}

func newUnstableClient(tb testing.TB) *Client {
	client, err := New(&unstableSyncService, 0)
	if err != nil {
		tb.Fatalf("got error: %v", err)
	}
	return client
}

func newStableClient(tb testing.TB) *Client {
	client, err := New(&stableSyncService, 0)
	if err != nil {
		tb.Fatalf("got error: %v", err)
	}
	return client
}

func fuzz(f func(tc TestCase, randV []byte)) {
	keysN := []int{0, 255}
	valsN := []int{0, 1}
	for _, kn := range keysN {
		for _, vn := range valsN {
			key := kvPoll.randN(kn)
			val := kvPoll.randN(vn)
			val2 := kvPoll.randN(vn)
			f(TestCase{key, val}, val2)
		}
	}

	for i := 0; i < N; i++ {
		f(kvPoll.randTestCase(), kvPoll.randTestCase().val)
	}
}

var fallbackFuncNone = func(key []byte) (val []byte, err error) {
	return nil, nil
}

func get(tb testing.TB, client *Client, tc TestCase) {
	val, err := client.GetOrSet(tc.key, fallbackFuncNone)
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

func mustSet(client *Client, key []byte, val []byte) error {
	err := client.Del(key)
	if err != nil {
		return err
	}
	fallbackFunc := func(key []byte) ([]byte, error) {
		return val, nil
	}
	_, err = client.GetOrSet(key, fallbackFunc)
	return err
}

func set(tb testing.TB, client *Client, tc TestCase) {
	err := mustSet(client, tc.key, tc.val)
	if err != nil {
		tb.Fatalf("got error: %v, test case: %v", err, tc)
	}
}

func del(tb testing.TB, client *Client, tc TestCase) {
	err := client.Del(tc.key)
	if err != nil {
		tb.Fatalf("got error: %v, key size: %d", err, len(tc.key))
	}
}

var errIntentionally = errors.New("error intentionally")

func fallbackGet(tc TestCase) umem_cache.FallbackGetFunc {
	return func(key []byte) (val []byte, err error) {
		return tc.val, nil
	}
}

func badFallbackGet(key []byte) (val []byte, err error) {
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
	client := newUnstableClient(t)
	defer client.Close()

	fuzz(func(tc TestCase, _ []byte) {
		set(t, client, tc)
		get(t, client, tc)
	})
}

func TestDel(t *testing.T) {
	client := newUnstableClient(t)
	defer client.Close()

	fuzz(func(tc TestCase, _ []byte) {
		set(t, client, tc)
		del(t, client, tc)
		tc.val = nil
		get(t, client, tc)
	})
}

func TestGetOrSet(t *testing.T) {
	client := newUnstableClient(t)
	defer client.Close()

	fuzz(func(tc TestCase, randV []byte) {
		tc2 := TestCase{tc.key, randV}
		set(t, client, tc)
		getOrSet(t, client, tc2)
		get(t, client, tc)

		del(t, client, tc)
		getOrSet(t, client, tc2)
		get(t, client, tc2)
	})
}

func TestBadGetOrSet(t *testing.T) {
	client := newUnstableClient(t)
	defer client.Close()

	fuzz(func(tc TestCase, _ []byte) {
		del(t, client, tc)
		badGetOrSet(t, client, tc)
		tc.val = nil
		get(t, client, tc)
	})
}

func TestReconnect(t *testing.T) {
	tc := TestCase{[]byte("test_reconnect_key"), []byte("test_reconnect_value")}

	client := newUnstableClient(t)
	defer client.Close()

	set(t, client, tc)

	client2 := newUnstableClient(t)
	defer client2.Close()

	if client2.version <= client.version {
		t.Fatalf("want bigger client2 version, got client version: %d"+
			"client2 version: %d", client.version, client2.version)
	}
	get(t, client2, tc)
	get(t, client, tc)
}

func TestDiscardValue(t *testing.T) {
	client := newUnstableClient(t)
	defer client.Close()

	fuzz(func(tc TestCase, randV []byte) {
		del(t, client, tc)

		fallbackGet := func(key []byte) (val []byte, err error) {
			del(t, client, tc)
			return tc.val, nil
		}

		_, err := client.GetOrSet(tc.key, fallbackGet)
		if err != nil {
			t.Fatalf("got error: %v", err)
		}

		tc.val = nil
		get(t, client, tc)
	})
}

func TestSetFail(t *testing.T) {
	client := newUnstableClient(t)
	defer client.Close()

	fuzz(func(tc TestCase, randV []byte) {
		del(t, client, tc)

		fallbackGet := func(key []byte) (val []byte, err error) {
			del(t, client, tc)
			return tc.val, nil
		}

		_, err := client.GetOrSet(tc.key, fallbackGet)
		if err != nil {
			t.Fatalf("got error: %v", err)
		}

		tc.val = nil
		get(t, client, tc)
	})
}

func TestTimeout(t *testing.T) {
	client := newStableClient(t)
	defer client.Close()

	tc := kvPoll.randTestCase()
	fallbackGet := func(key []byte) (val []byte, err error) {
		return tc.val, nil
	}

	var wg sync.WaitGroup
	wg.Add(1)
	badFallbackGet := func(key []byte) (val []byte, err error) {
		go func() {
			client.GetOrSet(tc.key, fallbackGet)
			wg.Done()
		}()
		time.Sleep((2*3 + 1) * time.Second)
		return tc.val, nil
	}

	del(t, client, tc)
	_, err := client.GetOrSet(tc.key, badFallbackGet)
	if !errors.Is(err, io.EOF) {
		t.Fatalf("want error: %v, got: %v", net.ErrClosed, err)
	}
	wg.Wait()

	get(t, client, tc)
}
