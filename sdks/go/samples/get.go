package main

import (
	"fmt"
	"rbus"
)

func main() {
	h, err := rbus.New(
		rbus.WithURL("unix://file/default"),
		rbus.WithApplicationName("foo"),
	)
	if err != nil {
		panic(err)
	}

	err = h.Open()
	if err != nil {
		panic(fmt.Sprintf("Failed to open rbus: %v", err))
	}
	defer h.Close()

	value1, err := h.Get("Device.Foo")
	if err != nil {
		panic(fmt.Sprintf("Failed to get value: %v", err))
	}

	value2 := rbus.NewValue[uint64](42)
	h.Set("Device.Foo", &value2)

	fmt.Printf("Value: %s\n", value1)
}
