.. SPDX-License-Identifier: BSD-3-Clause
.. Copyright (C) 2024, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

========================
缓存性能对比 (512B固定大小)
========================
我们对UMEM-CACHE,MEMCACHED和REDIS的缓存性能进行了比较。服务端内存配置大小为100MB，测试集的
大小为服务器配置内存大小的4倍，包含了不同大小的值，值的大小统一为512字节。我们首先向服务端 请求
对应的值，如果未请求到，我们将发送存储命令缓存该值，同时我们发出的80%的请求使用的是前20%的测试
键值对。我们会先发送N次请求预热缓存，然后再发送N次请求来收集统计数据。

结论
----
性能测试结果显示UMEM-CACHE，MEMCACHED和REDIS三者的缓存性能表现十分接近，UMEM-CACHE最好，
MEMCACHED次之，REDIS最差。UMEM-CACHE比REDIS命中率高1%左右，同时内存使用少5%左右。

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
	-bench=BenchmarkMemcachedPerformance -benchtime=2000000x -cpu=8

测试结果
~~~~~~~
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

编译配置
~~~~~~~
默认配置

运行命令
~~~~~~~
::

	./umem-cache

测试命令
~~~~~~~
::

	go test -timeout=20h -run=BenchmarkPerformance			       \
	-bench=BenchmarkPerformance -benchtime=2000000x -cpu=8

测试结果
~~~~~~~
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

运行命令
~~~~~~~
::

	./src/redis-server --protected-mode no --appendonly no --save ""       \
	--maxmemory 100mb --maxclients 512 --maxmemory-policy allkeys-lru

测试命令
~~~~~~~
::

	go test -timeout=20h -run=BenchmarkRedisPerformance		       \
	-bench=BenchmarkRedisPerformance -benchtime=2000000x -cpu=8

测试结果
~~~~~~~
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
