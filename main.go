package main

import (
	"fmt"
	"os"

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

	n, err := config.BuildNetwork(cfg.Server)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer n.Close()

	handlers := NewHandlers(s)

	fmt.Printf("Listening on port %d\n", cfg.Server.Port)
	if err := n.Listen(handlers, func(command string, args []resp.Value) {
		if command == "SET" || command == "HSET" {
			cmd := persistence.Command{Name: command, Args: make([]string, len(args))}
			for i, arg := range args {
				cmd.Args[i] = arg.Bulk
			}
			p.Write(cmd)
		}
	}); err != nil {
		fmt.Println(err)
	}
}
