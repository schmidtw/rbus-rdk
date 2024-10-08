package rbus

import (
	"errors"
	"rbus/internal/rtmessage"
)

type Handle struct {
	Connection *rtmessage.Connection
}

func Open() (*Handle, error) {
	con, err := rtmessage.NewConnection(rtmessage.Config{
		URL:             "unix:///tmp/rtrouted",
		ApplicationName: "go_client",
	})

	if err != nil {
		return nil, err
	}

	err = con.Connect()
	if err != nil {
		return nil, err
	}

	return &Handle{
		Connection: con,
	}, nil
}

func (h *Handle) Get(name string) (*Value, error) {
	return nil, errors.New("not implemented")
}

func (h *Handle) Set(name string, value *Value) error {
	return errors.New("not implemented")
}

func (h *Handle) Close() {
	if h.Connection != nil {
		h.Connection.Disconnect()
		h.Connection = nil
	}
}
