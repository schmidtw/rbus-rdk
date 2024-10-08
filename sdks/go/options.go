// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package rbus

import (
	"errors"
	"os"
)

// Option interface for setting configuration options
type Option interface {
	apply(*config) error
}

// optionFunc wraps a function that modifies Config into an
// implementation of the Options interface.
type optionFunc func(*config) error

func (f optionFunc) apply(cfg *config) error {
	return f(cfg)
}

// WithURL sets the URL for the rbus connection
func WithURL(url string) Option {
	return optionFunc(func(cfg *config) error {
		cfg.url = url
		return nil
	})
}

// WithApplicationName sets the application name for the rbus connection
func WithApplicationName(name string) Option {
	return optionFunc(func(cfg *config) error {
		cfg.appName = name
		return nil
	})
}

func WithInboxID(id int) Option {
	return optionFunc(func(cfg *config) error {
		cfg.id = id
		return nil
	})
}

func WithInboxAsPID() Option {
	return WithInboxID(os.Getpid())
}

// -------- Below are options that validate the configuration --------

// assertURL validates the URL
func assertURL() Option {
	return optionFunc(func(cfg *config) error {
		if cfg.url == "" {
			return errors.New("URL is required")
		}
		return nil
	})
}

// assertApplicationName validates the application name
func assertApplicationName() Option {
	return optionFunc(func(cfg *config) error {
		if cfg.appName == "" {
			return errors.New("Application name is required")
		}
		return nil
	})
}
