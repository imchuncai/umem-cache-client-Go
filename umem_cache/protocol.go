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
)

const KEY_SIZE_LIMIT = math.MaxUint8

type FallbackGetFunc func(key string) (val []byte, err error)

type command byte

const (
	_CMD_GET command = iota
	_CMD_SET
	_CMD_DEL
)

type getFlag byte
type setFlag byte
type delFlag byte

const (
	GET_FLAG_NON_BLOCK   = getFlag(1 << 0)
	GET_FLAG_SET_ON_MISS = getFlag(1 << 1)

	SET_FLAG_NON_BLOCK = setFlag(1 << 0)

	DEL_FLAG_NON_BLOCK = delFlag(1 << 0)
)

type Conn struct {
	tcp *net.TCPConn
}

type GetResp struct {
	WillBlock bool
	Val       []byte
	Err       error
}

type SetResp struct {
	WillBlock bool
	Err       error
}

type DelResp struct {
	WillBlock bool
	Err       error
}

type errno byte

const (
	_E_NONE errno = iota
	_E_OUTDATED
	_E_TOO_MANY
	_E_WILL_BLOCK
	_E_MISS
	_E_NOMEM
	_E_KILL
	_E_NR
)

var (
	errOutdated = errors.New("client version too old")
	errTooMany  = errors.New("too many connections")
	errNoMem    = errors.New("allocate memory from system failed ")
	errKill     = errors.New("connection is force killed by server")

	// server side error
	ConnectErrOutdated = errOutdated
	ConnectErrTooMany  = errTooMany
	ConnectErrKill     = errKill
	GetOrSetErrNoMem   = errNoMem
	SetErrNoMem        = errNoMem

	// client side error
	ErrBadKeySize     = fmt.Errorf("key size out of limit: %d", KEY_SIZE_LIMIT)
	ErrBadFallbackGet = errors.New("bad fallback get")
	ErrBadErrno       = errors.New("bad errno")
)

var validErrors = [...]error{
	_E_OUTDATED: errOutdated,
	_E_NOMEM:    errNoMem,
	_E_TOO_MANY: errTooMany,
	_E_KILL:     errKill,
	_E_NR:       nil,
}

func errnoError(e errno) error {
	if e >= _E_NR {
		return ErrBadErrno
	}
	return validErrors[e]
}

const writeOnceSize = 4096

func (c *Conn) write(b []byte) error {
	_, err := c.tcp.Write(b)
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

func Dial(address string, threadID uint32, version uint32) (*Conn, error) {
	addr, err := net.ResolveTCPAddr("tcp6", address)
	if err != nil {
		return nil, fmt.Errorf("umem-cache: connect resolve tcp address failed: %w", err)
	}

	tcp, err := net.DialTCP("tcp6", nil, addr)
	if err != nil {
		return nil, fmt.Errorf("umem-cache: connect dial tcp failed: %w", err)
	}

	tcp.SetNoDelay(true)
	tcp.SetLinger(0)

	c := Conn{tcp}
	err = c.connectThread(threadID, version)
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("umem-cache: connect failed: %w", err)
	}

	return &c, nil
}

func (c *Conn) Close() {
	c.tcp.Close()
}

func cmdSize(key string) int {
	return 1 + 1 + 1 + 8 + len(key)
}

func appendCMD(buf []byte, cmd command, flag byte, key string, val []byte) []byte {
	buf = append(buf, byte(cmd))
	buf = append(buf, flag)
	buf = append(buf, byte(len(key)))
	buf = binary.BigEndian.AppendUint64(buf, uint64(len(val)))
	return append(buf, key...)
}

func (c *Conn) sendCMD(cmd command, flag byte, key string, val []byte) error {
	buf := make([]byte, 0, cmdSize(key))
	buf = appendCMD(buf, cmd, flag, key, val)
	return c.write(buf)
}

func checkKey(key string) error {
	if len(key) > KEY_SIZE_LIMIT {
		return ErrBadKeySize
	}
	return nil
}

func (c *Conn) GetSend(key string, flag getFlag) error {
	err := checkKey(key)
	if err != nil {
		return fmt.Errorf("umem-cache: get failed: %w", err)
	}

	err = c.sendCMD(_CMD_GET, byte(flag), key, nil)
	if err != nil {
		return fmt.Errorf("umem-cache: get out cmd failed: %w", err)
	}
	return nil
}

func (c *Conn) GetRecv() GetResp {
	var buf [1 + 8]byte
	err := c.readFull(buf[:])
	if err != nil {
		return GetResp{false, nil, fmt.Errorf("umem-cache: get in value size failed: %w", err)}
	}

	switch errno(buf[0]) {
	case _E_WILL_BLOCK:
		return GetResp{true, nil, nil}
	case _E_MISS:
		return GetResp{false, nil, nil}
	}

	val_size := binary.BigEndian.Uint64(buf[1:])
	val := make([]byte, val_size)
	err = c.readFull(val)
	if err != nil {
		return GetResp{false, nil, fmt.Errorf("umem-cache: get in value failed: %w", err)}
	}

	return GetResp{false, val, nil}
}

