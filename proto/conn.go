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
	"net"
	"time"
)

type command byte

const (
	_CMD_CONNECT command = iota
	_CMD_LEADER
	_CMD_CLUSTER
	_CMD_AUTHORITY
	_CMD_ADMIN_DIVIDER
	_CMD_REQUEST_VOTE
	_CMD_APPEND_LOG
	_CMD_HEARTBEAT
	_CMD_INIT_CLUSTER
	_CMD_CHANGE_CLUSTER
)

func dial(deadline time.Time, address string, config *tls.Config) (net.Conn, error) {
	tcpDialer := net.Dialer{Deadline: deadline}
	if config == nil {
		return tcpDialer.Dial("tcp6", address)
	} else {
		dialer := tls.Dialer{Config: config, NetDialer: &tcpDialer}
		return dialer.Dial("tcp6", address)
	}
}

func communicate(deadline time.Time, conn net.Conn, req []byte, res []byte) error {
	err := conn.SetDeadline(deadline)
	if err != nil {
		return fmt.Errorf("set deadline failed: %w", err)
	}

	_, err = conn.Write(req)
	if err != nil {
		return fmt.Errorf("write failed: %w", err)
	}

	_, err = conn.Read(res)
	if err != nil {
		return fmt.Errorf("read failed: %w", err)
	}

	return nil
}

type Conn struct {
	conn   net.Conn
	reader *bufio.Reader
}

func Dial(deadline time.Time, address string, config *tls.Config) (*Conn, error) {
	conn, err := dial(deadline, address, config)
	if err != nil {
		return nil, err
	}

	return &Conn{conn, bufio.NewReaderSize(conn, 1024)}, nil
}

func (c *Conn) bufferRead(b []byte) error {
	_, err := io.ReadFull(c.reader, b)
	return err
}

func (c *Conn) communicate(deadline time.Time, req []byte, res []byte) error {
	return communicate(deadline, c.conn, req, res)
}

func (c *Conn) bufferCommunicate(deadline time.Time, req []byte, res []byte) error {
	err := c.conn.SetDeadline(deadline)
	if err != nil {
		return fmt.Errorf("set deadline failed: %w", err)
	}

	_, err = c.conn.Write(req)
	if err != nil {
		return fmt.Errorf("write failed: %w", err)
	}

	err = c.bufferRead(res)
	if err != nil {
		return fmt.Errorf("buffer read failed: %w", err)
	}

	return nil
}

func (c *Conn) changeCluster(deadline time.Time, machines []Machine, cmd command) error {
	size := len(machines) * _MACHINE_BIN_SIZE
	req := make([]byte, 0, 8+8+size)
	req = append(req, byte(cmd), 0, 0, 0, 0, 0, 0, 0)
	req = binary.LittleEndian.AppendUint64(req, uint64(size))
	for i := range machines {
		req = machines[i].append(req)
	}
	return c.communicate(deadline, req, []byte{0})
}

func (c *Conn) InitCluster(deadline time.Time, addrs []*net.TCPAddr) error {
	machines := make([]Machine, len(addrs))
	for i := range machines {
		machines[i] = NewInitialMachine(addrs[i])
	}
	return c.changeCluster(deadline, machines, _CMD_INIT_CLUSTER)
}

func (c *Conn) ChangeCluster(deadline time.Time, machines []Machine) error {
	return c.changeCluster(deadline, machines, _CMD_CHANGE_CLUSTER)
}

func (c *Conn) Leader(deadline time.Time) (string, error) {
	req := []byte{byte(_CMD_LEADER)}
	res := make([]byte, 16+2+2)
	err := c.communicate(deadline, req, res)
	if err != nil {
		return "", err
	}

	if res[18] == 1 {
		return "", errors.New("lost leader")
	}

	addr := new(net.TCPAddr)
	addr.IP = make([]byte, 16)
	copy(addr.IP, res)
	addr.Port = int(binary.BigEndian.Uint16(res[16:]))
	return addr.String(), nil
}

func (c *Conn) Cluster(deadline time.Time) (Cluster, error) {
	req := []byte{byte(_CMD_CLUSTER)}
	res := make([]byte, 1+7+8+8)
	err := c.bufferCommunicate(deadline, req, res)
	if err != nil {
		return Cluster{}, err
	}

	_type := res[0]
	size := binary.LittleEndian.Uint64(res[8:])
	version := binary.LittleEndian.Uint64(res[8+8:])
	bin := make([]byte, size)
	err = c.bufferRead(bin)
	if err != nil {
		return Cluster{}, fmt.Errorf("read machines failed: %w", err)
	}

	n := size / _MACHINE_BIN_SIZE
	machines := make([]Machine, n)
	for i := range n {
		machines[i] = newMachine(bin)
		bin = bin[_MACHINE_BIN_SIZE:]
	}

	return Cluster{ClusterType(_type), version, machines}, nil
}

func (c *Conn) Close() {
	c.conn.Close()
}
