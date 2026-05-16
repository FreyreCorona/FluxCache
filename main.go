package main

import (
	"context"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/FreyreCorona/FluxCache/config"
	"github.com/FreyreCorona/FluxCache/handler"
	"github.com/FreyreCorona/FluxCache/health"
	"github.com/FreyreCorona/FluxCache/log"
	"github.com/FreyreCorona/FluxCache/persistence"
	"github.com/FreyreCorona/FluxCache/resp"
)

// version is set at build time via -ldflags=-X main.version=<tag>
var version string

// main loads the config, builds the store/persistence/network, and starts the server with graceful shutdown.
func main() {
	cfgPath := "config.yaml"
	if v, ok := os.LookupEnv("FLUXCACHE_CONFIG"); ok {
		cfgPath = v
	}
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Error("config load failed", "error", err)
		os.Exit(1)
	}

	if cfg.Server.MaxMemory != "" {
		limit, _ := cfg.MaxMemoryBytes()
		debug.SetMemoryLimit(limit)
	}

	s, p, err := config.Build(cfg)
	if err != nil {
		log.Error("config build failed", "error", err)
		os.Exit(1)
	}

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

	handlers := handler.NewHandlers(s)

	n, err := config.BuildNetwork(cfg.Server)
	if err != nil {
		log.Error("network build failed", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	if cfg.Server.HealthPort > 0 {
		go health.StartServer(cfg.Server.HealthPort, ctx)
	}

	log.Info("listening", "port", cfg.Server.Port)

	errCh := make(chan error, 1)
	go func() {
		errCh <- n.Listen(handlers, func(command string, args []resp.Value) {
			if command == "SET" || command == "HSET" {
				cmd := persistence.Command{Name: command, Args: make([]string, len(args))}
				for i, arg := range args {
					cmd.Args[i] = arg.Bulk
				}
				p.Write(cmd)
			}
		})
	}()

	select {
	case <-ctx.Done():
		log.Info("shutting down")
	case err := <-errCh:
		if err != nil {
			log.Error("server error", "error", err)
		}
	}

	n.Close()
	<-errCh
	p.Close()
	s.Close()
	log.Info("shutdown complete")
}
