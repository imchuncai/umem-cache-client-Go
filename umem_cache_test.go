// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2024, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package client

import "testing"

var umemCacheGetOrSet = func(client *Client, key []byte, fallbackVal func() []byte) error {
	fallbackGet := func([]byte) ([]byte, error) {
		return fallbackVal(), nil
	}
	_, err := client.GetOrSet(key, fallbackGet)
	return err
}

func BenchmarkPerformance(b *testing.B) {
	client := newUnstableClient(b)
	defer client.Close()

	performance(b, func(key []byte, i int, fallbackVal func() []byte) error {
		return umemCacheGetOrSet(client, key, fallbackVal)
	})
}

func BenchmarkGetOrSet(b *testing.B) {
	client := newUnstableClient(b)
	defer client.Close()

	performanceFast(b, func(key []byte, i int, fallbackVal func() []byte) error {
		return umemCacheGetOrSet(client, key, fallbackVal)
	})
}
