// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2024, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package client

import (
	"testing"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
)

const memcacheAddress = machine + ":11211"

func BenchmarkMemcachedPerformance(b *testing.B) {
	client := memcache.New(memcacheAddress)
	if client == nil {
		b.Fatal("client is nil")
	}
	defer client.Close()

	client.Timeout = time.Second * 3

	performance(b, func(key []byte, i int, fallbackVal func() []byte) error {
		strKey := stringKey(key)
		_, err := client.Get(strKey)
		if err == nil || err != memcache.ErrCacheMiss {
			return err
		}

		val := fallbackVal()
		if val == nil {
			return nil
		}

		err = client.Set(&memcache.Item{Key: strKey, Value: val})
		if err != nil {
			// shut: SERVER_ERROR out of memory storing object
			time.Sleep(time.Second * 10)
		}
		return client.Set(&memcache.Item{Key: strKey, Value: val})
	})
}

func BenchmarkMemcachedGetOrSet(b *testing.B) {
	client := memcache.New(memcacheAddress)
	if client == nil {
		b.Fatal("client is nil")
	}
	defer client.Close()

	client.Timeout = time.Second * 3

	performanceFast(b, func(key []byte, i int, fallbackVal func() []byte) error {
		strKey := stringKey(key)
		_, err := client.Get(strKey)
		if err == nil || err != memcache.ErrCacheMiss {
			return err
		}

		client.Set(&memcache.Item{Key: strKey, Value: fallbackVal()})
		// to continue test, ignore error: SERVER_ERROR out of memory storing object
		return nil
	})
}
