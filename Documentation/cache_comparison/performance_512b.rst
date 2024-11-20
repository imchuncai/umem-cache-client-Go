.. SPDX-License-Identifier: BSD-3-Clause
.. Copyright (C) 2024, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

==============================================
CACHE PERFORMANCE COMPARISON (512B FIXED SIZE)
==============================================
We compared the cache performance of UMEM-CACHE, MEMCACHED and REDIS. Server
side memory limit is set to 100MB, and test case size is set to 4 times of that,
and value size is all 512 bytes. we first get the value from the server, if the
get missed, we stored it. And 80% of the time the first 20% of the test cases
are used. We first make N requests to warm up the cache, and then make N more
requests to collect statistics.

CONCLUSION
----------
The test results showed that the performance of UMEM-CACHE, MEMCACHE and REDIS
are very close, UMEM-CACHE is the best, MEMCACHED is the second, and REDIS is
the worst. UMEM-CACHE has a hit rate about 1% higher than REDIS, while using
about 5% less memory.

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
	BenchmarkMemcachedPerformance-8   	 2000000	    290332 ns/op
	--- BENCH: BenchmarkMemcachedPerformance-8
	bench_test.go:194: 
		=======================================================
		case:  685343    hot:  137068(20%)    hot_access: 80% 
		get: 2000000    hit: 1105623    hit_rate: 55.28% 
		hot: 1599997    hit: 1085033    hit_rate: 67.81% 
		cached:   81m(  463m  18%  -    100m  82%)
		hot:   62m(   92m  67%)   91423(137068  67%)      
		cold:   19m(  370m   5%)   28662(548275   5%)      
		=======================================================
	PASS
	ok  	github.com/imchuncai/umem-cache-client-Go	581.697s

	VmHWM:	  106600 kB

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
	BenchmarkPerformance-8   	 2000000	    335535 ns/op
	--- BENCH: BenchmarkPerformance-8
	bench_test.go:194: 
		=======================================================
		case:  685343    hot:  137068(20%)    hot_access: 80% 
		get: 2000000    hit: 1114452    hit_rate: 55.72% 
		hot: 1599997    hit: 1087344    hit_rate: 67.96% 
		cached:   88m(  463m  19%  -    100m  88%)
		hot:   62m(   92m  68%)   93070(137068  68%)      
		cold:   25m(  370m   7%)   37835(548275   7%)      
		=======================================================
	PASS
	ok  	github.com/imchuncai/umem-cache-client-Go	672.136s

	VmHWM:	  104100 kB

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
	BenchmarkRedisPerformance-8   	 2000000	    271160 ns/op
	--- BENCH: BenchmarkRedisPerformance-8
	bench_test.go:194: 
		=======================================================
		case:  685343    hot:  137068(20%)    hot_access: 80% 
		get: 2000000    hit: 1099028    hit_rate: 54.95% 
		hot: 1599997    hit: 1071106    hit_rate: 66.94% 
		cached:   88m(  463m  19%  -    100m  88%)
		hot:   61m(   92m  67%)   91617(137068  67%)      
		cold:   26m(  370m   7%)   39072(548275   7%)      
		=======================================================
	PASS
	ok  	github.com/imchuncai/umem-cache-client-Go	543.264s

	VmHWM:	  109316 kB
