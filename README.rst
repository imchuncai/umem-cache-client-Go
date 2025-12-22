.. SPDX-License-Identifier: BSD-3-Clause
.. Copyright (C) 2025, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

====================
UMEM-CACHE-CLIENT-GO
====================
UMEM-CACHE-CLIENT-GO is an UMEM-CACHE client written in Go. This is also the
test project of UMEM-CACHE.

Multilingual 多语言
==================

- `简体中文 <https://github.com/imchuncai/umem-cache-client-Go/tree/master/Documentation/translations/zh_CN/README.rst>`_

TEST SOURCE CODE
================
::

	make SRC=../umem-cache

TEST EXECUTABLE
===============
::

	make EXE=../umem-cache/umem-cache RAFT=0 TLS=0 DEBUG=0
