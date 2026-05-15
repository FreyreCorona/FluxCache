package network

import (
	"fmt"
	"net"
	"strings"

	"github.com/FreyreCorona/FluxCache/resp"
)

type TCP struct {
	addr string
	ln   net.Listener
}

func NewTCP(addr string) *TCP {
	return &TCP{addr: addr}
}

func (t *TCP) Listen(handlers map[string]HandlerFunc, onWrite WriteFunc) error {
	ln, err := net.Listen("tcp", t.addr)
	if err != nil {
		return fmt.Errorf("tcp: listen: %w", err)
	}
	t.ln = ln

	for {
		conn, err := ln.Accept()
		if err != nil {
			return fmt.Errorf("tcp: accept: %w", err)
		}
		go t.handleConn(conn, handlers, onWrite)
	}
}

func (t *TCP) handleConn(conn net.Conn, handlers map[string]HandlerFunc, onWrite WriteFunc) {
	defer conn.Close()

	for {
		rd := resp.NewResp(conn)
		value, err := rd.Read()
		if err != nil {
			return
		}

		if value.Type != "array" || len(value.Array) == 0 {
			continue
		}

		command := strings.ToUpper(value.Array[0].Bulk)
		args := value.Array[1:]

		handler, ok := handlers[command]
		if !ok {
			continue
		}

		if onWrite != nil {
			onWrite(command, args)
		}

		result := handler(args)
		wr := resp.NewWriter(conn)
		wr.Write(result)
	}
}

func (t *TCP) Close() error {
	if t.ln != nil {
		return t.ln.Close()
	}
	return nil
}
