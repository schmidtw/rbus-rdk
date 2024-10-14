// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package rtmessage

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"time"
)

const (
	flags_REQUEST = 1 << iota
	flags_RESPONSE
	flags_UNDELIVERABLE
	flags_TAINTED
	flags_RAW_BINARY
	flags_ENCRYPTED

	header_VERSION       = 2
	header_MARKER        = 0xaaaa
	header_MAX_TOPIC_LEN = 128
	header_LEN_NO_TS     = 32
	header_LEN_W_TS      = 52
)

//	Wire format
//  --------------------------------------
//	Preamble         uint16
//	Version          uint16
//	HeaderLength     uint16
//	SequenceNumber   uint32
//	Flags            uint32
//	ControlData      uint32
//	PayloadLength    uint32
//	TopicLength      uint32
//	Topic            []byte
//	ReplyTopicLength uint32
//	ReplyTopic       []byte
//	Timestamp1       uint32
//	Timestamp2       uint32
//	Timestamp3       uint32
//	Timestamp4       uint32
//	Timestamp5       uint32
//	Postamble        uint16

type MsgType int

const (
	MsgTypeUnknown MsgType = iota
	MsgTypeRequest
	MsgTypeResponse
)

type PayloadType int

const (
	PayloadTypeUnknown PayloadType = iota
	PayloadTypeMsgPack
	PayloadTypeBinary
)

// Message represents a message that can be sent or received over the connection.
type Message struct {
	Type           MsgType
	PayloadType    PayloadType
	Encrypted      bool
	Undeliverable  bool
	SequenceNumber uint32
	ControlData    uint32 // Either the subscription ID or the client ID
	Topic          string
	ReplyTopic     string
	Timestamps     []time.Time
	Payload        []byte
}

