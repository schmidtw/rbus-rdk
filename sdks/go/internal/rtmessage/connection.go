// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
package rtmessage

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

var (
	ErrInvalidState = errors.New("invalid state")
	ErrInvalidInput = errors.New("invalid input")
)

type Config struct {
	URL             string
	ApplicationName string
}

type Connection struct {
	network string
	address string
	con     net.Conn
	m       sync.Mutex
}

func createInboxName(appName string) string {
	return fmt.Sprintf("%s.INBOX.%d", appName, os.Getpid())
}

func NewConnection(c Config) (*Connection, error) {
	parts := strings.Split(c.URL, "://")
	switch parts[0] {
	case "unix":
		parts[1] = "/" + parts[1]
	case "tcp":
	default:
		return nil, ErrInvalidInput
	}

	return &Connection{
		network: parts[0],
		address: parts[1],
	}, nil
}

func (c *Connection) Connect() error {
	c.m.Lock()
	defer c.m.Unlock()

	if c.con != nil {
		return nil
	}

	con, err := net.Dial(c.network, c.address)
	if err != nil {
		return err
	}

	c.con = con
	return nil
}

func (c *Connection) Disconnect() error {
	c.m.Lock()
	defer c.m.Unlock()

	err := c.con.Close()
	c.con = nil
	return err
}

func (c *Connection) Send(m *Message) error {
	c.m.Lock()
	defer c.m.Unlock()

	if c.con == nil {
		return ErrInvalidState
	}

	buf, err := m.Encode()
	if err != nil {
		return err
	}

	n, err := c.con.Write(buf)
	if err != nil {
		return err
	}

	if n < len(buf) {
		return errors.New("not all bytes were sent")
	}

	return nil
}

func (c *Connection) Read() (*Message, error) {
	c.m.Lock()
	defer c.m.Unlock()

	if c.con == nil {
		return nil, ErrInvalidState
	}

	buf := make([]byte, 4096)

	n, err := c.con.Read(buf)
	if err != nil {
		return nil, err
	}

	buf = buf[0:n]

	return Decode(buf)
}
