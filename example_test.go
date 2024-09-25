package client

import (
	"fmt"
	"log"
	"sync"
)

var exampleMu sync.Mutex
var exampleKey = "hello"
var exampleVal = []byte("umem-cache")

func ExampleClient_GetOrSet() {
	service, _ := globalSyncService.GetService()

	client, err := New(service, &globalSyncService)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	fallbackGet := func(key string) ([]byte, error) {
		exampleMu.Lock()
		defer exampleMu.Unlock()

		return exampleVal, nil
	}
	val, err := client.GetOrSet(exampleKey, fallbackGet)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(val))
	// Output: umem-cache
}

func ExampleClient_Set() {
	service, _ := globalSyncService.GetService()

	client, err := New(service, &globalSyncService)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// make sure the key will not cached during the update or delete
	exampleMu.Lock()
	defer exampleMu.Unlock()

	client.Del(exampleKey)
	// update or delete the key on the database
	exampleVal = []byte("umem-cache")
	// you can choose to re-cache immediately or at the next retrieval time

	val, err := client.Get(exampleKey)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%#v\n", val)
	// Output:
	// []byte(nil)
}
