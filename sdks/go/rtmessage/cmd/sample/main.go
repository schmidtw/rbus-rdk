// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

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

	con, err := rtmessage.New(tcpURL, appName, 123,
		rtmessage.WithReadTimeout(5*time.Second),
		rtmessage.WithWriteTimeout(5*time.Second),
		rtmessage.WithErrorListener(rtmessage.ReadErrorListenerFunc(func(err error) {
			fmt.Printf("Read error: %s\n", err.Error())
		})),
		rtmessage.WithMessageListener(rtmessage.MessageListenerFunc(func(msg rtmessage.Message) {
			fmt.Printf("Received message: %s\n", string(msg.Payload))
		})),
		rtmessage.WithSubscription("A.B.C"),
	)
	if err != nil {
		panic(fmt.Sprintf("Failed to create rbus handle. %s", err.Error()))
	}

	fmt.Println("Object created.")

	if err := con.Connect(); err != nil {
		panic(fmt.Sprintf("Failed to connect. %s", err.Error()))
	}

	fmt.Println("Connected.")

	fmt.Println("Listening for messages.")

	for {
		time.Sleep(1 * time.Second)
	}
}
