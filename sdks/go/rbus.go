// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package rbus

import (
	"errors"
	"rbus/internal/rtmessage"
)

// config holds the configuration for the rbus connection
type config struct {
	url     string
	appName string
	id      int
}

// Assure that optionFunc implements the Options interface.
var _ Option = optionFunc(nil)

type Handle struct {
	cfg  config
	conn *rtmessage.Connection
}

// New creates a new rbus handle or returns an error.
func New(opts ...Option) (*Handle, error) {
	var h Handle

	required := []Option{
		assertApplicationName(),
		assertURL(),
	}

	opts = append(opts, required...)

	for _, opt := range opts {
		err := opt.apply(&h.cfg)
		if err != nil {
			return nil, err
		}
	}

	return &h, nil
}

// Open creates a new rbus connection or returns an error.
func (h *Handle) Open() error {
	con, err := rtmessage.New(h.cfg.url)
	if err != nil {
		return err
	}

	err = con.Connect()
	if err != nil {
		return err
	}

	h.conn = con
	return nil
}

func (h *Handle) Get(name string) (*Value, error) {
	return nil, errors.New("not implemented")
}

func (h *Handle) Set(name string, value *Value) error {
	return errors.New("not implemented")
}

func (h *Handle) Close() {
	if h.conn != nil {
		h.conn.Disconnect()
		h.conn = nil
	}
}
