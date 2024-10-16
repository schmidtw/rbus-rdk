package main

import (
	"fmt"
	"time"

	"github.com/schmidtw/rbus-rdk/sdks/go/rbus"
)

const (
	tcpURL  = "tcp://127.0.0.1:10001"
	unixURL = "unix:///tmp/rtrouted"
)

func main() {
	h, err := rbus.New(rbus.WithURL(unixURL), rbus.WithApplicationName("go_app"))
	if err != nil {
		panic(fmt.Sprintf("Failed to create rbus handle. %s", err.Error()))
	}

	if err := h.Open(); err != nil {
		panic(fmt.Sprintf("Failed to open rbus connection. %s", err.Error()))
	}

	v, err := h.Get("Device.SampleProvider.AllTypes.Int16Data")
	if err != nil {
		panic(fmt.Sprintf("Failed to get value. %s", err.Error()))
	}

	fmt.Printf("Value: %v\n", v)

	for {
		time.Sleep(1 * time.Second)
	}
}
