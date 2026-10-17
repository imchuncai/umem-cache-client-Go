// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2025-2026, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package proto

import (
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

type AuthorityConn struct {
	conn net.Conn
}

func authority(deadline time.Time, conn net.Conn) error {
	req := []byte{byte(_CMD_AUTHORITY)}
	res := make([]byte, 8+8)
	return communicate(deadline, conn, req, res)
}

func DialAuthority(deadline time.Time, address string, config *tls.Config) (*AuthorityConn, error) {
	conn, err := dial(deadline, address, config)
	if err != nil {
		return nil, err
	}

	err = authority(deadline, conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("request authority failed: %w", err)
	}

	err = conn.SetReadDeadline(time.Time{})
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("set read deadline failed: %w", err)
	}

	return &AuthorityConn{conn}, nil
}

func (c *AuthorityConn) RequestPermission(deadline time.Time, n int) error {
	err := c.conn.SetWriteDeadline(deadline)
	if err != nil {
		return err
	}

	k := n / 255
	buff := make([]byte, k, k+1)
	for i := range k {
		buff[i] = 255
	}

	n -= k * 255
	if n > 0 {
		buff = append(buff, byte(n))
	}
	_, err = c.conn.Write(buff)
	return err
}

type Approval struct {
	Version uint64
	Count   uint64
}

func (c *AuthorityConn) RecvApproval() (Approval, error) {
	data := make([]byte, 8+8)
	_, err := c.conn.Read(data)
	if err != nil {
		return Approval{}, err
	}

	return Approval{
		binary.LittleEndian.Uint64(data),
		binary.LittleEndian.Uint64(data[8:]),
	}, nil
}

func (c *AuthorityConn) Close() {
	c.conn.Close()
}
