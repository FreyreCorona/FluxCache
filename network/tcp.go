package network

import (
	"fmt"
	"net"
	"strings"

	"github.com/FreyreCorona/FluxCache/resp"
)

type TCP struct {
	addr     string
	maxConns int
	sem      chan struct{}
	ln       net.Listener
}

func NewTCP(addr string, maxConns int) *TCP {
	return &TCP{addr: addr, maxConns: maxConns}
}

func (t *TCP) Listen(handlers map[string]HandlerFunc, onWrite WriteFunc) error {
	ln, err := net.Listen("tcp", t.addr)
	if err != nil {
		return fmt.Errorf("tcp: listen: %w", err)
	}
	t.ln = ln

	if t.maxConns > 0 {
		t.sem = make(chan struct{}, t.maxConns)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			return fmt.Errorf("tcp: accept: %w", err)
		}

		if t.sem != nil {
			select {
			case t.sem <- struct{}{}:
			default:
				conn.Close()
				continue
			}
		}

		go t.handleConn(conn, handlers, onWrite)
	}
}

func (t *TCP) handleConn(conn net.Conn, handlers map[string]HandlerFunc, onWrite WriteFunc) {
	defer conn.Close()
	if t.sem != nil {
		defer func() { <-t.sem }()
	}

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

func (t *TCP) Addr() net.Addr {
	if t.ln != nil {
		return t.ln.Addr()
	}
	return nil
}

func (t *TCP) Close() error {
	if t.ln != nil {
		return t.ln.Close()
	}
	return nil
}
