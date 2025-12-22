// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2025, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package client

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

var exampleMu sync.RWMutex
var exampleKey = []byte("hello")
var exampleVal = []byte("umem-cache")

func exampleConfig() (Config, error) {
	cert, err := tls.LoadX509KeyPair("cert.pem", "key.pem")
	if err != nil {
		return Config{}, fmt.Errorf("load cert.pem and key.pem failed: %w", err)
	}

	caCert, err := os.ReadFile("ca-cert.pem")
	if err != nil {
		return Config{}, fmt.Errorf("read ca-cert.pem failed: %w", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Note: set TLSConfig to nil if tls is not enabled
	return Config{
		Timeout:  3 * time.Second,
		ThreadNR: 4,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      caCertPool,
		},
	}, nil
}

func exampleClient() (*Client, error) {
	config, err := exampleConfig()
	if err != nil {
		return nil, fmt.Errorf("get example config failed: %w", err)
	}

	return New("[::1]:10047", config)
}

func exampleCluster() (*Cluster, error) {
	config, err := exampleConfig()
	if err != nil {
		return nil, fmt.Errorf("get example config failed: %w", err)
	}

	addresses := []string{
		"[::1]:10048",
		"[::1]:10050",
		"[::1]:10052",
		"[::1]:10054",
	}
	err = AdminInitCluster(DEADLINE(), addresses, config.TLSConfig)
	if err != nil {
		return nil, fmt.Errorf("init cluster failed: %w", err)
	}

	return NewCluster([]string{
		"[::1]:10047",
		"[::1]:10049",
		"[::1]:10051",
		"[::1]:10053",
	}, config)
}

func exampleGetOrSet(client ClientInterface) {
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
}

func exampleDel(client ClientInterface) {
	// make sure the key will not cached during the deletion
	exampleMu.Lock()
	defer exampleMu.Unlock()

	err := client.Del(exampleKey)
	fmt.Printf("%v\n", err)
	// you can do some update to the key now, or just delete it.

	// Note: you have no reason to re-cache the key now, because if it is a
	// cold key, it may never be fetched, and if it is a hot key, it may
	// already have someone waiting on the lock to set it. If you insist to
	// do that, don't forget release the lock first.
}

func ExampleClient() {
	client, err := exampleClient()
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	exampleGetOrSet(client)
	// Output: umem-cache
}

func ExampleClient_Del() {
	client, err := exampleClient()
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	exampleDel(client)
	// Output:
	// <nil>
}

func ExampleCluster_GetOrSet() {
	cluster, err := exampleCluster()
	if err != nil {
		log.Fatal(err)
	}
	defer cluster.Close()

	exampleGetOrSet(cluster)
	// Output: umem-cache
}

func ExampleCluster_Del() {
	cluster, err := exampleCluster()
	if err != nil {
		log.Fatal(err)
	}
	defer cluster.Close()

	exampleDel(cluster)
	// Output:
	// <nil>
}
