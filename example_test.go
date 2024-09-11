package client

import (
	"fmt"
	"log"
)

func exampleFallbackGet(key string) ([]byte, error) {
	return []byte("umem-cache"), nil
}

func exampleNilFallbackGet(key string) ([]byte, error) {
	return nil, nil
}

func ExampleClient_GetOrSet() {
	service, _ := globalSyncService.GetService()

	client, err := New(service, &globalSyncService)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	key := "hello"
	val, err := client.GetOrSet(key, exampleFallbackGet)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(val))
	// Output: umem-cache
}

func ExampleClient_DelForSet() {
	service, _ := globalSyncService.GetService()

	client, err := New(service, &globalSyncService)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	key := "hello"
	err = client.DelForSet(key, exampleFallbackGet)
	if err != nil {
		log.Fatal(err)
	}

	val, err := client.Get(key)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(val))

	err = client.DelForSet(key, exampleNilFallbackGet)
	if err != nil {
		log.Fatal(err)
	}

	val, err = client.Get(key)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%#v\n", val)
	// Output:
	// umem-cache
	// []byte(nil)
}
