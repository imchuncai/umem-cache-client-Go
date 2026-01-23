// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2025, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package client

import (
	"container/list"
	"errors"
	"sync"
	"time"

	"github.com/imchuncai/umem-cache-client-Go/proto"
)

type authority struct {
	conn *proto.AuthorityConn

	mu       sync.Mutex
	requests *list.List
	waitN    int
	busy     bool
}

func newAuthority(conn *proto.AuthorityConn) *authority {
	auth := &authority{
		conn:     conn,
		requests: list.New(),
	}
	go func() {
		for {
			approval, err := conn.RecvApproval()
			if err != nil {
				auth.Close()
				return
			}

			auth.receivedApproval(approval)
		}
	}()
	return auth
}

func (auth *authority) __closed() bool {
	return auth.requests == nil
}

func (auth *authority) Closed() bool {
	auth.mu.Lock()
	defer auth.mu.Unlock()

	return auth.__closed()
}

func (auth *authority) Close() {
	auth.mu.Lock()
	defer auth.mu.Unlock()

	if auth.__closed() {
		return
	}

	auth.conn.Close()

	element := auth.requests.Front()
	for element != nil {
		ch := element.Value.(chan<- uint64)
		close(ch)

		element = element.Next()
	}
	auth.requests = nil
}

func (auth *authority) pushAuthority() (<-chan uint64, int, error) {
	auth.mu.Lock()
	defer auth.mu.Unlock()

	if auth.__closed() {
		return nil, 0, errors.New("authority is closed")
	}

	ch := make(chan uint64, 1)
	auth.requests.PushBack(chan<- uint64(ch))
	auth.waitN++

	if auth.busy {
		return ch, 0, nil
	}

	auth.busy = true
	n := auth.waitN
	auth.waitN = 0
	return ch, n, nil
}

func (auth *authority) RequestPermission(deadline time.Time) (<-chan uint64, error) {
	ch, n, err := auth.pushAuthority()
	if err != nil {
		return nil, err
	}

	err = auth.conn.RequestPermission(deadline, n)
	if err != nil {
		auth.Close()
		return nil, err
	}

	return ch, nil
}

func (auth *authority) popAuthority(approval proto.Approval) int {
	auth.mu.Lock()
	defer auth.mu.Unlock()

	if auth.__closed() {
		return 0
	}

	for range approval.Count {
		element := auth.requests.Front()
		auth.requests.Remove(element)
		ch := element.Value.(chan<- uint64)
		ch <- approval.Version
	}

	n := auth.waitN
	if n > 0 {
		auth.waitN = 0
	} else {
		auth.busy = false
	}
	return n
}

func (auth *authority) receivedApproval(approval proto.Approval) {
	n := auth.popAuthority(approval)
	if n > 0 {
		err := auth.conn.RequestPermission(time.Time{}, n)
		if err != nil {
			auth.Close()
		}
	}
}
