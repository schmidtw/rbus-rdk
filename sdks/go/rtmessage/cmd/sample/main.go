package main

import (
	"fmt"
	"time"

	"github.com/schmidtw/rbus-rdk/sdks/go/rbus/rtmessage"
)

const (
	tcpURL  = "tcp://127.0.0.1:10001"
	unixURL = "unix:///tmp/rtrouted"
)

func main() {
	appName := "my_go_app"

	con, err := rtmessage.New(tcpURL, appName)
	if err != nil {
		panic(fmt.Sprintf("Failed to create connection. %s", err.Error()))
	}

	if err := con.Connect(); err != nil {
		panic(fmt.Sprintf("Failed to connect. %s", err.Error()))
	}

	con.Add(rtmessage.MessageListenerFunc(func(msg rtmessage.Message) {
		fmt.Printf("Received message: %s\n", string(msg.Payload))
	}), "A.B.C")

	// Only run for a minute, then exit.
	time.Sleep(1 * time.Minute)
}
