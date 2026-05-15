package network_test

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/FreyreCorona/FluxCache/network"
	"github.com/FreyreCorona/FluxCache/resp"
	"github.com/FreyreCorona/FluxCache/store"
)

type handlerMap map[string]network.HandlerFunc

func testHandlers(s store.Store) handlerMap {
	return handlerMap{
		"PING": func(args []resp.Value) resp.Value {
			return resp.Value{Type: resp.TypeString, Str: "PONG"}
		},
		"SET": func(args []resp.Value) resp.Value {
			if len(args) < 2 {
				return resp.Value{Type: resp.TypeError, Str: "ERR wrong number of arguments for 'set' command"}
			}
			s.Set(args[0].Bulk, args[1].Bulk)
			return resp.Value{Type: resp.TypeString, Str: "OK"}
		},
		"GET": func(args []resp.Value) resp.Value {
			if len(args) != 1 {
				return resp.Value{Type: resp.TypeError, Str: "ERR wrong number of arguments for 'get' command"}
			}
			val, ok := s.Get(args[0].Bulk)
			if !ok {
				return resp.Value{Type: resp.TypeNull}
			}
			return resp.Value{Type: resp.TypeBulk, Bulk: val}
		},
		"DEL": func(args []resp.Value) resp.Value {
			count := 0
			for _, arg := range args {
				_, ok := s.Get(arg.Bulk)
				if ok {
					s.Del(arg.Bulk)
					count++
				}
			}
			return resp.Value{Type: resp.TypeInteger, Num: count}
		},
	}
}

func waitForAddr(t *testing.T, n network.Network) {
	t.Helper()
	for range 100 {
		if n.Addr() != nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("server did not start listening")
}

func waitForAddrB(b *testing.B, n network.Network) {
	b.Helper()
	for range 100 {
		if n.Addr() != nil {
			return
		}
	}
	b.Fatalf("server did not start listening")
}

func setupNetwork(t *testing.T, n network.Network) {
	t.Helper()

	s := store.NewMapStore()
	handlers := testHandlers(s)
	onWrite := func(command string, args []resp.Value) {}

	go func() {
		if err := n.Listen(handlers, onWrite); err != nil {
			t.Logf("server stopped: %v", err)
		}
	}()

	waitForAddr(t, n)
}

func setupNetworkB(b *testing.B, n network.Network) network.Network {
	b.Helper()

	s := store.NewMapStore()
	handlers := testHandlers(s)
	onWrite := func(command string, args []resp.Value) {}

	go func() {
		if err := n.Listen(handlers, onWrite); err != nil {
			b.Logf("server stopped: %v", err)
		}
	}()

	waitForAddrB(b, n)
	return n
}

func respDial(addr string) (net.Conn, error) {
	return net.DialTimeout("tcp", addr, 5*time.Second)
}

func respCommand(conn net.Conn, args ...string) error {
	v := resp.Value{Type: resp.TypeArray, Array: make([]resp.Value, len(args))}
	for i, a := range args {
		v.Array[i] = resp.Value{Type: resp.TypeBulk, Bulk: a}
	}
	_, err := conn.Write(v.Marshal())
	return err
}

func respRead(conn net.Conn) (resp.Value, error) {
	rd := resp.NewResp(conn)
	return rd.Read()
}

func assertPONG(t *testing.T, v resp.Value) {
	t.Helper()
	if v.Type != resp.TypeString || v.Str != "PONG" {
		t.Fatalf("expected PONG, got %+v", v)
	}
}

func assertOK(t *testing.T, v resp.Value) {
	t.Helper()
	if v.Type != resp.TypeString || v.Str != "OK" {
		t.Fatalf("expected OK, got %+v", v)
	}
}

func assertBulk(t *testing.T, v resp.Value, expected string) {
	t.Helper()
	if v.Type != resp.TypeBulk || v.Bulk != expected {
		t.Fatalf("expected %q, got %+v", expected, v)
	}
}

func assertNull(t *testing.T, v resp.Value) {
	t.Helper()
	if v.Type != resp.TypeNull {
		t.Fatalf("expected null, got %+v", v)
	}
}

func assertInt(t *testing.T, v resp.Value, expected int) {
	t.Helper()
	if v.Type != resp.TypeInteger || v.Num != expected {
		t.Fatalf("expected %d, got %+v", expected, v)
	}
}

func mustResp(c net.Conn, cmd ...string) resp.Value {
	if err := respCommand(c, cmd...); err != nil {
		panic(fmt.Sprintf("resp command %v: %v", cmd, err))
	}
	v, err := respRead(c)
	if err != nil {
		panic(fmt.Sprintf("resp read: %v", err))
	}
	return v
}

func mustRespErr(c net.Conn, cmd ...string) {
	if err := respCommand(c, cmd...); err != nil {
		panic(fmt.Sprintf("resp command %v: %v", cmd, err))
	}
}
