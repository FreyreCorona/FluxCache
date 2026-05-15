package network

import (
	"net"

	"github.com/FreyreCorona/FluxCache/resp"
)

type HandlerFunc func(args []resp.Value) resp.Value

type WriteFunc func(command string, args []resp.Value)

type Network interface {
	Listen(handlers map[string]HandlerFunc, onWrite WriteFunc) error
	Addr() net.Addr
	Close() error
}
