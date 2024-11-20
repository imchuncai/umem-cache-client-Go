// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2024, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package umem_cache

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"net"

	"time"
)

const KEY_SIZE_LIMIT = math.MaxUint8

type FallbackGetFunc func(key []byte) (val []byte, err error)

type command byte

const (
	_CMD_GET_OR_SET command = iota
	_CMD_DEL
)

type Conn struct {
	tcp *net.TCPConn
}

type errno byte

const (
	_E_NONE errno = iota
	_E_CONNECT_OUTDATED
	_E_CONNECT_TOO_MANY
	_E_CONNECT_KILL
	_E_GET_MISS
	_E_NR
)

var (
	// server side error
	ErrConnectOutdated = errors.New("client version too old")
	ErrConnectTooMany  = errors.New("too many connections")
	ErrConnectKill     = errors.New("connection is killed by server")

	// client side error
	ErrBadKeySize     = fmt.Errorf("key size out of limit: %d", KEY_SIZE_LIMIT)
	ErrBadFallbackGet = errors.New("bad fallback get")
	ErrBadErrno       = errors.New("bad errno")
)

var validErrors = [...]error{
	_E_CONNECT_OUTDATED: ErrConnectOutdated,
	_E_CONNECT_TOO_MANY: ErrConnectTooMany,
	_E_CONNECT_KILL:     ErrConnectKill,
	_E_NR:               nil,
}

func errnoError(e errno) error {
	if e >= _E_NR {
		return ErrBadErrno
	}
	return validErrors[e]
}

func (c *Conn) write(b []byte) error {
	_, err := c.tcp.Write(b)
	return err
}

func (c *Conn) writev(v [][]byte) error {
	buffer := net.Buffers(v)
	_, err := buffer.WriteTo(c.tcp)
	return err
}

func (c *Conn) readFull(b []byte) error {
	_, err := io.ReadFull(c.tcp, b)
	return err
}

func (c *Conn) connectThread(threadID uint32, version uint32) error {
	buf := make([]byte, 0, 4+4)
	buf = binary.BigEndian.AppendUint32(buf, threadID)
	buf = binary.BigEndian.AppendUint32(buf, version)
	err := c.write(buf)
	if err != nil {
		return fmt.Errorf("out failed: %w", err)
	}

	err = c.readFull(buf[:1])
	if err != nil {
		return fmt.Errorf("in failed: %w", err)
	}
	return errnoError(errno(buf[0]))
}

func Dial(address string, threadID uint32, version uint32, timeout time.Duration) (c *Conn, err error) {
	conn, err := net.DialTimeout("tcp6", address, timeout)
	if err != nil {
		return nil, fmt.Errorf("umem-cache: dial tcp failed: %w", err)
	}
	defer func() {
		if err != nil {
			conn.Close()
		}
	}()

	tcp := conn.(*net.TCPConn)
	err = tcp.SetLinger(0)
	if err != nil {
		return nil, fmt.Errorf("umem-cache: tcp set linger failed: %w", err)
	}

	c = &Conn{tcp}
	err = c.connectThread(threadID, version)
	if err != nil {
		return nil, fmt.Errorf("umem-cache: connect failed: %w", err)
	}
	return c, nil
}

func (c *Conn) Close() {
	c.tcp.Close()
}

func (c *Conn) setDeadline(timeout time.Duration) error {
	if timeout != 0 {
		return c.tcp.SetDeadline(time.Now().Add(timeout))
	}
	return nil
}

func (c *Conn) sendCMD(cmd command, key []byte) error {
	if len(key) > KEY_SIZE_LIMIT {
		return ErrBadKeySize
	}

	buf := make([]byte, 0, 1+1+len(key))
	buf = append(buf, byte(cmd), byte(len(key)))
	buf = append(buf, key...)
	return c.write(buf)
}

func (c *Conn) Del(key []byte, timeout time.Duration) error {
	err := c.setDeadline(timeout)
	if err != nil {
		return fmt.Errorf("umem-cache: del failed: set deadline failed: %w", err)
	}

	err = c.sendCMD(_CMD_DEL, key)
	if err != nil {
		return fmt.Errorf("umem-cache: del failed: out cmd failed: %w", err)
	}

	var buf [1]byte
	err = c.readFull(buf[:])
	if err != nil {
		return fmt.Errorf("umem-cache: del failed: in errno failed: %w", err)
	}
	return nil
}

func (c *Conn) setSend(val []byte) (err error) {
	if val == nil {
		err = c.write(make([]byte, 1+8))
	} else {
		buf := make([]byte, 1, 1+8)
		buf[0] = 1
		buf = binary.BigEndian.AppendUint64(buf, uint64(len(val)))
		err = c.writev([][]byte{buf, val})
	}
	if err != nil {
		return fmt.Errorf("out value failed: %w", err)
	}
	return nil
}

func (c *Conn) setRecv() error {
	var buf [1]byte
	err := c.readFull(buf[:])
	if err != nil {
		return fmt.Errorf("in errno failed: %w", err)
	}
	return nil
}

func (c *Conn) set(key []byte, fallbackGet FallbackGetFunc) (val []byte, err error) {
	val, err = fallbackGet((key))
	if err != nil {
		// too much error, can't handle
		c.setSend(nil)
		c.setRecv()
		return nil, fmt.Errorf("%w: %w", ErrBadFallbackGet, err)
	}

	err = c.setSend(val)
	if err != nil {
		return nil, err
	}

	err = c.setRecv()
	if err != nil {
		return nil, err
	}
	return val, err
}

func (c *Conn) get(key []byte) (val []byte, err error) {
	err = c.sendCMD(_CMD_GET_OR_SET, key)
	if err != nil {
		return nil, fmt.Errorf("out cmd failed: %w", err)
	}

	var buf [1 + 8]byte
	err = c.readFull(buf[:])
	if err != nil {
		return nil, fmt.Errorf("in value size failed: %w", err)
	}
	if errno(buf[0]) == _E_GET_MISS {
		return nil, nil
	}

	val_size := binary.BigEndian.Uint64(buf[1:])
	val = make([]byte, val_size)
	err = c.readFull(val)
	if err != nil {
		return nil, fmt.Errorf("in value failed: %w", err)
	}
	return val, nil
}

func (c *Conn) GetOrSet(key []byte, fallbackGet FallbackGetFunc, timeout time.Duration) (val []byte, err error) {
	err = c.setDeadline(timeout)
	if err != nil {
		return nil, fmt.Errorf("umem-cache: get or set failed: set deadline failed: %w", err)
	}

	val, err = c.get(key)
	if err != nil {
		return nil, fmt.Errorf("umem-cache: get or set failed: get failed: %w", err)
	}

	if val != nil {
		return val, nil
	}

	val, err = c.set(key, fallbackGet)
	if err != nil {
		return nil, fmt.Errorf("umem-cache: get or set failed: set failed: %w", err)
	}
	return val, nil
}
