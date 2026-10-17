// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2025-2026, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package proto

import (
	"bufio"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
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

const _DEFAULT_BUFFER_SIZE = 16 << 10

type _CacheCMD byte

const (
	_CMD_GET_OR_SET _CacheCMD = iota
	_CMD_DEL
	_CMD_STATS
)

type CacheConn struct {
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
}

func connect(deadline time.Time, conn net.Conn, threadID uint32) error {
	req := make([]byte, 4+4)
	req[0] = byte(_CMD_CONNECT)
	binary.LittleEndian.PutUint32(req[4:], threadID)
	return communicate(deadline, conn, req, []byte{0})
}

func DialCache(deadline time.Time, address string, threadID uint32, config *tls.Config) (*CacheConn, error) {
	conn, err := dial(deadline, address, config)
	if err != nil {
		return nil, fmt.Errorf("dial failed: %w", err)
	}

	err = connect(deadline, conn, threadID)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("request connect failed: %w", err)
	}

	var writer *bufio.Writer
	if config != nil {
		writer = bufio.NewWriterSize(conn, _DEFAULT_BUFFER_SIZE)
	}
	return &CacheConn{conn, bufio.NewReaderSize(conn, _DEFAULT_BUFFER_SIZE), writer}, nil
}

func (c *CacheConn) SetDeadline(deadline time.Time) error {
	return c.conn.SetDeadline(deadline)
}

func (c *CacheConn) writev(buffers net.Buffers) error {
	if c.writer == nil {
		_, err := buffers.WriteTo(c.conn)
		return err
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

func (c *CacheConn) write(b []byte) error {
	_, err := c.conn.Write(b)
	return err
}

func (c *CacheConn) read(b []byte) error {
	_, err := c.conn.Read(b)
	return err
}

func (c *CacheConn) bufferRead(b []byte) error {
	_, err := io.ReadFull(c.reader, b)
	return err
}

func (c *CacheConn) bufferDiscard(n int) error {
	_, err := c.reader.Discard(n)
	return err
}

func (c *CacheConn) writeCMD(cmd _CacheCMD, key []byte) error {
	if len(key) > math.MaxUint8 {
		return ErrBadKeySize
	}

	return c.writev(net.Buffers{{byte(cmd), byte(len(key))}, key})
}

func (c *CacheConn) get(key []byte) ([]byte, error) {
	err := c.writeCMD(_CMD_GET_OR_SET, key)
	if err != nil {
		return nil, fmt.Errorf("write cmd failed: %w", err)
	}

	res := make([]byte, 8+1)
	err = c.bufferRead(res)
	if err != nil {
		return nil, fmt.Errorf("read cmd response failed: %w", err)
	}

	if res[8] == 1 {
		return nil, nil
	}

	size := binary.LittleEndian.Uint64(res)
	val := make([]byte, size)
	err = c.bufferRead(val)
	if err != nil {
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

	err = c.set(val)
	if err != nil {
		return val, fmt.Errorf("set failed: %w", err)
	}
	return val, nil
}

func (c *CacheConn) Del(key []byte) error {
	err := c.writeCMD(_CMD_DEL, key)
	if err != nil {
		return fmt.Errorf("write cmd failed: %w", err)
	}

	err = c.read([]byte{0})
	if err != nil {
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
	err := c.write([]byte{byte(_CMD_STATS), 0})
	if err != nil {
		return Stats{Error: fmt.Errorf("write cmd failed: %w", err)}
	}

	const sizeV1 = 76
	res := make([]byte, 4+sizeV1)
	err = c.bufferRead(res)
	if err != nil {
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
		err := c.bufferDiscard(int(size) - sizeV1)
		if err != nil {
			return Stats{Error: fmt.Errorf("discard failed: %w", err)}
		}
	}
	return stats
}

func (c *CacheConn) Close() {
	c.conn.Close()
}
