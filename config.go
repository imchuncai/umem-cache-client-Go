// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2025-2026, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package client

import (
	"crypto/tls"
	"fmt"
	"time"
)

type Config struct {
	Timeout           time.Duration
	MaxConnsPerThread uint // 0 for no limit
	TLSConfig         *tls.Config
}

func (conf *Config) check() error {
	if conf.Timeout <= 0 {
		return fmt.Errorf("bad Timeout: %d", conf.Timeout)
	}
	return nil
}

func (conf *Config) deadline() time.Time {
	return time.Now().Add(conf.Timeout)
}
