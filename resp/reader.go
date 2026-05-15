package resp

import (
	"bufio"
	"io"
	"strconv"
)

// Resp reads and parses RESP protocol data from an io.Reader.
type Resp struct {
	reader *bufio.Reader
}

// NewResp creates a new Resp that reads from the given io.Reader.
func NewResp(rd io.Reader) *Resp {
	return &Resp{reader: bufio.NewReader(rd)}
}

func (r *Resp) readLine() (line []byte, n int, err error) {
	for {
		b, err := r.reader.ReadByte()
		if err != nil {
			return nil, 0, err
		}
		n += 1
		line = append(line, b)
		if len(line) >= 2 && line[len(line)-2] == '\r' {
			break
		}
	}
	return line[:len(line)-2], n, nil
}

func (r *Resp) readInteger() (x int, n int, err error) {
	line, n, err := r.readLine()
	if err != nil {
		return 0, 0, err
	}
	i64, err := strconv.ParseInt(string(line), 10, 64)
	if err != nil {
		return 0, n, err
	}
	return int(i64), n, nil
}

// Read parses the next RESP value from the underlying reader.
func (r *Resp) Read() (Value, error) {
	_type, err := r.reader.ReadByte()
	if err != nil {
		return Value{}, err
	}

	switch _type {
	case ARRAY:
		return r.readArray()
	case BULK:
		return r.readBulk()
	case STRING:
		return r.readSimpleString()
	case ERROR:
		return r.readError()
	case INTEGER:
		return r.readIntegerValue()
	default:
		return Value{}, nil
	}
}

func (r *Resp) readArray() (Value, error) {
	v := Value{}
	v.Type = TypeArray

	len, _, err := r.readInteger()
	if err != nil {
		return v, err
	}

	v.Array = make([]Value, 0)
	for range len {
		val, err := r.Read()
		if err != nil {
			return v, err
		}
		v.Array = append(v.Array, val)
	}

	return v, nil
}

func (r *Resp) readBulk() (Value, error) {
	v := Value{}
	v.Type = TypeBulk

	len, _, err := r.readInteger()
	if err != nil {
		return v, err
	}

	if len == -1 {
		v.Type = TypeNull
		return v, nil
	}

	bulk := make([]byte, len)
	r.reader.Read(bulk)
	v.Bulk = string(bulk)

	r.readLine()

	return v, nil
}

func (r *Resp) readSimpleString() (Value, error) {
	line, _, err := r.readLine()
	if err != nil {
		return Value{}, err
	}
	return Value{Type: TypeString, Str: string(line)}, nil
}

func (r *Resp) readError() (Value, error) {
	line, _, err := r.readLine()
	if err != nil {
		return Value{}, err
	}
	return Value{Type: TypeError, Str: string(line)}, nil
}

func (r *Resp) readIntegerValue() (Value, error) {
	n, _, err := r.readInteger()
	if err != nil {
		return Value{}, err
	}
	return Value{Type: TypeInteger, Num: n}, nil
}
