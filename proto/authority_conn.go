// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2025, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package proto

import (
	"crypto/tls"
	"encoding/binary"
	"time"
)

type AuthorityConn struct {
	conn *Conn
}

func DialAuthority(deadline time.Time, address string, config *tls.Config) (*AuthorityConn, error) {
	c, err := Dial(deadline, address, config, false)
	if err != nil {
		return nil, err
	}

	err = c.SetDeadline(deadline)
	if err != nil {
		c.Close()
		return nil, err
	}

	req := []byte{byte(_CMD_AUTHORITY)}
	res := make([]byte, 8+8)
	err = c.communicate(req, res)
	if err != nil {
		c.Close()
		return nil, err
	}

	err = c.setReadDeadline(time.Time{})
	if err != nil {
		c.Close()
		return nil, err
	}

	return &AuthorityConn{c}, nil
}

func (c *AuthorityConn) RequestPermission(deadline time.Time) error {
	err := c.conn.setWriteDeadline(deadline)
	if err != nil {
		return err
	}
	return c.conn.write([]byte{1})
}

type Approval struct {
	Version uint64
	Count   uint64
}

func (c *AuthorityConn) RecvApproval() (Approval, error) {
	data := make([]byte, 8+8)
	err := c.conn.read(data)
	if err != nil {
		return Approval{}, err
	}

	return Approval{
		binary.BigEndian.Uint64(data),
		binary.BigEndian.Uint64(data[8:]),
	}, nil
}

func (c *AuthorityConn) Close() {
	c.conn.Close()
}
