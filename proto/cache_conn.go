// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2025-2026, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package proto

import (
	"bufio"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net"
	"time"
)

type FallbackGetFunc func(key []byte) (val []byte, err error)

var (
	ErrClientSide  = errors.New("client side error")
	ErrBadKeySize  = fmt.Errorf("%w: key size out of limit", ErrClientSide)
	ErrFallbackGet = fmt.Errorf("%w: fallback get failed", ErrClientSide)
)

type _CMD byte

const (
	_CMD_GET_OR_SET _CMD = iota
	_CMD_DEL
	_CMD_STATS
)

type CacheConn struct {
	writer *bufio.Writer
	conn   *Conn
}

func DialCache(deadline time.Time, address string, threadID uint32, config *tls.Config) (*CacheConn, error) {
	conn, err := Dial(deadline, address, config)
	if err != nil {
		return nil, fmt.Errorf("dial failed: %w", err)
	}

	err = conn.SetDeadline(deadline)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("set deadline failed: %w", err)
	}

	err = conn.connect(threadID)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("connect thread failed: %w", err)
	}

	if config == nil {
		return &CacheConn{nil, conn}, nil
	}

	return &CacheConn{bufio.NewWriterSize(conn.v, DEFAULT_BUFFER_SIZE), conn}, nil
}

func (c *CacheConn) SetDeadline(deadline time.Time) error {
	return c.conn.SetDeadline(deadline)
}

func (c *CacheConn) read(buff []byte) error {
	return c.conn.read(buff)
}

func (c *CacheConn) writev(buffers net.Buffers) error {
	if c.writer == nil {
		return c.conn.writev(buffers)
	}

	for _, buff := range buffers {
		_, err := c.writer.Write(buff)
		if err != nil {
			return fmt.Errorf("buffer write failed: %w", err)
		}
	}
	err := c.writer.Flush()
	if err != nil {
		return fmt.Errorf("buffer flush failed: %w", err)
	}
	return nil
}

func (c *CacheConn) discard(n int) error {
	return c.conn.discard(n)
}

func (c *CacheConn) writeCMD(cmd _CMD, key []byte) error {
	if len(key) > math.MaxUint8 {
		return ErrBadKeySize
	}

	return c.writev(net.Buffers{{byte(cmd), byte(len(key))}, key})
}

func (c *CacheConn) get(key []byte) ([]byte, error) {
	if err := c.writeCMD(_CMD_GET_OR_SET, key); err != nil {
		return nil, fmt.Errorf("write cmd failed: %w", err)
	}

	res := make([]byte, 8+1)
	if err := c.read(res); err != nil {
		return nil, fmt.Errorf("read cmd response failed: %w", err)
	}

	if res[8] == 1 {
		return nil, nil
	}

	size := binary.LittleEndian.Uint64(res)
	val := make([]byte, size)
	if err := c.read(val); err != nil {
		return nil, fmt.Errorf("read value failed: %w", err)
	}
	return val, nil
}

func (c *CacheConn) set(val []byte) error {
	size := make([]byte, 8)
	binary.LittleEndian.PutUint64(size, uint64(len(val)))
	return c.writev(net.Buffers{size, val})
}

func (c *CacheConn) GetOrSet(key []byte, get FallbackGetFunc) ([]byte, error) {
	val, err := c.get(key)
	if err != nil {
		return nil, fmt.Errorf("get failed: %w", err)
	}

	if val != nil {
		return val, nil
	}

	if get == nil {
		return nil, fmt.Errorf("%w: nil", ErrFallbackGet)
	}

	val, err = get(key)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFallbackGet, err)
	}

	if err := c.set(val); err != nil {
		return val, fmt.Errorf("set failed: %w", err)
	}
	return val, nil
}

func (c *CacheConn) Del(key []byte) error {
	err := c.writeCMD(_CMD_DEL, key)
	if err != nil {
		return fmt.Errorf("write cmd failed: %w", err)
	}

	if err := c.read([]byte{0}); err != nil {
		return fmt.Errorf("read failed: %w", err)
	}
	return nil
}

type Stats struct {
	Error error

	ConfigThreadNR   uint32
	ConfigTCPTimeout uint32
	ConfigMaxConn    uint32
	ConfigMemLimit   uint64
	Version          uint64

	StatNanoseconds uint64
	Connects        uint64
	Disconnects     uint64
	Hits            uint64
	Misses          uint64
	Dels            uint64
}

func (c *CacheConn) Stats() Stats {
	err := c.writeCMD(_CMD_STATS, nil)
	if err != nil {
		return Stats{Error: fmt.Errorf("write cmd failed: %w", err)}
	}

	const sizeV1 = 76
	res := make([]byte, 4+sizeV1)
	if err := c.read(res); err != nil {
		return Stats{Error: fmt.Errorf("read failed: %w", err)}
	}
	size := binary.LittleEndian.Uint32(res)

	var stats Stats
	stats.ConfigThreadNR = binary.LittleEndian.Uint32(res[4:])
	stats.ConfigTCPTimeout = binary.LittleEndian.Uint32(res[8:])
	stats.ConfigMaxConn = binary.LittleEndian.Uint32(res[12:])
	stats.ConfigMemLimit = binary.LittleEndian.Uint64(res[16:])
	stats.Version = binary.LittleEndian.Uint64(res[24:])

	stats.StatNanoseconds = binary.LittleEndian.Uint64(res[32:])
	stats.Connects = binary.LittleEndian.Uint64(res[40:])
	stats.Disconnects = binary.LittleEndian.Uint64(res[48:])
	stats.Hits = binary.LittleEndian.Uint64(res[56:])
	stats.Misses = binary.LittleEndian.Uint64(res[64:])
	stats.Dels = binary.LittleEndian.Uint64(res[72:])

	if size > sizeV1 {
		err := c.discard(int(size) - sizeV1)
		if err != nil {
			return Stats{Error: fmt.Errorf("discard failed: %w", err)}
		}
	}
	return stats
}

func (c *CacheConn) Close() {
	c.conn.Close()
}
