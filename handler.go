package main

import (
	"strconv"
	"time"

	"github.com/FreyreCorona/FluxCache/resp"
	"github.com/FreyreCorona/FluxCache/store"
)

func NewHandlers(s *store.TTLStore) map[string]func([]resp.Value) resp.Value {
	return map[string]func([]resp.Value) resp.Value{
		"PING":    ping,
		"SET":     setHandler(s),
		"GET":     getHandler(s),
		"HSET":    hsetHandler(s),
		"HGET":    hgetHandler(s),
		"HGETALL": hgetallHandler(s),
		"EXPIRE":  expireHandler(s),
		"TTL":     ttlHandler(s),
		"DEL":     delHandler(s),
	}
}

func ping(args []resp.Value) resp.Value {
	if len(args) == 0 {
		return resp.Value{Type: resp.TypeString, Str: "PONG"}
	}
	return resp.Value{Type: resp.TypeString, Str: args[0].Bulk}
}

func setHandler(s *store.TTLStore) func([]resp.Value) resp.Value {
	return func(args []resp.Value) resp.Value {
		if len(args) < 2 {
			return resp.Value{Type: resp.TypeError, Str: "ERR wrong number of arguments for 'set' command"}
		}
		key := args[0].Bulk
		val := args[1].Bulk

		if len(args) >= 4 {
			flag := args[2].Bulk
			if flag == "EX" || flag == "ex" {
				sec, err := strconv.Atoi(args[3].Bulk)
				if err == nil {
					s.SetWithTTL(key, val, time.Duration(sec)*time.Second)
					return resp.Value{Type: resp.TypeString, Str: "OK"}
				}
			}
		}

		s.Set(key, val)
		return resp.Value{Type: resp.TypeString, Str: "OK"}
	}
}

func getHandler(s *store.TTLStore) func([]resp.Value) resp.Value {
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

func hsetHandler(s *store.TTLStore) func([]resp.Value) resp.Value {
	return func(args []resp.Value) resp.Value {
		if len(args) != 3 {
			return resp.Value{Type: resp.TypeError, Str: "ERR wrong number of arguments for 'hset' command"}
		}
		s.HSet(args[0].Bulk, args[1].Bulk, args[2].Bulk)
		return resp.Value{Type: resp.TypeString, Str: "OK"}
	}
}

func hgetHandler(s *store.TTLStore) func([]resp.Value) resp.Value {
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

func hgetallHandler(s *store.TTLStore) func([]resp.Value) resp.Value {
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

func expireHandler(s *store.TTLStore) func([]resp.Value) resp.Value {
	return func(args []resp.Value) resp.Value {
		if len(args) != 2 {
			return resp.Value{Type: resp.TypeError, Str: "ERR wrong number of arguments for 'expire' command"}
		}
		sec, err := strconv.Atoi(args[1].Bulk)
		if err != nil {
			return resp.Value{Type: resp.TypeError, Str: "ERR value is not an integer or out of range"}
		}
		ok := s.Expire(args[0].Bulk, time.Duration(sec)*time.Second)
		if ok {
			return resp.Value{Type: resp.TypeInteger, Num: 1}
		}
		return resp.Value{Type: resp.TypeInteger, Num: 0}
	}
}

func ttlHandler(s *store.TTLStore) func([]resp.Value) resp.Value {
	return func(args []resp.Value) resp.Value {
		if len(args) != 1 {
			return resp.Value{Type: resp.TypeError, Str: "ERR wrong number of arguments for 'ttl' command"}
		}
		d := s.TTL(args[0].Bulk)
		return resp.Value{Type: resp.TypeInteger, Num: int(d.Seconds())}
	}
}

func delHandler(s *store.TTLStore) func([]resp.Value) resp.Value {
	return func(args []resp.Value) resp.Value {
		if len(args) < 1 {
			return resp.Value{Type: resp.TypeError, Str: "ERR wrong number of arguments for 'del' command"}
		}
		count := 0
		for _, arg := range args {
			_, ok := s.Get(arg.Bulk)
			if ok {
				s.Del(arg.Bulk)
				count++
			}
		}
		return resp.Value{Type: resp.TypeInteger, Num: count}
	}
}
