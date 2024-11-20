.. SPDX-License-Identifier: BSD-3-Clause
.. Copyright (C) 2024, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

==========================
缓存速度对比 (1MB限制随机大小)
==========================
我们对UMEM-CACHE,MEMCACHED和REDIS的缓存性能进行了比较。服务端内存配置大小为2GB，测试集的大
小为服务器配置内存大小的4倍，包含了不同大小的值，值的大小在[0, 1m]区间随机。我们首先向服务端
请求对应的值，如果未请求到，我们将发送存储命令缓存该值，同时我们发出的80%的请求使用的是前20%的
测试键值对。

结论
----
MEMCACHED在高负载下会报告内存不足错误，为完成测试，我们忽略了该错误，所以MEMCACHED不参与比
较，测试结果仅供参考。

另外值得一提的是，缓存缺失的代价未在测试结果中体现，在测试用例中，我们模拟访问后备数据库的时间
基本可以忽略不计。UMEM-CACHE拥有更好的命中率所以速度表现还会更好。

速度测试结果显示UMEM-CACHE比REDIS慢了8%左右。这是受高命中率拖累，因为缓存的值比较大，IO占用
了绝大部分的CPU时间，而且由于命中率高于50%，所以此时的性能瓶颈在服务端出口IO上，更高的命中率带
来了更多的服务端出口数据，从而拖累了速度。

测试机器
------------
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

	go test -timeout=20h -run=BenchmarkMemcachedGetOrSet		       \
	-bench=BenchmarkMemcachedGetOrSet -benchtime=100000x -cpu=8

测试结果
~~~~~~~
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

	go test -timeout=20h -run=BenchmarkGetOrSet			       \
	-bench=BenchmarkGetOrSet -benchtime=100000x -cpu=8

测试结果
~~~~~~~
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

运行命令
~~~~~~~
::

	./src/redis-server --protected-mode no --appendonly no --save ""       \
	--maxmemory 2gb --maxclients 512 --maxmemory-policy allkeys-lru

测试命令
~~~~~~~
::

	go test -timeout=20h -run=^BenchmarkRedisGetOrSet$		       \
	-bench=^BenchmarkRedisGetOrSet$ -benchtime=100000x -cpu=8

测试结果
~~~~~~~
::

	seed: 47	N: 1000
	goos: linux
	goarch: arm64
	pkg: github.com/imchuncai/umem-cache-client-Go
	BenchmarkRedisGetOrSet-8   	  100000	   3065866 ns/op
	PASS
	ok  	github.com/imchuncai/umem-cache-client-Go	306.686s

	VmHWM:	 2153256 kB
