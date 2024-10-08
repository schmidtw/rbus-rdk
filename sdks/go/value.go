package rbus

import "fmt"

type ValueConstraint interface {
	int | bool | string | int8 | int16 | int32 | int64 | uint8 | uint16 | uint32 | uint64
}

type ValueVariant interface {
	isVariant()
}

type Value struct {
	Value ValueVariant
}

type Variant[T ValueConstraint] struct {
	unwrap T
}

func (v Variant[T]) isVariant() {}

func NewValue[T ValueConstraint](v T) Value {
	return Value{Variant[T]{v}}
}

func (val Value) String() string {
	switch v := val.Value.(type) {
	case Variant[int]:
		return fmt.Sprintf("%d", v.unwrap)
	case Variant[int64]:
		return fmt.Sprintf("%d", v.unwrap)
	case Variant[bool]:
		return fmt.Sprintf("%t", v.unwrap)
	case Variant[string]:
		return v.unwrap
	default:
		panic(fmt.Errorf("unsupported type: %T", v))
	}
}
