package rbus

import "fmt"

type ValueType int

const (
	Boolean ValueType = 0x500 + iota
	Character
	Byte
	Int8
	UInt8
	Int16
	UInt16
	Int32
	UInt32
	Int64
	UInt64
	Single
	Double
	DateTime
	String
	Bytes
	_Property
	Object
	None
)

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
	case Variant[int16]:
		return fmt.Sprintf("%d", v.unwrap)
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

func (t ValueType) String() string {
	return [...]string{
		"Boolean",
		"Char",
		"Byte",
		"Int8",
		"UInt8",
		"Int16",
		"UInt16",
		"Int32",
		"UInt32",
		"Int64",
		"UInt64",
		"Stringle",
		"Double",
		"DateTime",
		"String",
		"Bytes",
		"Property",
		"Object",
		"None",
	}[t-0x500]
}
