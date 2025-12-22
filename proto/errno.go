// SPDX-License-Identifier: BSD-3-Clause
// Copyright (C) 2025, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

package proto

import (
	"errors"
	"fmt"
)

var (
	// client side error
	ErrClient      = errors.New("client error")
	ErrBadKeySize  = fmt.Errorf("%w: key size out of limit: client: %d cluster: %d", ErrClient, _KEY_SIZE_MAX, _KEY_SIZE_MAX-8)
	ErrFallbackGet = fmt.Errorf("%w: fallback get failed", ErrClient)
)
