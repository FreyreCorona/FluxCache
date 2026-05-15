package main

import (
	"github.com/FreyreCorona/FluxCache/resp"
	"github.com/FreyreCorona/FluxCache/store"
)

func NewHandlers(s store.Store) map[string]func([]resp.Value) resp.Value {
	return map[string]func([]resp.Value) resp.Value{
		"PING":    ping,
		"SET":     setHandler(s),
		"GET":     getHandler(s),
		"HSET":    hsetHandler(s),
		"HGET":    hgetHandler(s),
		"HGETALL": hgetallHandler(s),
	}
}

func ping(args []resp.Value) resp.Value {
	if len(args) == 0 {
		return resp.Value{Type: resp.TypeString, Str: "PONG"}
	}
	return resp.Value{Type: resp.TypeString, Str: args[0].Bulk}
}

func setHandler(s store.Store) func([]resp.Value) resp.Value {
	return func(args []resp.Value) resp.Value {
		if len(args) != 2 {
			return resp.Value{Type: resp.TypeError, Str: "ERR wrong number of arguments for 'set' command"}
		}
		s.Set(args[0].Bulk, args[1].Bulk)
		return resp.Value{Type: resp.TypeString, Str: "OK"}
	}
}

func getHandler(s store.Store) func([]resp.Value) resp.Value {
	return func(args []resp.Value) resp.Value {
		if len(args) != 1 {
			return resp.Value{Type: resp.TypeError, Str: "ERR wrong number of arguments for 'get' command"}
		}
		val, ok := s.Get(args[0].Bulk)
		if !ok {
			return resp.Value{Type: resp.TypeNull}
		}
		return resp.Value{Type: resp.TypeBulk, Bulk: val}
	}
}

func hsetHandler(s store.Store) func([]resp.Value) resp.Value {
	return func(args []resp.Value) resp.Value {
		if len(args) != 3 {
			return resp.Value{Type: resp.TypeError, Str: "ERR wrong number of arguments for 'hset' command"}
		}
		s.HSet(args[0].Bulk, args[1].Bulk, args[2].Bulk)
		return resp.Value{Type: resp.TypeString, Str: "OK"}
	}
}

func hgetHandler(s store.Store) func([]resp.Value) resp.Value {
	return func(args []resp.Value) resp.Value {
		if len(args) != 2 {
			return resp.Value{Type: resp.TypeError, Str: "ERR wrong number of arguments for 'hget' command"}
		}
		val, ok := s.HGet(args[0].Bulk, args[1].Bulk)
		if !ok {
			return resp.Value{Type: resp.TypeNull}
		}
		return resp.Value{Type: resp.TypeBulk, Bulk: val}
	}
}

func hgetallHandler(s store.Store) func([]resp.Value) resp.Value {
	return func(args []resp.Value) resp.Value {
		if len(args) != 1 {
			return resp.Value{Type: resp.TypeError, Str: "ERR wrong number of arguments for 'hgetall' command"}
		}
		h := s.HGetAll(args[0].Bulk)
		if h == nil {
			return resp.Value{Type: resp.TypeArray, Array: []resp.Value{}}
		}
		values := make([]resp.Value, 0, len(h)*2)
		for k, v := range h {
			values = append(values, resp.Value{Type: resp.TypeBulk, Bulk: k})
			values = append(values, resp.Value{Type: resp.TypeBulk, Bulk: v})
		}
		return resp.Value{Type: resp.TypeArray, Array: values}
	}
}
