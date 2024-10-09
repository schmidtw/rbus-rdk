// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
package rtmessage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"sync"
	"sync/atomic"

	"github.com/xmidt-org/eventor"
)

var (
	ErrInvalidState = errors.New("invalid state")
	ErrInvalidInput = errors.New("invalid input")
)

type SubscriptionIDGenerator struct {
	counter uint32
}

func (s *SubscriptionIDGenerator) getNextSubscriptionID() int {
	return int(atomic.AddUint32(&s.counter, 1))
}

type ReadState int

const (
	ReadStateReadHeaderPreamble = iota
	ReadStateReadHeader
	ReadStateReadPayload
)

type Connection struct {
	url       *url.URL
	con       net.Conn
	cancel    context.CancelFunc
	m         sync.Mutex
	appName   string
	generator SubscriptionIDGenerator
	state     ReadState
	listeners eventor.Eventor[MessageListener]
}

type subscriptionRequest struct {
	Topic   string `json:"topic"`
	Add     int    `json:"add"`
	RouteID int    `json:"route_id"`
}

// New creates a new connection or returns an error.
func New(rawURL string, appName string) (*Connection, error) {
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
		url:     u,
		appName: appName,
		state:   ReadStateReadHeaderPreamble,
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

	ctx, cancel := context.WithCancel(context.Background())
	c.con = con
	c.cancel = cancel

	// TODO: check if this is needed
	// inboxName := fmt.Sprintf("%s.INBOX.%d", c.appName, os.Getpid())
	// c.AddListener(inboxName)

	go c.readLoop(ctx)

	return nil
}

// Disconnect closes the connection to the server.
func (c *Connection) Disconnect() error {
	c.m.Lock()
	defer c.m.Unlock()

	if c.con == nil {
		return nil
	}

	c.cancel()
	err := c.con.Close()
	c.con = nil
	c.cancel = nil

	return err
}

func (c *Connection) Add(listener MessageListener, expression string) (CancelListenerFunc, error) {
	req := subscriptionRequest{
		Topic:   expression,
		Add:     1,
		RouteID: c.generator.getNextSubscriptionID(),
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	if err := c.Send(context.Background(), jsonData, "_RTROUTED.INBOX.SUBSCRIBE"); err != nil {
		return nil, err
	}

	// TODO: you need to associate this listener with the req.RouteID
	return CancelListenerFunc(c.listeners.Add(listener)), nil
}

func sendAll(ctx context.Context, conn net.Conn, buff []byte) error {
	total := len(buff)
	sent := 0

	for sent < total {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			n, err := conn.Write(buff[sent:])
			if err != nil {
				return err
			}
			sent += n
		}
	}

	return nil
}

func (c *Connection) makeEncodedHeader(payload []byte, topic string, replyTopic string) ([]byte, error) {
	topicLength := len(topic)
	replyTopicLength := 0
	sizeWithoutStringsInBytes := 32
	sizeWithoutStringsInBytes += 20 // 4 time_t fields

	header := Header{
		Version:        2,
		HeaderLength:   uint16(sizeWithoutStringsInBytes + topicLength + replyTopicLength),
		SequenceNumber: uint32(c.generator.getNextSubscriptionID()),
		Flags:          0,
		ControlData:    0,
		PayloadLength:  uint32(len(payload)),
		Topic:          topic,
		ReplyTopic:     replyTopic,
	}

	encodedHeader, err := header.encode()
	if err != nil {
		return nil, err
	}

	return encodedHeader, nil
}

// Send sends a message to the server.  If the context is canceled, the function
// will return immediately with the context error.
func (c *Connection) Send(ctx context.Context, payload []byte, topic string) error {
	encodedHeader, err := c.makeEncodedHeader(payload, topic, "")
	if err != nil {
		return err
	}

	return c.sendWithHeader(ctx, encodedHeader, payload)
}

func (c *Connection) sendWithHeader(ctx context.Context, header []byte, payload []byte) error {
	c.m.Lock()
	defer c.m.Unlock()

	if c.con == nil {
		return ErrInvalidState
	}

	if err := sendAll(ctx, c.con, header); err != nil {
		return err
	}

	if err := sendAll(ctx, c.con, payload); err != nil {
		return err
	}

	return nil
}

func (c *Connection) readUntil(ctx context.Context, buf []byte) error {
	total := len(buf)
	read := 0

	for read < total {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			n, err := c.con.Read(buf[read:])
			if err != nil {
				return err
			}
			read += n
		}
	}

	return nil
}

// readLoop reads messages from the server and sends events to registered listeners.
func (c *Connection) readLoop(ctx context.Context) {
	var header *Header

	const headerPreambleLength = uint16(6)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			switch c.state {
			case ReadStateReadHeaderPreamble:
				header = &Header{}
				buff := make([]byte, headerPreambleLength)

				if err := c.readUntil(ctx, buff); err != nil {
					fmt.Printf("Failed to read header preamble: %v\n", err)
					return
				}

				if err := header.decodePreamble(buff); err != nil {
					fmt.Printf("Failed to decode header preamble: %v\n", err)
					return
				}

				c.state = ReadStateReadHeader

			case ReadStateReadHeader:
				buff := make([]byte, header.HeaderLength-headerPreambleLength)

				if err := c.readUntil(ctx, buff); err != nil {
					fmt.Printf("Failed to read header: %v\n", err)
					return
				}

				if err := header.decodePostPreamble(buff); err != nil {
					fmt.Printf("Failed to decode header: %v\n", err)
					return
				}

				c.state = ReadStateReadPayload

			case ReadStateReadPayload:
				buff := make([]byte, header.PayloadLength)
				if err := c.readUntil(ctx, buff); err != nil {
					fmt.Printf("Failed to read payload: %v\n", err)
					return
				}

				c.listeners.Visit(func(listener MessageListener) {
					listener.OnMessage(Message{
						Header:  header,
						Payload: buff,
					})
				})

				c.state = ReadStateReadHeaderPreamble
			}
		}
	}
}
