package main

import (
	"fmt"
	"rbus/internal/rtmessage"
	"time"
)

func main() {
	url := "unix:///tmp/rtrouted"
	appName := "my_go_app"

	con, err := rtmessage.NewConnection(url, appName)
	if err != nil {
		panic(fmt.Sprintf("Failed to create connection. %s", err.Error()))
	}

	if err := con.Connect(); err != nil {
		panic(fmt.Sprintf("Failed to connect. %s", err.Error()))
	}

	con.Add(rtmessage.MessageListenerFunc(func(msg rtmessage.Message) {
		fmt.Printf("Received message: %s\n", string(msg.Payload))
	}), "A.B.C")

	// This is somewhat weird.
	for {
		time.Sleep(1 * time.Second)
	}
}
