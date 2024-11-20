.. SPDX-License-Identifier: BSD-3-Clause
.. Copyright (C) 2024, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

====================
UMEM-CACHE-CLIENT-GO
====================
UMEM-CACHE-CLIENT-GO is an UMEM-CACHE client written in Go. This is also the
test project of UMEM-CACHE.

Multilingual 多语言
==================

- `简体中文 <https://github.com/imchuncai/umem-cache-client-Go/tree/master/Documentation/translations/zh_CN/README.rst>`_

FUNCTIONAL TEST
===============
::

	go test -failfast -p=1 -v

PERFORMANCE TEST
================
::

	go test -timeout=20h -run=BenchmarkPerformance			       \
	-bench=BenchmarkPerformance -benchtime=2000000x -cpu=8

BENCHMARK TEST
==============
::

	go test -timeout=20h -run=BenchmarkGetOrSet			       \
	-bench=BenchmarkGetOrSet -benchtime=4000000x -cpu=8

CACHE COMPARISON
================
We compared the performance and speed of UMEM-CACHE, MEMCACHED and REDIS with
tests. The test process and results are at:
`cache_comparison <https://github.com/imchuncai/umem-cache-client-Go/tree/master/Documentation/cache_comparison>`_

Specifically, UMEM-CACHE showed a clear lead in the following two tests:

- `speed_1kb_random.rst <https://github.com/imchuncai/umem-cache-client-Go/tree/master/Documentation/cache_comparison/speed_1kb_random.rst>`_
- `performance_512kb.rst <https://github.com/imchuncai/umem-cache-client-Go/tree/master/Documentation/cache_comparison/performance_512kb.rst>`_

In most tests, UMEM-CACHE shows better cache hit rate and less memory usage.
In the test of caching small objects, UMEM-CACHE is about 20% faster than REDIS.
