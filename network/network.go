package network

import (
	"net"

	"github.com/FreyreCorona/FluxCache/resp"
)

// HandlerFunc is a function that handles a command and returns a response.
type HandlerFunc func(args []resp.Value) resp.Value

// WriteFunc is a function that is called when a write command is executed.
type WriteFunc func(command string, args []resp.Value)

// Network defines the interface for network transport backends.
type Network interface {
	Listen(handlers map[string]HandlerFunc, onWrite WriteFunc) error
	Addr() net.Addr
	Close() error
}
