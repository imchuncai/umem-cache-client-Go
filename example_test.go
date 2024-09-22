package client

import (
	"fmt"
	"log"
	"sync"
)

func ExampleClient_GetOrSet() {
	service, _ := globalSyncService.GetService()

	client, err := New(service, &globalSyncService)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	key := "hello"
	fallbackGet := func(key string) ([]byte, error) {
		return []byte("umem-cache"), nil
	}
	val, err := client.GetOrSet(key, fallbackGet)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(val))
	// Output: umem-cache
}

// When we update the backend database, we will not update the cache at the
// same time. Instead, we will delete the cache before updating and cache it
// again the next time we retrieve it.
func ExampleClient_Set() {
	service, _ := globalSyncService.GetService()

	client, err := New(service, &globalSyncService)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	type db struct {
		mu  sync.RWMutex
		key string
	}

	db1 := db{key: "hello"}

	// make sure the key will not cached during the update or delete
	db1.mu.Lock()
	defer db1.mu.Unlock()

	client.Del(db1.key)
	// update or delete the key on the database

	val, err := client.Get(db1.key)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%#v\n", val)
	// Output:
	// []byte(nil)
}
