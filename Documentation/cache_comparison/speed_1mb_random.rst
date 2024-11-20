.. SPDX-License-Identifier: BSD-3-Clause
.. Copyright (C) 2024, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

=========================================
CACHE SPEED COMPARISON (1MB LIMIT RANDOM)
=========================================
We compared the cache performance of UMEM-CACHE, MEMCACHED and REDIS. Server
side memory limit is set to 2GB, and test case size is set to 4 times of that,
and value size is random in the range[0, 1m]. we first get the value from the
server, if the get missed, we stored it. And 80% of the time the first 20% of
the test cases are used.

CONCLUSION
----------
Some MEMCACHED set failed with (Insufficient memory to store objects) error,
which we simply ignored to complete the test. So MEMCACHED is not included in
the comparison, and the test results are for reference only.

It is also worth mentioning that the cost of cache miss is not reflected in the
test results. In the test case, the time we simulated to access the backup
database is basically negligible. UMEM-CACHE has a better hit rate so the speed
performance will be even better.

The test results showed that UMEM-CACHE is 8% slower than REDIS. This is due to
the higher hit rate. Because the cache value is relatively large, IO occupies
most of the CPU time. In addition, because the hit rate is higher than 50%, so
the performance bottleneck is at the server side out IO. A higher hit rate
brings more server side out data, which slows down the speed.

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

	go test -timeout=20h -run=BenchmarkMemcachedGetOrSet		       \
	-bench=BenchmarkMemcachedGetOrSet -benchtime=100000x -cpu=8

TEST RESULT
~~~~~~~~~~~
::

	seed: 47	N: 1000
	goos: linux
	goarch: arm64
	pkg: github.com/imchuncai/umem-cache-client-Go
	BenchmarkMemcachedGetOrSet-8   	  100000	   3142730 ns/op
	PASS
	ok  	github.com/imchuncai/umem-cache-client-Go	314.373s

	VmHWM:	 2111052 kB

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

	go test -timeout=20h -run=BenchmarkGetOrSet			       \
	-bench=BenchmarkGetOrSet -benchtime=100000x -cpu=8

TEST RESULT
~~~~~~~~~~~
::

	seed: 47	N: 1000
	goos: linux
	goarch: arm64
	pkg: github.com/imchuncai/umem-cache-client-Go
	BenchmarkGetOrSet-8   	  100000	   3295444 ns/op
	PASS
	ok  	github.com/imchuncai/umem-cache-client-Go	329.641s

	VmHWM:	 2098388 kB

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

	go test -timeout=20h -run=^BenchmarkRedisGetOrSet$		       \
	-bench=^BenchmarkRedisGetOrSet$ -benchtime=100000x -cpu=8

TEST RESULT
~~~~~~~~~~~
::

	seed: 47	N: 1000
	goos: linux
	goarch: arm64
	pkg: github.com/imchuncai/umem-cache-client-Go
	BenchmarkRedisGetOrSet-8   	  100000	   3065866 ns/op
	PASS
	ok  	github.com/imchuncai/umem-cache-client-Go	306.686s

	VmHWM:	 2153256 kB
