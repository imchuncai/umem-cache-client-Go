.. SPDX-License-Identifier: BSD-3-Clause
.. Copyright (C) 2024, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

====================
UMEM-CACHE-CLIENT-GO
====================
UMEM-CACHE-CLIENT-GO是UMEM-CACHE的Go客户端，它同时也是UMEM-CACHE的测试项目。

Multilingual 多语言
==================

- `简体中文 <https://github.com/imchuncai/umem-cache-client-Go/tree/master/Documentation/translations/zh_CN/README.rst>`_

功能测试
=======
::

	go test -failfast -p=1 -v

性能测试
=======
::

	go test -timeout=20h -run=BenchmarkPerformance			       \
	-bench=BenchmarkPerformance -benchtime=2000000x -cpu=8

基准测试
=======
::

	go test -timeout=20h -run=BenchmarkGetOrSet			       \
	-bench=BenchmarkGetOrSet -benchtime=4000000x -cpu=8

缓存对比
=======
我们对UMEM-CACHE, MEMCACHED和REDIS的缓存性能和速度进行了测试比较。测试流程以及结果参见：
`cache_comparison <https://github.com/imchuncai/umem-cache-client-Go/tree/master/Documentation/translations/zh_CN/cache_comparison>`_

特别在以下两项测试中，UMEM-CACHE表现出了明显的领先：

- `speed_1kb_random.rst <https://github.com/imchuncai/umem-cache-client-Go/tree/master/Documentation/translations/zh_CN/cache_comparison/speed_1kb_random.rst>`_
- `performance_512kb.rst <https://github.com/imchuncai/umem-cache-client-Go/tree/master/Documentation/translations/zh_CN/cache_comparison/performance_512kb.rst>`_

在绝大多数测试中，UMEM-CACHE表现出更好的缓存命中率以及更少的内存使用量。在缓存小对象的测试中，
UMEM-CACHE比REDIS快20%左右。
