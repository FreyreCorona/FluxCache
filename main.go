package main

import (
	"fmt"
	"net"
	"strings"

	"github.com/FreyreCorona/FluxCache/persistence"
	"github.com/FreyreCorona/FluxCache/resp"
	"github.com/FreyreCorona/FluxCache/store"
)

func main() {
	fmt.Println("Listening on port :6379")

	l, err := net.Listen("tcp", ":6379")
	if err != nil {
		fmt.Println(err)
		return
	}

	s := store.NewMapStore()
	p, err := persistence.NewAOF("database.aof")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer p.Close()

	p.Replay(func(cmd persistence.Command) {
		switch cmd.Name {
		case "SET":
			if len(cmd.Args) >= 2 {
				s.Set(cmd.Args[0], cmd.Args[1])
			}
		case "HSET":
			if len(cmd.Args) >= 3 {
				s.HSet(cmd.Args[0], cmd.Args[1], cmd.Args[2])
			}
		}
	})

	Handlers := NewHandlers(s)

	conn, err := l.Accept()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()

	for {
		respReader := resp.NewResp(conn)
		value, err := respReader.Read()
		if err != nil {
			fmt.Println(err)
			return
		}

		if value.Type != "array" {
			fmt.Println("Invalid request, expected array")
			continue
		}

		if len(value.Array) == 0 {
			fmt.Println("Invalid request, expected array length > 0")
			continue
		}

		command := strings.ToUpper(value.Array[0].Bulk)
		args := value.Array[1:]

		writer := resp.NewWriter(conn)

		handler, ok := Handlers[command]
		if !ok {
			fmt.Println("Invalid command: ", command)
			writer.Write(resp.Value{Type: resp.TypeString, Str: ""})
			continue
		}

		if command == "SET" || command == "HSET" {
			cmd := persistence.Command{Name: command, Args: make([]string, len(args))}
			for i, arg := range args {
				cmd.Args[i] = arg.Bulk
			}
			p.Write(cmd)
		}

		result := handler(args)
		writer.Write(result)
	}
}
