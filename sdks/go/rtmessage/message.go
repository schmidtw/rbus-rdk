// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
package rtmessage

import (
	"bytes"
	"encoding/binary"
	"fmt"
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

type Header struct {
	Version        uint16
	HeaderLength   uint16
	SequenceNumber uint32
	Flags          uint32
	ControlData    uint32
	PayloadLength  uint32
	Topic          string
	ReplyTopic     string
}

type Message struct {
	Header  *Header
	Payload []byte
}

func (h *Header) decodePreamble(buff []byte) error {
	const headerMagic = uint16(0xaaaa)

	reader := bytes.NewReader(buff)
	marker := uint16(0)

	// first two bytes are a magic number
	if err := binary.Read(reader, binary.BigEndian, &marker); err != nil {
		return err
	}

	if marker != headerMagic {
		return fmt.Errorf("invalid header maggic: 0x%02x. Expected: 0x%02x", marker, headerMagic)
	}

	// header_version
	if err := binary.Read(reader, binary.BigEndian, &h.Version); err != nil {
		return err
	}

	// header_length
	if err := binary.Read(reader, binary.BigEndian, &h.HeaderLength); err != nil {
		return err
	}

	return nil
}

func (h *Header) decodePostPreamble(buff []byte) error {
	reader := bytes.NewReader(buff)

	if err := binary.Read(reader, binary.BigEndian, &h.SequenceNumber); err != nil {
		return err
	}

	if err := binary.Read(reader, binary.BigEndian, &h.Flags); err != nil {
		return err
	}

	if err := binary.Read(reader, binary.BigEndian, &h.ControlData); err != nil {
		return err
	}

	if err := binary.Read(reader, binary.BigEndian, &h.PayloadLength); err != nil {
		return err
	}

	// topic
	topicLength := uint32(0)
	if err := binary.Read(reader, binary.BigEndian, &topicLength); err != nil {
		return err
	}
	if topicLength > 0 {
		topic := make([]byte, topicLength)
		if _, err := reader.Read(topic); err != nil {
			return err
		}
		h.Topic = string(topic)
	}

	// reply_topic
	replyTopicLen := uint32(0)
	if err := binary.Read(reader, binary.BigEndian, &replyTopicLen); err != nil {
		return err
	}
	if replyTopicLen > 0 {
		replyTopic := make([]byte, replyTopicLen)
		if _, err := reader.Read(replyTopic); err != nil {
			return err
		}
		h.ReplyTopic = string(replyTopic)
	}

	// Those 5 timestamps
	unused := uint32(0)
	if err := binary.Read(reader, binary.BigEndian, &unused); err != nil {
		return err
	}
	if err := binary.Read(reader, binary.BigEndian, &unused); err != nil {
		return err
	}
	if err := binary.Read(reader, binary.BigEndian, &unused); err != nil {
		return err
	}
	if err := binary.Read(reader, binary.BigEndian, &unused); err != nil {
		return err
	}
	if err := binary.Read(reader, binary.BigEndian, &unused); err != nil {
		return err
	}

	magic := uint16(0)
	if err := binary.Read(reader, binary.BigEndian, &magic); err != nil {
		return err
	}

	return nil
}

func (h *Header) encode() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := binary.Write(buf, binary.BigEndian, uint16(header_MARKER)); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.BigEndian, h.Version); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.BigEndian, h.HeaderLength); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.BigEndian, h.SequenceNumber); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.BigEndian, h.Flags); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.BigEndian, h.ControlData); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.BigEndian, h.PayloadLength); err != nil {
		return nil, err
	}

	// read topic length and topic
	topicLength := uint32(len(h.Topic))
	if topicLength == 0 {
		return nil, fmt.Errorf("invalid topic length of zero")
	}

	if err := binary.Write(buf, binary.BigEndian, topicLength); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.BigEndian, []byte(h.Topic)); err != nil {
		return nil, err
	}

	// read the reply topic length and reply topic
	replyTopicLen := uint32(len(h.ReplyTopic))
	if err := binary.Write(buf, binary.BigEndian, replyTopicLen); err != nil {
		return nil, err
	}

	if replyTopicLen > 0 {
		if err := binary.Write(buf, binary.BigEndian, []byte(h.ReplyTopic)); err != nil {
			return nil, err
		}
	}

	// TODO: read the timestamps (not recording them right now)
	zero := 0
	if err := binary.Write(buf, binary.BigEndian, uint32(zero)); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.BigEndian, uint32(zero)); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.BigEndian, uint32(zero)); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.BigEndian, uint32(zero)); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.BigEndian, uint32(zero)); err != nil {
		return nil, err
	}

	// read trailing marker
	if err := binary.Write(buf, binary.BigEndian, uint16(header_MARKER)); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
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
