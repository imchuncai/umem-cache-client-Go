.. SPDX-License-Identifier: BSD-3-Clause
.. Copyright (C) 2024, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

==========================
缓存性能对比 (1MB限制随机大小)
==========================
我们对UMEM-CACHE,MEMCACHED和REDIS的缓存性能进行了比较。服务端内存配置大小为2GB，测试集的大
小为服务器配置内存大小的4倍，包含了不同大小的值，值的大小在[0, 1m]区间随机。我们首先向服务端
请求对应的值，如果未请求到，我们将发送存储命令缓存该值，同时我们发出的80%的请求使用的是前20%的
测试键值对。我们会先发送N次请求预热缓存，然后再发送N次请求来收集统计数据。

结论
----
性能测试结果显示UMEM-CACHE，MEMCACHED和REDIS三者的缓存性能表现十分接近，UMEM-CACHE最好，
MEMCACHED次之，REDIS最差。UMEM-CACHE比REDIS命中率高8%左右，同时内存使用少2%左右。

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

	./memcached --conn-limit=512 --memory-limit=2048 --max-item-size=2m

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
