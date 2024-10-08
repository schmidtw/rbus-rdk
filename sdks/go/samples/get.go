package main

import (
	"fmt"
	"rbus"
)

func main() {
	rbusHandle, err := rbus.Open()
	if err != nil {
		panic(fmt.Sprintf("Failed to open rbus: %v", err))
	}
	defer rbusHandle.Close()

	value1, err := rbusHandle.Get("Device.Foo")
	if err != nil {
		panic(fmt.Sprintf("Failed to get value: %v", err))
	}

	value2 := rbus.NewValue[uint64](42)
	rbusHandle.Set("Device.Foo", &value2)

	fmt.Printf("Value: %s\n", value1)
}
