// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package rtmessage

// CancelListenerFunc removes the listener it's associated with and cancels any
// future events sent to that listener.
//
// A CancelListenerFunc is idempotent:  after the first invocation, calling this
// closure will have no effect.
type CancelListenerFunc func()

//------------------------------------------------------------------------------

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

//------------------------------------------------------------------------------

// ReadErrorListener provides a simple way to get notified when an error occurs
// while reading from the bus.
type ReadErrorListener interface {
	OnReadError(error)
}

// ReadErrorListenerFunc is a function that implements the ReadErrorListener
// interface.  It is useful for creating a listener from a function.
type ReadErrorListenerFunc func(error)

func (f ReadErrorListenerFunc) OnReadError(err error) {
	f(err)
}
