.. SPDX-License-Identifier: BSD-3-Clause
.. Copyright (C) 2024, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

====================
UMEM-CACHE-CLIENT-GO
====================
UMEM-CACHE-CLIENT-GO is an UMEM-CACHE client written in Go. It implements mixed
TCP connections using full-duplex and half-duplex protocols. This is also the
test project of UMEM-CACHE.

FUNCTIONAL TEST
===============
::

	go test -failfast -p=1 -v

PERFORMANCE TEST
================
::

	go test -benchtime=100000x -run=BenchmarkPerformance \
	-bench=BenchmarkPerformance

BENCHMARK TEST
==============
::

	go test -benchtime=100000x -run=BenchmarkGetOrSet \
	-bench=BenchmarkGetOrSet

PERFORMANCE COMPARISON WITH MEMCACHED
=====================================
We have some key-value test cases which value size is random in the
range [0, 1m], we first get the value from the server, if the get missed, we
stored it. And 80% of the time the first 20% of the test cases are used.

CONCLUSION
----------
The test results showed that the performance data of UMEM-CACHE and MEMCACHE are
very close. But MEMCACHED used 25% more memory than expected.

The reason why MEMCACHED takes a lot more time than UMEM-CACHE is that we want a
complete test result, so when MEMCACHED has an (insufficient memory to store
objects) error, we sleep for 10 seconds.

TEST MACHINE
------------
Two 4GB version of Raspberry Pi 4 Model B connected in LAN with Gigabit network.
One used as a server and the other as a client.

MEMCACHED
---------
commit: 9723c0ea8ec1237b8364410ba982af8ea020a2b6

RUN COMMAND
~~~~~~~~~~~
::

	./memcached --memory-limit=2048

TEST COMMAND
~~~~~~~~~~~~
::

	go test -benchtime=100000x -run=BenchmarkMemcachedPerformance \
	-bench=BenchmarkMemcachedPerformance

TEST RESULT
~~~~~~~~~~~
::

	seed: 47	N: 1000
	goos: linux
	goarch: arm64
	pkg: github.com/imchuncai/umem-cache-client-Go
	BenchmarkMemcachedPerformance-4   	  100000	   5134670 ns/op
	--- BENCH: BenchmarkMemcachedPerformance-4
	client_test.go:510: 
		====================================================
		 case: 20480     hot:  4096(20%)    hot_access: 80% 
		  get: 84314    miss: 30514    hit_rate: 64% 
		  hot: 75771    miss: 23173    hit_rate: 69% 
		 cached: 1819m(10187m  18%  -   2048m  89%)
		   hot: 1397m( 2039m  69%)    2804( 4096  68%)      
		   cold:  422m( 8147m   5%)    1015(16384   6%)      
		====================================================
	PASS
	ok  	github.com/imchuncai/umem-cache-client-Go	513.549s

	VmPeak:	 2645052 kB

UMEM-CACHE
----------
commit: 5b9d123e3a6c3ca073a2303b69b7587c6057d6e7

BUILT CONFIG
~~~~~~~~~~~~
default

TEST COMMAND
~~~~~~~~~~~~
::

	go test -benchtime=100000x -run=BenchmarkPerformance \
	-bench=BenchmarkPerformance

TEST RESULT
~~~~~~~~~~~
::

	seed: 47	N: 1000
	goos: linux
	goarch: arm64
	pkg: github.com/imchuncai/umem-cache-client-Go
	BenchmarkPerformance-4   	  100000	   3892229 ns/op
	--- BENCH: BenchmarkPerformance-4
	client_test.go:510: 
		====================================================
		 case: 20480     hot:  4096(20%)    hot_access: 80% 
		  get: 84314    miss: 31285    hit_rate: 63% 
		  hot: 75771    miss: 24030    hit_rate: 68% 
		 cached: 1902m(10187m  19%  -   2048m  93%)
		    hot: 1369m( 2039m  67%)    2759( 4096  67%)      
		   cold:  532m( 8147m   7%)    1070(16384   7%)      
		====================================================
	PASS
	ok  	github.com/imchuncai/umem-cache-client-Go	389.307s
	
	VmPeak:	 1994564 kB

SPEED COMPARISON WITH MEMCACHED
===============================
The testing process is the same as performance test, but without statistics.

CONCLUSION
----------
The test results showed that UMEM-CACHE took more time than MEMCACHED, and I
must mention that some MEMCACHED set failed with (Insufficient memory to store
objects) errors, which we simply ignored to complete the test.

TEST MACHINE
------------
The test machine is the same as performance test.

MEMCACHED
---------
commit: 9723c0ea8ec1237b8364410ba982af8ea020a2b6

RUN COMMAND
~~~~~~~~~~~
::

	./memcached --memory-limit=2048

TEST COMMAND
~~~~~~~~~~~~
::

	go test -benchtime=100000x -run=BenchmarkMemcachedGetOrSet \
	-bench=BenchmarkMemcachedGetOrSet

TEST RESULT
~~~~~~~~~~~
::

	seed: 47	N: 1000
	goos: linux
	goarch: arm64
	pkg: github.com/imchuncai/umem-cache-client-Go
	BenchmarkMemcachedGetOrSet-4   	  100000	   3278541 ns/op
	PASS
	ok  	github.com/imchuncai/umem-cache-client-Go	327.928s
	
	VmPeak:	 2645052 kB

UMEM-CACHE
----------
commit: 5b9d123e3a6c3ca073a2303b69b7587c6057d6e7

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

	go test -benchtime=100000x -run=BenchmarkGetOrSet \
	-bench=BenchmarkGetOrSet

TEST RESULT
~~~~~~~~~~~
::

	seed: 47	N: 1000
	goos: linux
	goarch: arm64
	pkg: github.com/imchuncai/umem-cache-client-Go
	BenchmarkGetOrSet-4   	  100000	   3707137 ns/op
	PASS
	ok  	github.com/imchuncai/umem-cache-client-Go	370.794s
	
	VmPeak:	 1994564 kB
