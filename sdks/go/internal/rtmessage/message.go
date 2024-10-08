// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
package rtmessage

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

const (
	FLAGS_REQUEST = 1 << iota
	FLAGS_RESPONSE
	FLAGS_UNDELIVERABLE
	FLAGS_TAINTED
	FLAGS_RAW_BINARY
	FLAGS_ENCRYPTED

	header_VERSION       = 2
	header_MARKER        = 0xaaaa
	header_MAX_TOPIC_LEN = 128
	header_MIN           = 32
)

var (
	ErrInvalidMsg = errors.New("invalid message")
)

type Message struct {
	SeqNum     uint32
	CtrlData   uint32
	Flags      uint32
	Topic      string
	ReplyTopic string
	Payload    []byte
	Times      [5]uint32
}

func (m Message) encode() ([]byte, error) {
	if len(m.Payload) > math.MaxUint32 {
		return nil, fmt.Errorf("%w: payload too large", ErrInvalidMsg)
	}
	if len(m.Topic) > header_MAX_TOPIC_LEN {
		return nil, fmt.Errorf("%w: topic too large", ErrInvalidMsg)
	}
	if len(m.ReplyTopic) > header_MAX_TOPIC_LEN {
		return nil, fmt.Errorf("%w: reply topic too large", ErrInvalidMsg)
	}

	buf := make([]byte, 0, 64+header_MAX_TOPIC_LEN*2)

	buf = binary.BigEndian.AppendUint16(buf, header_MARKER)
	buf = binary.BigEndian.AppendUint16(buf, header_VERSION)
	buf = binary.BigEndian.AppendUint16(buf, 0) // Write a placeholder
	buf = binary.BigEndian.AppendUint32(buf, m.SeqNum)
	buf = binary.BigEndian.AppendUint32(buf, m.Flags)
	buf = binary.BigEndian.AppendUint32(buf, m.CtrlData)
	buf = binary.BigEndian.AppendUint32(buf, uint32(len(m.Payload)))
	buf = binary.BigEndian.AppendUint32(buf, uint32(len(m.Topic)))
	buf = append(buf, []byte(m.Topic)...)
	buf = binary.BigEndian.AppendUint32(buf, uint32(len(m.ReplyTopic)))
	buf = append(buf, []byte(m.ReplyTopic)...)

	// Only include timing information if one or more are non zero
	timing := make([]byte, 0, len(m.Times)*4)
	send := false
	for _, time := range m.Times {
		timing = binary.BigEndian.AppendUint32(timing, uint32(time))
		if time != 0 {
			send = true
		}
	}
	if send {
		buf = append(buf, timing...)
	}
	buf = binary.BigEndian.AppendUint16(buf, header_MARKER)

	// Write the actual length.
	binary.BigEndian.PutUint16(buf[4:6], uint16(len(buf)))

	buf = append(buf, m.Payload...)

	return buf, nil
}

func decode(buf []byte) (m Message, err error) {
	// Since error should be few and far between so catching seem like a good approach.
	defer func() {
		if recover() != nil {
			err = fmt.Errorf("%w: buffer too small", ErrInvalidMsg)
		}
	}()

	if header_MARKER != binary.BigEndian.Uint16(buf) {
		return Message{}, fmt.Errorf("%w: invalid leading marker", ErrInvalidMsg)
	}
	headerLen := binary.BigEndian.Uint16(buf[2:3])

	header := buf[4 : headerLen-1]
	m.Payload = buf[headerLen:]

	if header_MARKER != binary.BigEndian.Uint16(header[len(header)-2:]) {
		return Message{}, fmt.Errorf("%w: invalid trailing marker", ErrInvalidMsg)
	}
	header = header[:len(header)-2]

	m.SeqNum = binary.BigEndian.Uint32(header)
	header = header[4:]

	m.Flags = binary.BigEndian.Uint32(header)
	header = header[4:]

	m.CtrlData = binary.BigEndian.Uint32(header)
	header = header[4:]

	payloadLen := binary.BigEndian.Uint32(header)
	header = header[4:]
	if payloadLen != uint32(len(m.Payload)) {
		return Message{}, fmt.Errorf("%w: invalid payload lenght", ErrInvalidMsg)
	}

	topicLen := binary.BigEndian.Uint32(header)
	header = header[4:]
	m.Topic = string(header[:topicLen])
	header = header[topicLen:]

	replyTopicLen := binary.BigEndian.Uint32(header)
	header = header[4:]
	m.ReplyTopic = string(header[:replyTopicLen])
	header = header[replyTopicLen:]

	if len(header) == 0 {
		return m, nil
	}

	if len(header) == 5*4 {
		for i := 0; i < 5; i++ {
			t := binary.BigEndian.Uint32(header)
			header = header[4:]
			m.Times[i] = t
		}
		return m, nil
	}

	return Message{}, fmt.Errorf("%w: unknown protocol version", ErrInvalidMsg)
}

// MessageListener provides a simple way to get notified when a new Message
// is read from the bus.
type MessageListener interface {
	OnMessage(Message)
}

// MessageListenerFunc is a function that implements the MessageListener
// interface.  It is useful for creating a listener from a function.
type MessageListenerFunc func(Message)

func (f MessageListenerFunc) OnMessage(m Message) {
	f(m)
}

// CancelListenerFunc removes the listener it's associated with and cancels any
// future events sent to that listener.
//
// A CancelListenerFunc is idempotent:  after the first invocation, calling this
// closure will have no effect.
type CancelListenerFunc func()
