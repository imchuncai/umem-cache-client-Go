.. SPDX-License-Identifier: BSD-3-Clause
.. Copyright (C) 2024, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

=========================
缓存性能对比 (512KB固定大小)
=========================
我们对UMEM-CACHE,MEMCACHED和REDIS的缓存性能进行了比较。服务端内存配置大小为2GB，测试集的大
小为服务器配置内存大小的4倍，包含了不同大小的值，值的大小统一为512KB。我们首先向服务端 请求对
应的值，如果未请求到，我们将发送存储命令缓存该值，同时我们发出的80%的请求使用的是前20%的测试键
值对。我们会先发送N次请求预热缓存，然后再发送N次请求来收集统计数据。

结论
----
性能测试结果显示UMEM-CACHE和MEMCACHED的缓存性能表现十分接近，MEMCACHED的命中率高4%左右。
REDIS的命中率非常差，与UMEM-CACHE相比低17%左右，同时内存使用量只占配置内存的81%左右。

测试机器
-------
两台4GB版本的树莓派4 Model B用千兆网络在局域网连接，一台用作服务端，另一台用作客户端。两台机
器所安装的操作系统都为Fedora-Server-40-1.14.aarch64。

MEMCACHED
---------
commit: 5609673ed29db98a377749fab469fe80777de8fd

运行命令
~~~~~~~
::

	./memcached --conn-limit=512 --memory-limit=100

测试命令
~~~~~~~
::

	go test -timeout=20h -run=BenchmarkMemcachedPerformance		       \
	-bench=BenchmarkMemcachedPerformance -benchtime=50000x -cpu=8

测试结果
~~~~~~~
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

编译配置
~~~~~~~
::

	#define CONFIG_MEM_LIMIT	((uint64_t)2 << 30 >> PAGE_SHIFT)

运行命令
~~~~~~~
::

	./umem-cache

测试命令
~~~~~~~
::

	go test -timeout=20h -run=BenchmarkPerformance			       \
	-bench=BenchmarkPerformance -benchtime=50000x -cpu=8

测试结果
~~~~~~~
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

运行命令
~~~~~~~
::

	./src/redis-server --protected-mode no --appendonly no --save ""       \
	--maxmemory 2gb --maxclients 512 --maxmemory-policy allkeys-lru

测试命令
~~~~~~~
::

	go test -timeout=20h -run=BenchmarkRedisPerformance		       \
	-bench=BenchmarkRedisPerformance -benchtime=50000x -cpu=8

测试结果
~~~~~~~
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
