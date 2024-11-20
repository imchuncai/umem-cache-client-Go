.. SPDX-License-Identifier: BSD-3-Clause
.. Copyright (C) 2024, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

===============================================
CACHE PERFORMANCE COMPARISON (512KB FIXED SIZE)
===============================================
We compared the cache performance of UMEM-CACHE, MEMCACHED and REDIS. Server
side memory limit is set to 2GB, and test case size is set to 4 times of that,
and value size is all 512KB. we first get the value from the server, if the
get missed, we stored it. And 80% of the time the first 20% of the test cases
are used. We first make N requests to warm up the cache, and then make N more
requests to collect statistics.

CONCLUSION
----------
The test results showed that the performance of UMEM-CACHE and MEMCACHE are
very close. MEMCACHED has a hit rate about 4% higher than UMEM-CACHE. REDIS has
a very bad hit rate, is about 17% lower than UMEM-CACHE. And REDIS only used
81% of the configured memory.

TEST MACHINE
------------
Two 4GB version of Raspberry Pi 4 Model B connected in LAN with Gigabit network.
One used as a server and the other as a client. And the installed operating
system is Fedora-Server-40-1.14.aarch64.

MEMCACHED
---------
commit: 5609673ed29db98a377749fab469fe80777de8fd

RUN COMMAND
~~~~~~~~~~~
::

	./memcached --conn-limit=512 --memory-limit=2048

TEST COMMAND
~~~~~~~~~~~~
::

	go test -timeout=20h -run=BenchmarkMemcachedPerformance		       \
	-bench=BenchmarkMemcachedPerformance -benchtime=50000x -cpu=8

TEST RESULT
~~~~~~~~~~~
::

	seed: 47	N: 1000
	goos: linux
	goarch: arm64
	pkg: github.com/imchuncai/umem-cache-client-Go
	BenchmarkMemcachedPerformance-8   	   50000	   7979388 ns/op
	--- BENCH: BenchmarkMemcachedPerformance-8
	bench_test.go:194: 
		=======================================================
		case:   16380    hot:    3276(20%)    hot_access: 80% 
		get:   50000    hit:   35477    hit_rate: 70.95% 
		hot:   39949    hit:   34516    hit_rate: 86.40% 
		cached: 2041m( 8193m  25%  -   2048m 100%)
		hot: 1406m( 1638m  86%)    2811( 3276  86%)      
		cold:  635m( 6554m  10%)    1271(13104  10%)      
		=======================================================
	PASS
	ok  	github.com/imchuncai/umem-cache-client-Go	399.088s

	VmHWM:	 2108840 kB

UMEM-CACHE
----------
commit: 5243e88e9300b15bcd106ecc88c8d864296f2da8

BUILT CONFIG
~~~~~~~~~~~~
::

	#define CONFIG_MEM_LIMIT	((uint64_t)2 << 30 >> PAGE_SHIFT)

RUN COMMAND
~~~~~~~~~~~
::

	./umem-cache

TEST COMMAND
~~~~~~~~~~~~
::

	go test -timeout=20h -run=BenchmarkPerformance			       \
	-bench=BenchmarkPerformance -benchtime=50000x -cpu=8

TEST RESULT
~~~~~~~~~~~
::

	seed: 47	N: 1000
	goos: linux
	goarch: arm64
	pkg: github.com/imchuncai/umem-cache-client-Go
	BenchmarkPerformance-8   	   50000	   6983944 ns/op
	--- BENCH: BenchmarkPerformance-8
	bench_test.go:194: 
		=======================================================
		case:   16380    hot:    3276(20%)    hot_access: 80% 
		get:   50000    hit:   34248    hit_rate: 68.50% 
		hot:   39949    hit:   33198    hit_rate: 83.10% 
		cached: 2046m( 8193m  25%  -   2048m 100%)
		hot: 1349m( 1638m  82%)    2698( 3276  82%)      
		cold:  697m( 6554m  11%)    1394(13104  11%)      
		=======================================================
	PASS
	ok  	github.com/imchuncai/umem-cache-client-Go	349.298s

	VmHWM:	 2097940 kB

REDIS
---------
version: 7.4.1
commit: 74b289a0e12f9f65a6daeec6a66cadc76792f644

RUN COMMAND
~~~~~~~~~~~
::

	./src/redis-server --protected-mode no --appendonly no --save ""       \
	--maxmemory 2gb --maxclients 512 --maxmemory-policy allkeys-lru

TEST COMMAND
~~~~~~~~~~~~
::

	go test -timeout=20h -run=BenchmarkRedisPerformance		       \
	-bench=BenchmarkRedisPerformance -benchtime=50000x -cpu=8

TEST RESULT
~~~~~~~~~~~
::

	seed: 47	N: 1000
	goos: linux
	goarch: arm64
	pkg: github.com/imchuncai/umem-cache-client-Go
	BenchmarkRedisPerformance-8   	   50000	   6209898 ns/op
	--- BENCH: BenchmarkRedisPerformance-8
	bench_test.go:194: 
		=======================================================
		case:   16380    hot:    3276(20%)    hot_access: 80% 
		get:   50000    hit:   28475    hit_rate: 56.95% 
		hot:   39949    hit:   27714    hit_rate: 69.37% 
		cached: 1633m( 8193m  20%  -   2048m  80%)
		hot: 1124m( 1638m  69%)    2248( 3276  69%)      
		cold:  509m( 6554m   8%)    1018(13104   8%)      
		=======================================================
	PASS
	ok  	github.com/imchuncai/umem-cache-client-Go	310.598s

	VmHWM:	 1704340 kB
