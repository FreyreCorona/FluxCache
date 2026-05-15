package main

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/FreyreCorona/FluxCache/config"
	"github.com/FreyreCorona/FluxCache/persistence"
	"github.com/FreyreCorona/FluxCache/resp"
)

func main() {
	cfgPath := "config.yaml"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Println(err)
		return
	}

	s, p, err := config.Build(cfg)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer s.Close()
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

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	fmt.Printf("Listening on port %d\n", cfg.Server.Port)

	l, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Println(err)
		return
	}

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
