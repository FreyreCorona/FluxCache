package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/FreyreCorona/FluxCache/config"
	"github.com/FreyreCorona/FluxCache/handler"
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
		os.Exit(1)
	}

	if cfg.Server.MaxMemory != "" {
		limit, _ := cfg.MaxMemoryBytes()
		debug.SetMemoryLimit(limit)
	}

	s, p, err := config.Build(cfg)
	if err != nil {
		fmt.Println(err)
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
		fmt.Println(err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	if cfg.Server.HealthPort > 0 {
		go startHealthServer(cfg.Server.HealthPort, ctx)
	}

	fmt.Printf("Listening on port %d\n", cfg.Server.Port)

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
		fmt.Println("\nshutting down...")
	case err := <-errCh:
		if err != nil {
			fmt.Println(err)
		}
	}

	n.Close()
	<-errCh
	p.Close()
	s.Close()
	fmt.Println("done")
}

func startHealthServer(port int, ctx context.Context) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
	})

	srv := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: mux}

	go func() {
		<-ctx.Done()
		srv.Close()
	}()

	fmt.Printf("Health endpoint on :%d\n", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Printf("health: %v\n", err)
	}
}
