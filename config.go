// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2025, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package client

import (
	"crypto/tls"
	"fmt"
	"time"
)

type Config struct {
	Timeout           time.Duration
	ThreadNR          int
	MaxConnsPerThread int
	TLSConfig         *tls.Config
}

func (conf *Config) check() error {
	if conf.ThreadNR <= 0 {
		return fmt.Errorf("bad ThreadNR: %d", conf.ThreadNR)
	}
	if conf.Timeout <= 0 {
		return fmt.Errorf("bad Timeout: %d", conf.Timeout)
	}
	if conf.MaxConnsPerThread <= 0 {
		conf.MaxConnsPerThread = 0
	}
	return nil
}

func (conf *Config) deadline() time.Time {
	return time.Now().Add(conf.Timeout)
}
