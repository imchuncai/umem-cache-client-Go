// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2024, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package client

import (
	"testing"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/imchuncai/umem-cache-client-Go/umem_cache"
)

const memcacheAddress = testMachine + ":11211"

func BenchmarkMemcachedPerformance(b *testing.B) {
	client := memcache.New(memcacheAddress)
	if client == nil {
		b.Fatal("client is nil")
	}
	defer client.Close()

	client.Timeout = time.Second * 3

	performance(b,
		func(key string, fallbackGet umem_cache.FallbackGetFunc) error {
			_, err := client.Get(key)
			if err == nil || err != memcache.ErrCacheMiss {
				return err
			}

			val, err := fallbackGet(key)
			if err != nil {
				return err
			}

			err = client.Set(&memcache.Item{Key: key, Value: val})
			if err != nil {
				// shut: SERVER_ERROR out of memory storing object
				time.Sleep(time.Second * 10)
			}
			return client.Set(&memcache.Item{Key: key, Value: val})
		},
		func(b *testing.B, key string) int {
			item, err := client.Get(key)
			if err == nil {
				return len(key) + len(item.Value)
			}
			if err != memcache.ErrCacheMiss {
				b.Fatalf("got error: %v", err)
			}
			return 0
		},
	)
}

func BenchmarkMemcachedGetOrSet(b *testing.B) {
	client := memcache.New(memcacheAddress)
	if client == nil {
		b.Fatal("client is nil")
	}
	defer client.Close()

	client.Timeout = time.Second * 3

	performanceFast(b, func(key string, fallbackGet umem_cache.FallbackGetFunc) error {
		_, err := client.Get(key)
		if err == nil || err != memcache.ErrCacheMiss {
			return err
		}

		val, err := fallbackGet(key)
		if err != nil {
			return err
		}

		client.Set(&memcache.Item{Key: key, Value: val})
		// to continue test, ignore error: SERVER_ERROR out of memory storing object
		return nil
	})
}
