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

func (auth *authority) pushAuthority() (<-chan uint64, error) {
	auth.mu.Lock()
	defer auth.mu.Unlock()

	if auth.__closed() {
		return nil, errors.New("authority is closed")
	}

	ch := make(chan uint64, 1)
	auth.requests.PushBack(chan<- uint64(ch))
	return ch, nil
}

func (auth *authority) RequestPermission(deadline time.Time) (<-chan uint64, error) {
	ch, err := auth.pushAuthority()
	if err != nil {
		return nil, err
	}

	err = auth.conn.RequestPermission(deadline)
	if err != nil {
		auth.Close()
		return nil, err
	}

	return ch, nil
}

func (auth *authority) receivedApproval(approval proto.Approval) {
	auth.mu.Lock()
	defer auth.mu.Unlock()

	if auth.__closed() {
		return
	}

	for range approval.Count {
		element := auth.requests.Front()
		auth.requests.Remove(element)
		ch := element.Value.(chan<- uint64)
		ch <- approval.Version
	}
}