// marshal returns the wire format of the message.
func (m *Message) marshal() ([]byte, error) {
	if m.Topic == "" {
		return nil, fmt.Errorf("topic is required")
	}

	var buf bytes.Buffer
	var err error

	headerLength := header_LEN_W_TS + len(m.Topic) + len(m.ReplyTopic)

	buf.Grow(int(headerLength) + len(m.Payload))

	writeOrDie(&buf, &err, uint16(header_MARKER))
	writeOrDie(&buf, &err, uint16(header_VERSION))
	writeOrDie(&buf, &err, uint16(headerLength))
	writeOrDie(&buf, &err, m.SequenceNumber)
	writeOrDie(&buf, &err, m.flagsOut())
	writeOrDie(&buf, &err, m.ControlData)
	writeOrDie(&buf, &err, uint32(len(m.Payload)))
	writeOrDie(&buf, &err, m.Topic)
	writeOrDie(&buf, &err, m.ReplyTopic)
	writeOrDie(&buf, &err, uint32(0))
	writeOrDie(&buf, &err, uint32(0))
	writeOrDie(&buf, &err, uint32(0))
	writeOrDie(&buf, &err, uint32(0))
	writeOrDie(&buf, &err, uint32(0))
	writeOrDie(&buf, &err, uint16(header_MARKER))
	writeOrDie(&buf, &err, m.Payload)

	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// unmarshal reads a message from an io.Reader.
func unmarshal(r io.Reader) (Message, error) {
	var msg Message
	var err error
	var preamble, postamble, version, headerSize uint16
	var flags, payloadLength uint32

	readOrDie(r, &err, &preamble)
	if preamble != header_MARKER {
		return Message{}, fmt.Errorf("invalid preamble: %x", preamble)
	}
	readOrDie(r, &err, &version)
	if version != header_VERSION {
		return Message{}, fmt.Errorf("invalid version: %d", version)
	}
	readOrDie(r, &err, &headerSize)
	readOrDie(r, &err, &msg.SequenceNumber)
	readOrDie(r, &err, &flags)
	readOrDie(r, &err, &msg.ControlData)
	readOrDie(r, &err, &payloadLength)
	readOrDie(r, &err, &msg.Topic)
	readOrDie(r, &err, &msg.ReplyTopic)
	if headerSize == header_LEN_W_TS+uint16(len(msg.Topic))+uint16(len(msg.ReplyTopic)) {
		var ts [5]uint32
		readOrDie(r, &err, &ts[0])
		readOrDie(r, &err, &ts[1])
		readOrDie(r, &err, &ts[2])
		readOrDie(r, &err, &ts[3])
		readOrDie(r, &err, &ts[4])
	}
	readOrDie(r, &err, &postamble)
	if postamble != header_MARKER {
		return Message{}, fmt.Errorf("invalid postamble: %x", postamble)
	}

	if err != nil {
		return Message{}, err
	}

	msg.Payload = make([]byte, payloadLength)
	_, err = io.ReadFull(r, msg.Payload)
	if err != nil {
		return Message{}, err
	}

	msg.flagsIn(flags)

	return msg, nil
}

// flagsIn sets the message type and flags based on the flags field.
func (m *Message) flagsIn(flags uint32) {
	// Determine the message type
	switch flags & (flags_REQUEST | flags_RESPONSE) {
	case flags_REQUEST:
		m.Type = MsgTypeRequest
	case flags_RESPONSE:
		m.Type = MsgTypeResponse
	}

	// Determine the payload type
	if flags&flags_RAW_BINARY != 0 {
		m.PayloadType = PayloadTypeBinary
	}

	if flags&flags_ENCRYPTED != 0 {
		m.Encrypted = true
	}

	if flags&flags_UNDELIVERABLE != 0 {
		m.Undeliverable = true
	}
}

// flagsOut returns the flags field based on the message type and flags.
func (m *Message) flagsOut() uint32 {
	var flags uint32
	switch m.Type {
	case MsgTypeRequest:
		flags |= flags_REQUEST
	case MsgTypeResponse:
		flags |= flags_RESPONSE
	}

	if m.Encrypted {
		flags |= flags_ENCRYPTED
	}

	if m.Undeliverable {
		flags |= flags_UNDELIVERABLE
	}

	if m.PayloadType == PayloadTypeBinary {
		flags |= flags_RAW_BINARY
	}
	return flags
}

// readOrDie reads a value of type T from the reader and sets the error if any.
// If an error has already been set, the function returns immediately.
// This allows for a more concise error handling pattern.
func readOrDie[T uint16 | uint32 | string](r io.Reader, err *error, data *T) {
	if *err != nil {
		return
	}

	switch v := any(data).(type) {
	case *uint16, *uint32:
		*err = binary.Read(r, binary.BigEndian, v)
	case *string:
		var length uint32
		*err = binary.Read(r, binary.BigEndian, &length)
		if *err != nil {
			return
		}

		buf := make([]byte, length)
		_, *err = io.ReadFull(r, buf)
		if *err == nil {
			*v = string(buf)
		}
	}
}

// writeOrDie writes a value of type T to the writer and sets the error if any.
// If an error has already been set, the function returns immediately.
// This allows for a more concise error handling pattern.
func writeOrDie[T uint16 | uint32 | string | []byte](w io.Writer, err *error, data T) {
	if *err != nil {
		return
	}

	switch v := any(data).(type) {
	case uint16:
		fmt.Printf("writing uint16: %d\n", v)
		*err = binary.Write(w, binary.BigEndian, v)
	case uint32:
		fmt.Printf("writing uint32: %d\n", v)
		*err = binary.Write(w, binary.BigEndian, v)
	case string:
		fmt.Printf("writing string: %d:%s\n", len(v), v)
		*err = binary.Write(w, binary.BigEndian, uint32(len(v)))
		if *err == nil {
			_, *err = w.Write([]byte(v))
		}
	case []byte:
		fmt.Printf("writing []byte: %d\n", len(v))
		_, *err = io.Copy(w, bytes.NewReader(v))
	}
}
