// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package rbus

type Property struct {
	Name  string
	Value Value
	next  *Property
}

func (prop *Property) Iterator() <-chan *Property {
	ch := make(chan *Property)
	go func() {
		for p := prop; p != nil; p = p.next {
			ch <- p
		}
		close(ch)
	}()
	return ch
}
