// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2024, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package client

import (
	"fmt"
	"log"
	"sync"
)

var exampleMu sync.RWMutex
var exampleKey = []byte("hello")
var exampleVal = []byte("umem-cache")

func ExampleClient_GetOrSet() {
	client, err := New(&unstableSyncService, 0)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	fallbackGet := func(key []byte) ([]byte, error) {
		exampleMu.RLock()
		defer exampleMu.RUnlock()

		return exampleVal, nil
	}
	val, err := client.GetOrSet(exampleKey, fallbackGet)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(val))
	// Output: umem-cache
}

func ExampleClient_Del() {
	client, err := New(&unstableSyncService, 0)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// make sure the key will not cached during the deletion
	exampleMu.Lock()
	defer exampleMu.Unlock()

	err = client.Del(exampleKey)
	fmt.Printf("%v\n", err)
	// you can do some update to the key now, or just delete it.

	// Note: you have no reason to re-cache the key now, because if it is a
	// cold key, it may never be fetched, and if it is a hot key, it may
	// already have someone waiting on the lock to set it. If you insist to
	// do that, don't forget release the lock first.

	// Output:
	// <nil>
}