func (c *Conn) Get(key string, flag getFlag) GetResp {
	err := c.GetSend(key, flag)
	if err != nil {
		return GetResp{false, nil, err}
	}
	return c.GetRecv()
}

func (c *Conn) SetSend(key string, val []byte, flag setFlag) error {
	err := checkKey(key)
	if err != nil {
		return fmt.Errorf("umem-cache: set failed: %w", err)
	}

	size := cmdSize(key) + len(val)
	if size <= writeOnceSize {
		buf := make([]byte, 0, size)
		buf = appendCMD(buf, _CMD_SET, byte(flag), key, val)
		buf = append(buf, val...)
		err := c.write(buf)
		if err != nil {
			return fmt.Errorf("umem-cache: set out once failed: %w", err)
		}
		return nil
	}

	err = c.sendCMD(_CMD_SET, byte(flag), key, val)
	if err != nil {
		return fmt.Errorf("umem-cache: set out cmd failed: %w", err)
	}

	err = c.write(val)
	if err != nil {
		return fmt.Errorf("umem-cache: set out value failed: %w", err)
	}
	return nil
}

func (c *Conn) SetRecv() SetResp {
	var buf [1]byte
	err := c.readFull(buf[:])
	if err != nil {
		return SetResp{false, fmt.Errorf("umem-cache: set in errno failed: %w", err)}
	}
	if errno(buf[0]) == _E_WILL_BLOCK {
		return SetResp{true, nil}
	}
	err = errnoError(errno(buf[0]))
	if err != nil {
		return SetResp{false, fmt.Errorf("umem-cache: set failed: %w", err)}
	}
	return SetResp{false, nil}
}

func (c *Conn) DelSend(key string, flag delFlag) error {
	err := checkKey(key)
	if err != nil {
		return fmt.Errorf("umem-cache: del failed: %w", err)
	}

	err = c.sendCMD(_CMD_DEL, byte(flag), key, nil)
	if err != nil {
		return fmt.Errorf("umem-cache: del out cmd failed: %w", err)
	}
	return nil
}

func (c *Conn) DelRecv() DelResp {
	var buf [1]byte
	err := c.readFull(buf[:])
	if err != nil {
		return DelResp{false, fmt.Errorf("umem-cache: del in errno failed: %w", err)}
	}

	if errno(buf[0]) == _E_WILL_BLOCK {
		return DelResp{true, nil}
	}

	err = errnoError(errno(buf[0]))
	if err != nil {
		return DelResp{false, fmt.Errorf("umem-cache: del failed: %w", err)}
	}
	return DelResp{false, nil}
}

func (c *Conn) Del(key string, flag delFlag) DelResp {
	err := c.DelSend(key, flag)
	if err != nil {
		return DelResp{false, err}
	}
	return c.DelRecv()
}

func (c *Conn) getSetSend(val []byte) error {
	if val == nil {
		var buf [1 + 8]byte
		err := c.write(buf[:])
		if err != nil {
			return fmt.Errorf("out nil value size failed: %w", err)
		}
		return nil
	}

	size := 1 + 8 + len(val)
	if size <= writeOnceSize {
		buf := make([]byte, 0, size)
		buf = append(buf, 1)
		buf = binary.BigEndian.AppendUint64(buf, uint64(len(val)))
		buf = append(buf, val...)
		err := c.write(buf)
		if err != nil {
			return fmt.Errorf("out once failed: %w", err)
		}
		return nil
	}

	buf := make([]byte, 0, 1+8)
	buf = append(buf, 1)
	buf = binary.BigEndian.AppendUint64(buf, uint64(len(val)))
	err := c.write(buf)
	if err != nil {
		return fmt.Errorf("out value size failed: %w", err)
	}

	err = c.write(val)
	if err != nil {
		return fmt.Errorf("out value failed: %w", err)
	}
	return nil
}

func (c *Conn) getSetRecv() error {
	var buf [1]byte
	err := c.readFull(buf[:])
	if err != nil {
		return fmt.Errorf("in errno failed: %w", err)
	}
	return errnoError(errno(buf[0]))
}

func (c *Conn) getSet(key string, fallbackGet FallbackGetFunc) (val []byte, err error) {
	val, err = fallbackGet((key))
	if err != nil {
		// too much error, can't handle
		c.getSetSend(nil)
		c.getSetRecv()
		return nil, fmt.Errorf("%w: %w", ErrBadFallbackGet, err)
	}

	err = c.getSetSend(val)
	if err != nil {
		return nil, err
	}

	err = c.getSetRecv()
	if err != nil {
		return nil, err
	}
	return val, nil
}

func (c *Conn) GetSet(key string, fallbackGet FallbackGetFunc) GetResp {
	val, err := c.getSet(key, fallbackGet)
	if err != nil {
		return GetResp{false, nil, fmt.Errorf("umem-cache: get-set failed: %w", err)}
	}
	return GetResp{false, val, nil}
}
