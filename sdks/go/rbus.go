// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package rbus

import (
	"context"
	"errors"
	"fmt"

	"github.com/schmidtw/rbus-rdk/sdks/go/rbus/rtmessage"
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

	defaults := []Option{
		WithInboxAsPID(),
	}

	opts = append(defaults, opts...)
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
	con, err := rtmessage.New(h.cfg.url, h.cfg.appName)
	if err != nil {
		return err
	}

	err = con.Connect(h.messageHandler)
	if err != nil {
		return err
	}

	h.conn = con
	return nil
}

func (h *Handle) messageHandler(header *rtmessage.Header, payload []byte) {
	msg, err := NewMessageFromBytes(payload)
	if err != nil {
		fmt.Printf("Failed to create message from bytes. %s\n", err.Error())
	} else {
    // TODO: wrap the following up in some type of read
    // the problem is that different rbus messages have different
    // msgpack structures
		returnCode, err := msg.PopInt32()
		if err != nil {
			panic(fmt.Sprintf("Failed to pop int32 for return code. %s\n", err.Error()))
		}

		valueSize, err := msg.PopInt32()
		if err != nil {
			panic(fmt.Sprintf("Failed to pop int32 for value size. %s\n", err.Error()))
		}

		parameterName, err := msg.PopString()
		if err != nil {
			panic(fmt.Sprintf("Failed to pop string. %s\n", err.Error()))
		}

		fmt.Printf("Return code: %d\n", returnCode)
		fmt.Printf("Value size: %d\n", valueSize)
		fmt.Printf("Parameter name: '%s'\n", parameterName)

		value, err := msg.PopValue()
		if err != nil {
			panic(fmt.Sprintf("Failed to pop value. %s\n", err.Error()))
		}

		fmt.Printf("%s == %s\n", parameterName, value.String())

	}
}

func (h *Handle) Get(ctx context.Context, name string) (*Value, error) {
	if h.conn == nil {
		return nil, errors.New("connection not open")
	}

	msg := NewMessage()
	msg.AppendString(h.cfg.appName)
	msg.AppendInt32(1)
	msg.AppendString(name)
	msg.SetMetaInfo("METHOD_GETPARAMETERVALUES", "todo_openTelemetry_parent", "todo_openTelemetry_state")

	if err := h.conn.SendRequest(ctx, msg.Bytes(), name); err != nil {
		return nil, err
	}

	// TODO: what do we do now? Where does user get result

	return nil, nil
}

func (h *Handle) Set(name string, value *Value) error {
	return errors.New("not implemented")
}

func (h *Handle) Close() error {
	var err error
	if h.conn != nil {
		err = h.conn.Disconnect()
		h.conn = nil
	}

	return err
}
