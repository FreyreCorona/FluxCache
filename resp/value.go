package resp

import "strconv"

const (
	STRING  = '+'
	ERROR   = '-'
	INTEGER = ':'
	BULK    = '$'
	ARRAY   = '*'
)

const (
	TypeArray  = "array"
	TypeBulk   = "bulk"
	TypeString = "string"
	TypeNull   = "null"
	TypeError  = "error"
)

type Value struct {
	Type  string
	Str   string
	Num   int
	Bulk  string
	Array []Value
}

func (v Value) Marshal() []byte {
	switch v.Type {
	case TypeArray:
		return v.marshalArray()
	case TypeBulk:
		return v.marshalBulk()
	case TypeString:
		return v.marshalString()
	case TypeNull:
		return v.marshallNull()
	case TypeError:
		return v.marshallError()
	default:
		return []byte{}
	}
}

func (v Value) marshalString() []byte {
	var bytes []byte
	bytes = append(bytes, STRING)
	bytes = append(bytes, v.Str...)
	bytes = append(bytes, '\r', '\n')
	return bytes
}

func (v Value) marshalBulk() []byte {
	var bytes []byte
	bytes = append(bytes, BULK)
	bytes = append(bytes, strconv.Itoa(len(v.Bulk))...)
	bytes = append(bytes, '\r', '\n')
	bytes = append(bytes, v.Bulk...)
	bytes = append(bytes, '\r', '\n')
	return bytes
}

func (v Value) marshalArray() []byte {
	var bytes []byte
	bytes = append(bytes, ARRAY)
	bytes = append(bytes, strconv.Itoa(len(v.Array))...)
	bytes = append(bytes, '\r', '\n')
	for i := range v.Array {
		bytes = append(bytes, v.Array[i].Marshal()...)
	}
	return bytes
}

func (v Value) marshallError() []byte {
	var bytes []byte
	bytes = append(bytes, ERROR)
	bytes = append(bytes, v.Str...)
	bytes = append(bytes, '\r', '\n')
	return bytes
}

func (v Value) marshallNull() []byte {
	return []byte("$-1\r\n")
}
