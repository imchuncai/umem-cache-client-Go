// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2024, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package client

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
)

var redisAddress = []string{
	machine + ":6379",
	machine + ":6380",
	machine + ":6381",
	machine + ":6382",
}

var redisGetOrSet = func(client *redis.Client, key []byte, fallbackVal func() []byte) error {
	strKey := stringKey(key)
	err := client.Get(context.Background(), strKey).Err()
	if err == nil || err != redis.Nil {
		return err
	}

	val := fallbackVal()
	if val == nil {
		return nil
	}

	return client.Set(context.Background(), strKey, val, 0).Err()
}

func BenchmarkRedisPerformance(b *testing.B) {
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddress[0],
		Password: "",
		DB:       0,
	})
	defer client.Close()

	performance(b, func(key []byte, i int, fallbackVal func() []byte) error {
		return redisGetOrSet(client, key, fallbackVal)
	})
}

func BenchmarkRedisGetOrSet(b *testing.B) {
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddress[0],
		Password: "",
		DB:       0,
	})
	defer client.Close()

	performanceFast(b, func(key []byte, i int, fallbackVal func() []byte) error {
		return redisGetOrSet(client, key, fallbackVal)
	})
}

func benchmarkRedisGetOrSetN(b *testing.B, n int) {
	var clients = make([]*redis.Client, n)
	for i := 0; i < n; i++ {
		clients[i] = redis.NewClient(&redis.Options{
			Addr:     redisAddress[i],
			Password: "",
			DB:       0,
		})
	}
	defer func() {
		for i := 0; i < n; i++ {
			clients[i].Close()
		}
	}()

	performanceFast(b, func(key []byte, i int, fallbackVal func() []byte) error {
		return redisGetOrSet(clients[i%n], key, fallbackVal)
	})
}

func BenchmarkRedisGetOrSet2(b *testing.B) {
	benchmarkRedisGetOrSetN(b, 2)
}

func BenchmarkRedisGetOrSet3(b *testing.B) {
	benchmarkRedisGetOrSetN(b, 3)
}

func BenchmarkRedisGetOrSet4(b *testing.B) {
	benchmarkRedisGetOrSetN(b, 4)
}
