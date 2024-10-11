// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package rbus

import (
	"bytes"
	"fmt"

	"github.com/vmihailenco/msgpack/v5"
)

type Message struct {
	buf            bytes.Buffer
	encoder        *msgpack.Encoder
	decoder        *msgpack.Decoder
	metaDataOffset int
}

func NewMessage() *Message {
	msg := &Message{}
	msg.buf = bytes.Buffer{}
	msg.encoder = msgpack.NewEncoder(&msg.buf)
	return msg
}

func NewMessageFromBytes(buf []byte) (*Message, error) {
	msg := Message{}
	msg.buf = *bytes.NewBuffer(buf)
	msg.decoder = msgpack.NewDecoder(&msg.buf)
	return &msg, nil
}

func (m *Message) SetMetaInfo(methodName string, otParent string, otState string) {
	m.BeginMetaSectionWrite()
	m.AppendString(methodName)
	m.AppendString(otParent)
	m.AppendString(otState)
	m.EndMetaSectionWrite()
}

func (m *Message) BeginMetaSectionWrite() {
	m.metaDataOffset = len(m.buf.Bytes())
}

func (m *Message) EndMetaSectionWrite() {
	m.AppendInt32(m.metaDataOffset)
}

func (m *Message) AppendString(s string) {
	t := s + "\x00"
	m.encoder.EncodeString(t)
}

func (m *Message) PopString() (string, error) {
	s, err := m.decoder.DecodeString()
	return s, err
}

func (m *Message) PopInt32() (int, error) {
	n, err := m.decoder.DecodeInt32()
	return int(n), err
}

func (m *Message) AppendInt32(i int) {
	m.encoder.EncodeInt32(int32(i))
}

func (m *Message) Bytes() []byte {
	return m.buf.Bytes()
}

func (m *Message) PopValue() (*Value, error) {
	typeCode, err := m.PopInt32()
	if err != nil {
		return nil, err
	}

	fmt.Printf("TypeCode: %d\n", typeCode)

	switch ValueType(typeCode) {
	case Int16:
		n, err := m.PopInt32()
		if err != nil {
			return nil, err
		}
		val := NewValue(int16(n))
		return &val, nil
	}

	return nil, fmt.Errorf("unsupported type: %d", typeCode)
}
