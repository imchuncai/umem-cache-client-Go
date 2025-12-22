// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2025, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package client

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/imchuncai/umem-cache-client-Go/proto"
)

const (
	SEED            = uint64(47)
	THREAD_NR       = 4
	KV_SIZE_LIMIT   = 1 << 20
	SERVER_MEMORY   = 100 << 20
	TIMEOUT         = 10 * time.Second
	THREAD_MAX_CONN = 512 / THREAD_NR
)

func ErrIsClosedByPeer(err error) bool {
	return errors.Is(err, io.EOF) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, io.ErrClosedPipe)
}

func DEADLINE() time.Time {
	return time.Now().Add(TIMEOUT)
}

type TestParam struct {
	ExePath string
	Config  Config
	Debug   bool
}

func InitTest(t *testing.T) TestParam {
	args := flag.Args()
	if len(args) < 3 {
		t.Fatal("bad args")
	}

	var param TestParam
	param.ExePath = args[0]
	param.Debug = args[2] != "0"
	param.Config.Timeout = TIMEOUT
	param.Config.ThreadNR = THREAD_NR
	_tls := args[1] != "0"
	if _tls {
		cert, err := tls.LoadX509KeyPair("cert.pem", "key.pem")
		if err != nil {
			t.Fatal(err)
		}

		caCert, err := os.ReadFile("ca-cert.pem")
		if err != nil {
			t.Fatal(err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		param.Config.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      caCertPool,
		}
	}
	return param
}

type Case struct {
	Key []byte
	Val []byte
}

func (c Case) String() string {
	return fmt.Sprintf("key size: %d, value size: %d", len(c.Key), len(c.Val))
}

type Pool struct {
	keySizeLimit int
	fuzzN        int
	r            *rand.Rand
	kv           []byte
}

func NewPool(keyMaxSize int, fuzzN int) Pool {
	p := Pool{
		keyMaxSize,
		fuzzN,
		rand.New(rand.NewPCG(SEED, SEED)),
		make([]byte, KV_SIZE_LIMIT+1),
	}
	for i := range p.kv {
		p.kv[i] = byte(p.r.Int())
	}
	return p
}

func (p *Pool) randN(n int) []byte {
	i := p.r.IntN(len(p.kv) - n)
	return p.kv[i : i+n]
}

func (p *Pool) RandCase() Case {
	n := p.r.IntN(KV_SIZE_LIMIT + 1)
	k := p.r.IntN(p.keySizeLimit + 1)
	if n <= k {
		n = k
	}
	return Case{p.randN(k), p.randN(n - k)}
}

type Machine struct {
	addr string
	cmd  *exec.Cmd
}

func (m *Machine) Ping(deadline time.Time, config *tls.Config) error {
	for {
		nap()
		conn, err := proto.Dial(deadline, m.addr, config, true)
		if err == nil {
			conn.Close()
			return nil
		}
		if errors.Is(err, os.ErrDeadlineExceeded) {
			return err
		}
	}
}

func MachineAddress(port int) string {
	return "[::1]:" + strconv.Itoa(port)
}

func runTestMachine(port int, param TestParam) (*Machine, error) {
	address := MachineAddress(port)
	_, err := net.ResolveTCPAddr("tcp6", address)
	if err != nil {
		return nil, fmt.Errorf("resolve address: %s failed: %w", address, err)
	}

	cmd := exec.Command(param.ExePath, strconv.Itoa(port), "cert.pem", "key.pem", "ca-cert.pem")
	if param.Debug {
		err := os.MkdirAll("logs", 0777)
		if err != nil {
			return nil, fmt.Errorf("make dir logs failed: %w", err)
		}
		name := fmt.Sprintf("logs/%d.log", port)
		logFile, err := os.OpenFile(name, os.O_APPEND|os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
		if err != nil {
			return nil, fmt.Errorf("open file: %s failed: %w", name, err)
		}

		cmd.Stdout = logFile
		cmd.Stderr = logFile
		cmd.ExtraFiles = []*os.File{logFile}
	}
	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("start server failed: %w", err)
	}

	return &Machine{address, cmd}, nil
}

func RunMachine(port int, param TestParam) (*Machine, error) {
	machine, err := runTestMachine(port, param)
	if err != nil {
		return nil, err
	}

	deadline := DEADLINE()
	err = machine.Ping(deadline, param.Config.TLSConfig)
	if err != nil {
		machine.Stop()
		return nil, err
	}
	return machine, nil
}

func (m *Machine) Stop() error {
	if m.cmd == nil {
		return nil
	}

	for _, f := range m.cmd.ExtraFiles {
		err := f.Close()
		if err != nil {
			return fmt.Errorf("close extra file failed: %w", err)
		}
	}
	err := m.cmd.Process.Signal(syscall.SIGTERM)
	if err != nil {
		return fmt.Errorf("signal SIGTERM failed: %w", err)
	}
	_, err = m.cmd.Process.Wait()
	if err != nil {
		return fmt.Errorf("wait failed: %w", err)
	}
	m.cmd = nil
	return nil
}

type Machines []*Machine

func RunMachines(ports []int, param TestParam) (Machines, error) {
	machines := make(Machines, 0, len(ports))
	for _, port := range ports {
		m, err := runTestMachine(port, param)
		if err != nil {
			machines.Stop()
			return nil, fmt.Errorf("run machine on port: %d failed: %w", port, err)
		}
		machines = append(machines, m)
	}

	deadline := DEADLINE()

	err := machines.Ping(deadline, param.Config.TLSConfig)
	if err != nil {
		machines.Stop()
		return nil, err
	}
	return machines, nil
}

func (machines Machines) Ping(deadline time.Time, config *tls.Config) error {
	for _, m := range machines {
		err := m.Ping(deadline, config)
		if err != nil {
			return fmt.Errorf("signal machine: %s failed: %w", m.addr, err)
		}
	}
	return nil
}

func (machines Machines) Stop() error {
	var err error
	for _, m := range machines {
		e := m.Stop()
		if e != nil {
			err = errors.Join(err, fmt.Errorf("stop machine: %s failed: %w", m.addr, e))
		}
	}
	return err
}

type ClientInterface interface {
	GetOrSet(key []byte, get proto.FallbackGetFunc) (val []byte, err error)
	Del(key []byte) error
	Close()
}

func testBasic(t *testing.T, client ClientInterface, pool Pool) {
	t.Run("Get", testGet(client, pool))
	t.Run("Del", testDel(client, pool))
	t.Run("GetOrSet", testGetOrSet(client, pool))
	// Note: BadGetOrSet takes more time than GetOrSet, because
	// when FallbackGet the connection will be closed
	t.Run("BadGetOrSet", testBadGetOrSet(client, pool))
	t.Run("ConcurrentSet", testConcurrentSet(client, pool))
}

func fallbackGet(tc Case) proto.FallbackGetFunc {
	return func(key []byte) (val []byte, err error) {
		return tc.Val, nil
	}
}

func getOrSet(tb testing.TB, client ClientInterface, tc Case) []byte {
	val, err := client.GetOrSet(tc.Key, fallbackGet(tc))
	if err != nil {
		tb.Fatalf("got error: %v test case: %v", err, tc)
	}
	return val
}

func del(tb testing.TB, client ClientInterface, tc Case) {
	err := client.Del(tc.Key)
	if err != nil {
		tb.Fatalf("got error: %v, key size: %d", err, len(tc.Key))
	}
}

func set(tb testing.TB, client ClientInterface, tc Case) {
	del(tb, client, tc)
	getOrSet(tb, client, tc)
}

func check(tb testing.TB, client ClientInterface, tc Case) {
	val := getOrSet(tb, client, tc)
	if string(val) != string(tc.Val) {
		tb.Fatalf("want: %v, got size: %d", tc, len(val))
	}
}

var errBadFallbackGet = errors.New("bad fallback get")

func badFallbackGet(key []byte) (val []byte, err error) {
	return nil, errBadFallbackGet
}

func badGetOrSet(t *testing.T, client ClientInterface, tc Case) {
	_, err := client.GetOrSet(tc.Key, badFallbackGet)
	if !errors.Is(err, errBadFallbackGet) {
		t.Fatalf("want error: %v, got error: %v test case: %v",
			errBadFallbackGet, err, tc)
	}
}

func fuzz(pool Pool, f func(t *testing.T, tc Case, randV []byte)) func(t *testing.T) {
	return func(t *testing.T) {
		keysN := []int{0, pool.keySizeLimit / 2, pool.keySizeLimit}
		valsN := []int{0, 1}
		for _, kn := range keysN {
			for _, vn := range valsN {
				key := pool.randN(kn)
				val := pool.randN(vn)
				val2 := pool.randN(vn)
				f(t, Case{key, val}, val2)
			}
		}

		for i := 0; i < pool.fuzzN; i++ {
			f(t, pool.RandCase(), pool.RandCase().Val)
		}
	}
}

func testGet(client ClientInterface, pool Pool) func(t *testing.T) {
	return fuzz(pool, func(t *testing.T, tc Case, randV []byte) {
		set(t, client, tc)
		check(t, client, tc)
	})
}

func testDel(client ClientInterface, pool Pool) func(t *testing.T) {
	return fuzz(pool, func(t *testing.T, tc Case, randV []byte) {
		set(t, client, tc)
		del(t, client, tc)
		tc.Val = nil
		check(t, client, tc)

		del(t, client, tc)
		check(t, client, tc)
	})
}

func testGetOrSet(client ClientInterface, pool Pool) func(t *testing.T) {
	return fuzz(pool, func(t *testing.T, tc Case, randV []byte) {
		set(t, client, tc)
		tc2 := Case{tc.Key, randV}
		getOrSet(t, client, tc2)
		check(t, client, tc)

		del(t, client, tc)
		getOrSet(t, client, tc2)
		check(t, client, tc2)
	})
}

func testBadGetOrSet(client ClientInterface, pool Pool) func(t *testing.T) {
	return fuzz(pool, func(t *testing.T, tc Case, randV []byte) {
		del(t, client, tc)
		badGetOrSet(t, client, tc)
		tc.Val = nil
		check(t, client, tc)
	})
}

func testConcurrentSet(client ClientInterface, pool Pool) func(t *testing.T) {
	return fuzz(pool, func(t *testing.T, tc Case, randV []byte) {
		del(t, client, tc)

		fallbackGet := func(key []byte) (val []byte, err error) {
			del(t, client, tc)
			return tc.Val, nil
		}

		_, err := client.GetOrSet(tc.Key, fallbackGet)
		if err != nil {
			t.Fatal(err)
		}

		tc.Val = nil
		check(t, client, tc)
	})
}
