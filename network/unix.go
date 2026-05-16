package network

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/FreyreCorona/FluxCache/resp"
	"github.com/FreyreCorona/FluxCache/telemetry"
)

// Unix is a Unix domain socket network transport.
type Unix struct {
	path     string
	maxConns int
	sem      chan struct{}
	ln       net.Listener
}

// NewUnix creates a new Unix domain socket transport at the given path.
func NewUnix(path string, maxConns int) *Unix {
	return &Unix{path: path, maxConns: maxConns}
}

// Listen starts the Unix socket listener and begins accepting connections.
func (u *Unix) Listen(handlers map[string]HandlerFunc, onWrite WriteFunc) error {
	os.Remove(u.path)

	ln, err := net.Listen("unix", u.path)
	if err != nil {
		return fmt.Errorf("unix: listen: %w", err)
	}
	u.ln = ln

	if u.maxConns > 0 {
		u.sem = make(chan struct{}, u.maxConns)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			return fmt.Errorf("unix: accept: %w", err)
		}

		if u.sem != nil {
			select {
			case u.sem <- struct{}{}:
			default:
				conn.Close()
				continue
			}
		}

		go u.handleConn(conn, handlers, onWrite)
	}
}

func (u *Unix) handleConn(conn net.Conn, handlers map[string]HandlerFunc, onWrite WriteFunc) {
	telemetry.ActiveConnections.Inc()
	defer telemetry.ActiveConnections.Dec()
	defer conn.Close()
	if u.sem != nil {
		defer func() { <-u.sem }()
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

// Addr returns the address the listener is bound to.
func (u *Unix) Addr() net.Addr {
	if u.ln != nil {
		return u.ln.Addr()
	}
	return nil
}

// Close stops the Unix socket listener.
func (u *Unix) Close() error {
	if u.ln != nil {
		return u.ln.Close()
	}
	return nil
}
