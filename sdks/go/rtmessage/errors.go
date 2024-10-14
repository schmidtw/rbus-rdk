// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package rtmessage

import "errors"

var (
	ErrInvalidMessage = errors.New("invalid message")
	ErrInvalidState   = errors.New("invalid state")
	ErrInvalidInput   = errors.New("invalid input")
)
