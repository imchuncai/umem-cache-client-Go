.. SPDX-License-Identifier: BSD-3-Clause
.. Copyright (C) 2025, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

====================
UMEM-CACHE-CLIENT-GO
====================
UMEM-CACHE-CLIENT-GO是UMEM-CACHE的Go客户端，它同时也是UMEM-CACHE的测试项目。

Multilingual 多语言
==================

- `简体中文 <https://github.com/imchuncai/umem-cache-client-Go/tree/master/Documentation/translations/zh_CN/README.rst>`_

测试源码
=======
::

	make SRC=../umem-cache

测试可执行文件
============
::

	make EXE=../umem-cache/umem-cache RAFT=0 TLS=0 DEBUG=0
