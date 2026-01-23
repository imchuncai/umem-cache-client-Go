// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2025, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package proto

import (
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

type raftCommand byte

const (
	_CMD_REQUEST_VOTE raftCommand = iota
	_CMD_APPEND_LOG
	_CMD_HEARTBEAT
	_CMD_INIT_CLUSTER
	_CMD_CHANGE_CLUSTER
	_CMD_ADMIN_DIVIDER
	_CMD_LEADER
	_CMD_CLUSTER
	_CMD_CONNECT
	_CMD_AUTHORITY
)

type Conn struct {
	v net.Conn
}

func (c *Conn) write(buff []byte) error {
	_, err := c.v.Write(buff)
	return err
}

func (c *Conn) read(buff []byte) error {
	_, err := io.ReadFull(c.v, buff)
	return err
}

func (c *Conn) writev(buffers net.Buffers) error {
	_, err := buffers.WriteTo(c.v)
	return err
}

func (c *Conn) communicate(req []byte, res []byte) error {
	err := c.write(req)
	if err != nil {
		return fmt.Errorf("write failed: %w", err)
	}

	err = c.read(res)
	if err != nil {
		return fmt.Errorf("read failed: %w", err)
	}

	return nil
}

func (c *Conn) communicateV(req net.Buffers, res []byte) error {
	err := c.writev(req)
	if err != nil {
		return fmt.Errorf("writeV failed: %w", err)
	}

	err = c.read(res)
	if err != nil {
		return fmt.Errorf("read failed: %w", err)
	}

	return nil
}

func Dial(deadline time.Time, address string, config *tls.Config) (*Conn, error) {
	var c net.Conn
	var err error
	var tcp *net.TCPConn
	tcpDialer := net.Dialer{Deadline: deadline}
	if config == nil {
		c, err = tcpDialer.Dial("tcp6", address)
		if err != nil {
			return nil, fmt.Errorf("tcp6 dial failed: %w", err)
		}
		tcp = c.(*net.TCPConn)
	} else {
		dialer := tls.Dialer{Config: config, NetDialer: &tcpDialer}
		c, err = dialer.Dial("tcp6", address)
		if err != nil {
			return nil, fmt.Errorf("tls dial failed: %w", err)
		}
		tcp = c.(*tls.Conn).NetConn().(*net.TCPConn)
	}

	err = tcp.SetLinger(0)
	if err != nil {
		return nil, fmt.Errorf("set linger failed: %w", err)
	}
	return &Conn{c}, nil
}

func (c *Conn) SetDeadline(deadline time.Time) error {
	return c.v.SetDeadline(deadline)
}

func (c *Conn) setReadDeadline(deadline time.Time) error {
	return c.v.SetReadDeadline(deadline)
}

func (c *Conn) setWriteDeadline(deadline time.Time) error {
	return c.v.SetWriteDeadline(deadline)
}

func (c *Conn) Close() {
	c.v.Close()
}

func (c *Conn) connect(threadID uint32) error {
	req := make([]byte, 4+4)
	req[0] = byte(_CMD_CONNECT)
	binary.BigEndian.PutUint32(req[4:], threadID)

	res := make([]byte, 1)
	return c.communicate(req, res)
}

func (c *Conn) changeCluster(machines []Machine, cmd raftCommand) error {
	size := len(machines) * _MACHINE_BIN_SIZE
	req := make([]byte, 0, 8+8+size)
	req = append(req, byte(cmd), 0, 0, 0, 0, 0, 0, 0)
	req = binary.BigEndian.AppendUint64(req, uint64(size))
	for i := range machines {
		req = machines[i].append(req)
	}

	res := make([]byte, 1)
	return c.communicate(req, res)
}

func (c *Conn) InitCluster(addrs []*net.TCPAddr) error {
	machines := make([]Machine, len(addrs))
	for i := range machines {
		machines[i] = NewInitialMachine(addrs[i])
	}
	return c.changeCluster(machines, _CMD_INIT_CLUSTER)
}

func (c *Conn) ChangeCluster(machines []Machine) error {
	return c.changeCluster(machines, _CMD_CHANGE_CLUSTER)
}

func (c *Conn) Leader() (string, error) {
	req := []byte{byte(_CMD_LEADER)}
	res := make([]byte, 16+2+2)
	err := c.communicate(req, res)
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

func (c *Conn) Cluster() (Cluster, error) {
	req := []byte{byte(_CMD_CLUSTER)}
	res := make([]byte, 1+7+8+8)
	err := c.communicate(req, res)
	if err != nil {
		return Cluster{}, err
	}

	_type := res[0]
	size := binary.BigEndian.Uint64(res[8:])
	version := binary.BigEndian.Uint64(res[8+8:])
	bin := make([]byte, size)
	err = c.read(bin)
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
