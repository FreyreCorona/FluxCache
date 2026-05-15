package network

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/FreyreCorona/FluxCache/resp"
)

type Unix struct {
	path string
	ln   net.Listener
}

func NewUnix(path string) *Unix {
	return &Unix{path: path}
}

func (u *Unix) Listen(handlers map[string]HandlerFunc, onWrite WriteFunc) error {
	os.Remove(u.path)

	ln, err := net.Listen("unix", u.path)
	if err != nil {
		return fmt.Errorf("unix: listen: %w", err)
	}
	u.ln = ln

	for {
		conn, err := ln.Accept()
		if err != nil {
			return fmt.Errorf("unix: accept: %w", err)
		}
		go u.handleConn(conn, handlers, onWrite)
	}
}

func (u *Unix) handleConn(conn net.Conn, handlers map[string]HandlerFunc, onWrite WriteFunc) {
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

func (u *Unix) Close() error {
	if u.ln != nil {
		return u.ln.Close()
	}
	return nil
}
