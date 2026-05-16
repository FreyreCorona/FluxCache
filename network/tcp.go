package network

import (
	"fmt"
	"net"
	"strings"

	"github.com/FreyreCorona/FluxCache/resp"
	"github.com/FreyreCorona/FluxCache/telemetry"
)

// TCP is a plain TCP network transport.
type TCP struct {
	addr     string
	maxConns int
	password string
	sem      chan struct{}
	ln       net.Listener
}

// NewTCP creates a new TCP transport that listens on the given address.
func NewTCP(addr string, maxConns int, password string) *TCP {
	return &TCP{addr: addr, maxConns: maxConns, password: password}
}

// Listen starts the TCP listener and begins accepting connections.
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
	telemetry.ActiveConnections.Inc()
	defer telemetry.ActiveConnections.Dec()
	defer conn.Close()
	if t.sem != nil {
		defer func() { <-t.sem }()
	}

	authenticated := t.password == ""
	wr := resp.NewWriter(conn)

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

		if !authenticated {
			if command == "AUTH" {
				if len(args) >= 1 && args[0].Bulk == t.password {
					authenticated = true
					wr.Write(resp.Value{Type: resp.TypeString, Str: "OK"})
				} else {
					wr.Write(resp.Value{Type: resp.TypeError, Str: "ERR invalid password"})
				}
				continue
			}
			wr.Write(resp.Value{Type: resp.TypeError, Str: "NOAUTH Authentication required."})
			continue
		}

		handler, ok := handlers[command]
		if !ok {
			continue
		}

		if onWrite != nil {
			onWrite(command, args)
		}

		result := handler(args)
		wr.Write(result)
	}
}

// Addr returns the address the listener is bound to.
func (t *TCP) Addr() net.Addr {
	if t.ln != nil {
		return t.ln.Addr()
	}
	return nil
}

// Close stops the TCP listener.
func (t *TCP) Close() error {
	if t.ln != nil {
		return t.ln.Close()
	}
	return nil
}
