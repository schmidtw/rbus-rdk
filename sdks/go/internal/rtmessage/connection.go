// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
package rtmessage

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"sync"

	"github.com/xmidt-org/eventor"
)

var (
	ErrInvalidState = errors.New("invalid state")
	ErrInvalidInput = errors.New("invalid input")
)

type Connection struct {
	url       *url.URL
	con       net.Conn
	m         sync.Mutex
	listeners eventor.Eventor[MessageListener]
}

// I don't think this belongs in here since it doesn't seem to be related
// to the rtmessage code directly.
/*
func createInboxName(appName string) string {
	return fmt.Sprintf("%s.INBOX.%d", appName, os.Getpid())
}
*/

// New creates a new connection or returns an error.
func New(rawURL string) (*Connection, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "unix", "tcp":
	default:
		return nil, fmt.Errorf("%w: unsupported URL scheme", ErrInvalidInput)
	}

	return &Connection{
		url: u,
	}, nil
}

// Connect establishes a connection to the server.
func (c *Connection) Connect() error {
	c.m.Lock()
	defer c.m.Unlock()

	if c.con != nil {
		return nil
	}

	var con net.Conn
	var err error

	switch c.url.Scheme {
	case "unix":
		con, err = net.Dial(c.url.Scheme, c.url.Path)
	case "tcp":
		con, err = net.Dial(c.url.Scheme, c.url.Host)
	}

	if err != nil {
		return err
	}

	c.con = con

	go c.readLoop()

	return nil
}

// Disconnect closes the connection to the server.
func (c *Connection) Disconnect() error {
	c.m.Lock()
	defer c.m.Unlock()

	if c.con == nil {
		return nil
	}

	err := c.con.Close()
	c.con = nil
	return err
}

// Send sends a message to the server.  If the context is canceled, the function
// will return immediately with the context error.
func (c *Connection) Send(ctx context.Context, m *Message) error {
	buf, err := m.encode()
	if err != nil {
		return err
	}

	c.m.Lock()
	defer c.m.Unlock()

	if c.con == nil {
		return ErrInvalidState
	}

	total := len(buf)
	sent := 0

	for sent < total {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			n, err := c.con.Write(buf[sent:])
			if err != nil {
				return err
			}
			sent += n
		}
	}

	return nil
}

// Add adds an event listener to the message listener.
// The listener will be called for each event that occurs.  The returned
// function can be called to remove the listener.
func (c *Connection) Add(listener MessageListener) CancelListenerFunc {
	return CancelListenerFunc(c.listeners.Add(listener))
}

// readLoop reads messages from the server and sends events to registered listeners.
func (c *Connection) readLoop() {
	buf := make([]byte, 4096)

	for {
		c.m.Lock()
		if c.con == nil {
			c.m.Unlock()
			return
		}
		c.m.Unlock()

		_, err := c.con.Read(buf)
		if err != nil {
			if err == net.ErrClosed {
				return
			}
			// TODO this probably needs to be better.
			continue
		}

		msg, err := decode(buf)
		if err != nil {
			// TODO this probably needs to be better.
			continue
		}

		// Send the event to registered listeners
		c.listeners.Visit(func(listener MessageListener) {
			listener.OnMessage(msg)
		})
	}
}
