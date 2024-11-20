.. SPDX-License-Identifier: BSD-3-Clause
.. Copyright (C) 2024, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

==============================================
CACHE PERFORMANCE COMPARISON (1M LIMIT RANDOM)
==============================================
We compared the cache performance of UMEM-CACHE, MEMCACHED and REDIS. Server
side memory limit is set to 2GB, and test case size is set to 4 times of that,
and value size is random in the range[0, 1m]. we first get the value from the
server, if the get missed, we stored it. And 80% of the time the first 20% of
the test cases are used. We first make N requests to warm up the cache, and
then make N more requests to collect statistics.

CONCLUSION
----------
The test results showed that the performance of UMEM-CACHE, MEMCACHE and REDIS
are very close, UMEM-CACHE is the best, MEMCACHED is the second, and REDIS is
the worst. UMEM-CACHE has a hit rate about 8% higher than REDIS, while using
about 2% less memory.

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

	./memcached --conn-limit=512 --memory-limit=2048 --max-item-size=2m

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
	BenchmarkMemcachedPerformance-8   	   50000	   8389995 ns/op
	--- BENCH: BenchmarkMemcachedPerformance-8
	bench_test.go:194: 
		=======================================================
		case:   16380    hot:    3276(20%)    hot_access: 80% 
		get:   50000    hit:   32587    hit_rate: 65.17% 
		hot:   39949    hit:   31688    hit_rate: 79.32% 
		cached: 1816m( 8156m  22%  -   2048m  89%)
		hot: 1295m( 1661m  78%)    2586( 3276  79%)      
		cold:  520m( 6494m   8%)    1248(13104  10%)      
		=======================================================
	PASS
	ok  	github.com/imchuncai/umem-cache-client-Go	419.653s

	VmHWM:	 2110892 kB

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
	BenchmarkPerformance-8   	   50000	   7141389 ns/op
	--- BENCH: BenchmarkPerformance-8
	bench_test.go:194: 
		=======================================================
		case:   16380    hot:    3276(20%)    hot_access: 80% 
		get:   50000    hit:   34030    hit_rate: 68.06% 
		hot:   39949    hit:   32986    hit_rate: 82.57% 
		cached: 2042m( 8156m  25%  -   2048m 100%)
		hot: 1358m( 1661m  82%)    2689( 3276  82%)      
		cold:  684m( 6494m  11%)    1376(13104  11%)      
		=======================================================
	PASS
	ok  	github.com/imchuncai/umem-cache-client-Go	357.180s

	VmHWM:	 2098352 kB

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
	BenchmarkRedisPerformance-8   	   50000	   6681040 ns/op
	--- BENCH: BenchmarkRedisPerformance-8
	bench_test.go:194: 
		=======================================================
		case:   16380    hot:    3276(20%)    hot_access: 80% 
		get:   50000    hit:   31434    hit_rate: 62.87% 
		hot:   39949    hit:   30497    hit_rate: 76.34% 
		cached: 1888m( 8156m  23%  -   2048m  92%)
		hot: 1265m( 1661m  76%)    2501( 3276  76%)      
		cold:  622m( 6494m  10%)    1254(13104  10%)      
		=======================================================
	PASS
	ok  	github.com/imchuncai/umem-cache-client-Go	334.166s

	VmHWM:	 2146608 kB
