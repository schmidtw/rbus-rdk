// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package rtmessage

import (
	"fmt"
	"net/url"
	"time"
)

// Option interface for setting configuration options
type Option interface {
	apply(*Connection) error
}

// optionFunc wraps a function that modifies Config into an
// implementation of the Options interface.
type optionFunc func(*Connection) error

func (f optionFunc) apply(c *Connection) error {
	return f(c)
}

// WithReadTimeout sets the read timeout for the connection.
func WithReadTimeout(timeout time.Duration) Option {
	return optionFunc(func(c *Connection) error {
		c.readTimeout = timeout
		return nil
	})
}

// WithWriteTimeout sets the write timeout for the connection.
func WithWriteTimeout(timeout time.Duration) Option {
	return optionFunc(func(c *Connection) error {
		c.writeTimeout = timeout
		return nil
	})
}

// WithErrorListener adds a listener for read errors.  Takes an optional cancel
// function pointer that can be used to remove the listener.
func WithErrorListener(listener ReadErrorListener, cancel ...*CancelListenerFunc) Option {
	return optionFunc(func(c *Connection) error {
		tmp := c.errListeners.Add(listener)
		if len(cancel) > 0 {
			*cancel[0] = tmp
		}
		return nil
	})
}

// WithMessageListener adds a listener for messages.  Takes an optional cancel
// function pointer that can be used to remove the listener.
func WithMessageListener(listener MessageListener, cancel ...*CancelListenerFunc) Option {
	return optionFunc(func(c *Connection) error {
		tmp := c.msgListeners.Add(listener)
		if len(cancel) > 0 {
			*cancel[0] = tmp
		}
		return nil
	})
}

// WithSubscription adds a subscription to the connection.
func WithSubscription(subscription string) Option {
	return optionFunc(func(c *Connection) error {
		c.subscriptions[subscription] = struct{}{}
		return nil
	})
}

// WithSubscriptions adds subscriptions to the connection.
func WithSubscriptions(subscriptions ...string) Option {
	return optionFunc(func(c *Connection) error {
		for _, s := range subscriptions {
			c.subscriptions[s] = struct{}{}
		}
		return nil
	})
}

// TODO Add WithAutoReconnect() Option

// withRawURL validates the URL
func withRawURL(rawURL string) Option {
	return optionFunc(func(c *Connection) error {
		u, err := url.Parse(rawURL)
		if err != nil {
			return err
		}

		switch u.Scheme {
		case "unix", "tcp":
		default:
			return fmt.Errorf("%w: unsupported URL scheme", ErrInvalidInput)
		}

		c.url = u
		return nil
	})
}
