// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2025, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package proto

import (
	"bufio"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"time"
)

type FallbackGetFunc func(key []byte) (val []byte, err error)

const _KEY_SIZE_MAX = math.MaxUint8

const (
	_CMD_GET_OR_SET byte = iota
	_CMD_DEL
)

type CacheConn struct {
	tlsBuffer *bufio.Writer
	conn      *Conn
}

func DialCache(deadline time.Time, address string, threadID uint32, config *tls.Config) (*CacheConn, error) {
	conn, err := Dial(deadline, address, config)
	if err != nil {
		return nil, err
	}

	err = conn.SetDeadline(deadline)
	if err != nil {
		conn.Close()
		return nil, err
	}

	err = conn.connect(threadID)
	if err != nil {
		conn.Close()
		return nil, err
	}

	var buff *bufio.Writer
	if _, ok := conn.v.(*tls.Conn); ok {
		buff = bufio.NewWriterSize(conn.v, 1<<14)
	}

	return &CacheConn{buff, conn}, nil
}

func (c *CacheConn) communicateV(req net.Buffers, res []byte) error {
	if _, ok := c.conn.v.(*net.TCPConn); ok {
		return c.conn.communicateV(req, res)
	}

	for _, buff := range req {
		_, err := c.tlsBuffer.Write(buff)
		if err != nil {
			return fmt.Errorf("tls buffer write failed: %w", err)
		}
	}
	err := c.tlsBuffer.Flush()
	if err != nil {
		return fmt.Errorf("tls buffer flush failed: %w", err)
	}
	return c.conn.read(res)
}

func (c *CacheConn) set(val []byte) error {
	size := make([]byte, 8)
	binary.BigEndian.PutUint64(size, uint64(len(val)))
	res := make([]byte, 1)
	return c.communicateV(net.Buffers{size, val}, res)
}

func (c *CacheConn) get(key []byte) (val []byte, err error) {
	res := make([]byte, 8+1)
	err = c.communicateV(net.Buffers{{_CMD_GET_OR_SET, byte(len(key))}, key}, res)
	if err != nil {
		return nil, err
	}
	if res[8] == 1 {
		return nil, nil
	}

	size := binary.BigEndian.Uint64(res)
	val = make([]byte, size)
	err = c.conn.read(val)
	if err != nil {
		return nil, fmt.Errorf("read value failed: %w", err)
	}
	return val, nil
}

func (c *CacheConn) GetOrSet(deadline time.Time, key []byte, get FallbackGetFunc) ([]byte, error) {
	if len(key) > _KEY_SIZE_MAX {
		return nil, ErrBadKeySize
	}

	err := c.conn.SetDeadline(deadline)
	if err != nil {
		return nil, err
	}

	val, err := c.get(key)
	if err != nil {
		return nil, fmt.Errorf("get failed: %w", err)
	}

	if val != nil {
		return val, nil
	}

	val, err = get(key)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFallbackGet, err)
	}

	return val, c.set(val)
}

func (c *CacheConn) Del(deadline time.Time, key []byte) error {
	if len(key) > _KEY_SIZE_MAX {
		return ErrBadKeySize
	}

	err := c.conn.SetDeadline(deadline)
	if err != nil {
		return err
	}

	res := make([]byte, 1)
	return c.communicateV(net.Buffers{{_CMD_DEL, byte(len(key))}, key}, res)
}

func (c *CacheConn) Close() {
	c.conn.Close()
}
