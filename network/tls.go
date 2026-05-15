package network

import (
	"crypto/tls"
	"fmt"
	"net"
	"strings"

	"github.com/FreyreCorona/FluxCache/resp"
)

// TLS is a TLS-encrypted TCP network transport.
type TLS struct {
	addr     string
	cert     string
	key      string
	config   *tls.Config
	maxConns int
	sem      chan struct{}
	ln       net.Listener
}

// NewTLS creates a new TLS transport with the given certificate and key files.
func NewTLS(addr, certFile, keyFile string, maxConns int) *TLS {
	return &TLS{addr: addr, cert: certFile, key: keyFile, maxConns: maxConns}
}

// Listen starts the TLS listener and begins accepting connections.
func (t *TLS) Listen(handlers map[string]HandlerFunc, onWrite WriteFunc) error {
	cert, err := tls.LoadX509KeyPair(t.cert, t.key)
	if err != nil {
		return fmt.Errorf("tls: load cert: %w", err)
	}
	t.config = &tls.Config{Certificates: []tls.Certificate{cert}}

	ln, err := tls.Listen("tcp", t.addr, t.config)
	if err != nil {
		return fmt.Errorf("tls: listen: %w", err)
	}
	t.ln = ln

	if t.maxConns > 0 {
		t.sem = make(chan struct{}, t.maxConns)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			return fmt.Errorf("tls: accept: %w", err)
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

func (t *TLS) handleConn(conn net.Conn, handlers map[string]HandlerFunc, onWrite WriteFunc) {
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

// Addr returns the address the listener is bound to.
func (t *TLS) Addr() net.Addr {
	if t.ln != nil {
		return t.ln.Addr()
	}
	return nil
}

// Close stops the TLS listener.
func (t *TLS) Close() error {
	if t.ln != nil {
		return t.ln.Close()
	}
	return nil
}
