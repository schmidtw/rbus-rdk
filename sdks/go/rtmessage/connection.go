// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
package rtmessage

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xmidt-org/eventor"
)

// Connection represents a connection to the server.
type Connection struct {
	url           *url.URL
	name          string
	id            int
	cancel        context.CancelFunc
	conn          net.Conn
	m             sync.Mutex
	readTimeout   time.Duration
	writeTimeout  time.Duration
	subscriptions map[string]struct{}
	msgListeners  eventor.Eventor[MessageListener]
	errListeners  eventor.Eventor[ReadErrorListener]
	routeID       uint32 // only access via atomic operations
}

// New creates a new connection or returns an error.
func New(rawURL string, name string, id int, opts ...Option) (*Connection, error) {
	c := Connection{
		name: name,
		id:   id,
	}

	required := []Option{
		withRawURL(rawURL),
	}

	opts = append(opts, required...)

	for _, opt := range opts {
		err := opt.apply(&c)
		if err != nil {
			return nil, err
		}
	}

	return &c, nil
}

// AddReadErrorListener adds a listener for read errors.
func (c *Connection) AddReadErrorListener(listener ReadErrorListener) CancelListenerFunc {
	return c.errListeners.Add(listener)
}

// AddMessageListener adds a listener for messages.
func (c *Connection) AddMessageListener(listener MessageListener) CancelListenerFunc {
	return c.msgListeners.Add(listener)
}

// Connect establishes a connection to the server.
func (c *Connection) Connect() error {
	c.m.Lock()
	defer c.m.Unlock()

	if c.conn != nil {
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

	ctx, cancel := context.WithCancel(context.Background())
	c.conn = con
	c.cancel = cancel

	go c.readLoop(ctx)

	// Subscribe to the inbox
	list := []string{fmt.Sprintf("%s.INBOX.%d", c.name, c.id)}
	for topic, _ := range c.subscriptions {
		list = append(list, topic)
	}

	// Subscribe to everything in the list.
	for _, topic := range list {
		err := c.subscribe(ctx, topic)
		if err != nil {
			return err
		}
	}

	return nil
}

// Disconnect closes the connection to the server.
func (c *Connection) Disconnect() error {
	c.m.Lock()
	defer c.m.Unlock()

	if c.conn == nil {
		return nil
	}

	c.cancel()
	err := c.conn.Close()
	c.conn = nil
	c.cancel = nil

	return err
}

// Send sends a message to the server.  If the context is canceled, the function
// will return immediately with the context error.
func (c *Connection) Send(ctx context.Context, msg Message) error {
	c.m.Lock()
	defer c.m.Unlock()
	return c.send(ctx, msg)
}

// Subscribe subscribes to a topic.
func (c *Connection) Subscribe(ctx context.Context, expression string) error {
	c.m.Lock()
	defer c.m.Unlock()
	c.subscriptions[expression] = struct{}{}
	return c.subscribe(ctx, expression)
}

func sooner(timeout time.Duration, ctx context.Context) time.Time {
	deadline := time.Time{}
	if when, valid := ctx.Deadline(); valid {
		deadline = when
	}

	if timeout > 0 {
		when := time.Now().Add(timeout)
		if !deadline.IsZero() && deadline.After(when) {
			return when
		}
	}

	return deadline
}

// setReadDeadline sets the read deadline on the connection.
func (c *Connection) setReadDeadline(ctx context.Context) error {
	if c.conn == nil {
		return ErrInvalidState
	}

	when := sooner(c.readTimeout, ctx)
	if !when.IsZero() {
		return c.conn.SetReadDeadline(when)
	}
	return nil
}

// setWriteDeadline sets the write deadline on the connection.
func (c *Connection) setWriteDeadline(ctx context.Context) error {
	if c.conn == nil {
		return ErrInvalidState
	}

	when := sooner(c.readTimeout, ctx)
	if !when.IsZero() {
		return c.conn.SetWriteDeadline(when)
	}
	return nil
}

// readLoop reads messages from the server and sends events to registered listeners.
func (c *Connection) readLoop(ctx context.Context) {
	defer func() {
		_ = c.Disconnect()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			err := c.setReadDeadline(ctx)
			if err != nil {
				c.errListeners.Visit(func(listener ReadErrorListener) {
					listener.OnReadError(err)
				})
				return
			}

			msg, err := unmarshal(c.conn)
			if err != nil {
				c.errListeners.Visit(func(listener ReadErrorListener) {
					listener.OnReadError(err)
				})
				return
			}

			c.msgListeners.Visit(func(listener MessageListener) {
				listener.OnMessage(msg)
			})
		}
	}
}

// send sends a message to the server.
func (c *Connection) send(ctx context.Context, msg Message) error {
	if c.conn == nil {
		return ErrInvalidState
	}

	b, err := msg.marshal()
	if err != nil {
		return err
	}

	total := len(b)
	sent := 0

	for sent < total {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			err := c.setWriteDeadline(ctx)
			if err != nil {
				return err
			}

			n, err := c.conn.Write(b[sent:])
			if err != nil {
				return err
			}
			sent += n
		}
	}

	return nil
}

// nextRouteID returns the next route ID.
func (c *Connection) nextRouteID() int {
	return int(atomic.AddUint32(&c.routeID, 1))
}

// subscribe subscribes to a topic.
func (c *Connection) subscribe(ctx context.Context, expression string) error {
	req := struct {
		Topic   string `json:"topic"`
		Add     int    `json:"add"`
		RouteID int    `json:"route_id"`
	}{
		Topic:   expression,
		Add:     1,
		RouteID: c.nextRouteID(),
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return err
	}

	m := Message{
		Topic:   "_RTROUTED.INBOX.SUBSCRIBE",
		Payload: jsonData,
	}

	return c.send(ctx, m)
}
