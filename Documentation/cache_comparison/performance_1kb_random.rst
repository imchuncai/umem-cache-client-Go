.. SPDX-License-Identifier: BSD-3-Clause
.. Copyright (C) 2024, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

===============================================
CACHE PERFORMANCE COMPARISON (1KB LIMIT RANDOM)
===============================================
We compared the cache performance of UMEM-CACHE, MEMCACHED and REDIS. Server
side memory limit is set to 100MB, and test case size is set to 4 times of that,
and value size is random in the range[0, 1k]. we first get the value from the
server, if the get missed, we stored it. And 80% of the time the first 20% of
the test cases are used. We first make N requests to warm up the cache, and
then make N more requests to collect statistics.

CONCLUSION
----------
The test results showed that the performance of UMEM-CACHE, MEMCACHE and REDIS
are very close, UMEM-CACHE is the best, MEMCACHED is the second, and REDIS is
the worst. UMEM-CACHE has a hit rate about 4% higher than REDIS, while using
about 6% less memory.

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

	./memcached --conn-limit=512 --memory-limit=100

TEST COMMAND
~~~~~~~~~~~~
::

	go test -timeout=20h -run=BenchmarkMemcachedPerformance		       \
	-bench=BenchmarkMemcachedPerformance -benchtime=2000000x -cpu=8

TEST RESULT
~~~~~~~~~~~
::

	seed: 47	N: 1000
	goos: linux
	goarch: arm64
	pkg: github.com/imchuncai/umem-cache-client-Go
	BenchmarkMemcachedPerformance-8   	 2000000	    286130 ns/op
	--- BENCH: BenchmarkMemcachedPerformance-8
	bench_test.go:194: 
		=======================================================
		case:  685343    hot:  137068(20%)    hot_access: 80% 
		get: 2000000    hit: 1193650    hit_rate: 59.68% 
		hot: 1599997    hit: 1160633    hit_rate: 72.54% 
		cached:   77m(  397m  20%  -    100m  78%)
		hot:   56m(   79m  71%)   99990(137068  73%)      
		cold:   21m(  318m   7%)   49465(548275   9%)      
		=======================================================
	PASS
	ok  	github.com/imchuncai/umem-cache-client-Go	573.031s

	VmHWM:	  106648 kB

UMEM-CACHE
----------
commit: 5243e88e9300b15bcd106ecc88c8d864296f2da8

BUILT CONFIG
~~~~~~~~~~~~
default

RUN COMMAND
~~~~~~~~~~~
::

	./umem-cache

TEST COMMAND
~~~~~~~~~~~~
::

	go test -timeout=20h -run=BenchmarkPerformance			       \
	-bench=BenchmarkPerformance -benchtime=2000000x -cpu=8

TEST RESULT
~~~~~~~~~~~
::

	seed: 47	N: 1000
	goos: linux
	goarch: arm64
	pkg: github.com/imchuncai/umem-cache-client-Go
	BenchmarkPerformance-8   	 2000000	    327135 ns/op
	--- BENCH: BenchmarkPerformance-8
	bench_test.go:194: 
		=======================================================
		case:  685343    hot:  137068(20%)    hot_access: 80% 
		get: 2000000    hit: 1232285    hit_rate: 61.61% 
		hot: 1599997    hit: 1199622    hit_rate: 74.98% 
		cached:   86m(  397m  22%  -    100m  86%)
		hot:   59m(   79m  75%)  102687(137068  75%)      
		cold:   26m(  318m   8%)   45713(548275   8%)      
		=======================================================
	PASS
	ok  	github.com/imchuncai/umem-cache-client-Go	655.146s

	VmHWM:	  103948 kB

REDIS
---------
version: 7.4.1
commit: 74b289a0e12f9f65a6daeec6a66cadc76792f644

RUN COMMAND
~~~~~~~~~~~
::

	./src/redis-server --protected-mode no --appendonly no --save ""       \
	--maxmemory 100mb --maxclients 512 --maxmemory-policy allkeys-lru

TEST COMMAND
~~~~~~~~~~~~
::

	go test -timeout=20h -run=BenchmarkRedisPerformance		       \
	-bench=BenchmarkRedisPerformance -benchtime=2000000x -cpu=8

TEST RESULT
~~~~~~~~~~~
::

	seed: 47	N: 1000
	goos: linux
	goarch: arm64
	pkg: github.com/imchuncai/umem-cache-client-Go
	BenchmarkRedisPerformance-8   	 2000000	    267711 ns/op
	--- BENCH: BenchmarkRedisPerformance-8
	bench_test.go:194: 
		=======================================================
		case:  685343    hot:  137068(20%)    hot_access: 80% 
		get: 2000000    hit: 1185063    hit_rate: 59.25% 
		hot: 1599997    hit: 1152605    hit_rate: 72.04% 
		cached:   83m(  397m  21%  -    100m  83%)
		hot:   57m(   79m  72%)   98736(137068  72%)      
		cold:   26m(  318m   8%)   44879(548275   8%)      
		=======================================================
	PASS
	ok  	github.com/imchuncai/umem-cache-client-Go	536.345s

	VmHWM:	  111044 kB
