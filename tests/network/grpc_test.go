package network_test

import (
	"testing"

	"github.com/FreyreCorona/FluxCache/network"
	"github.com/FreyreCorona/FluxCache/network/grpcpb"
	"github.com/FreyreCorona/FluxCache/resp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestGRPCRoundtrip(t *testing.T) {
	n := network.NewGRPC(":0")
	setupNetwork(t, n)
	defer n.Close()

	conn, err := grpc.NewClient(n.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	client := grpcpb.NewFluxCacheClient(conn)

	r, err := client.Exec(t.Context(), &grpcpb.Command{Name: "PING"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Type != resp.TypeString || r.Str != "PONG" {
		t.Fatalf("expected PONG, got %+v", r)
	}

	r, err = client.Exec(t.Context(), &grpcpb.Command{Name: "SET", Args: []string{"foo", "bar"}})
	if err != nil {
		t.Fatal(err)
	}
	if r.Type != resp.TypeString || r.Str != "OK" {
		t.Fatalf("expected OK, got %+v", r)
	}

	r, err = client.Exec(t.Context(), &grpcpb.Command{Name: "GET", Args: []string{"foo"}})
	if err != nil {
		t.Fatal(err)
	}
	if r.Type != resp.TypeBulk || r.Bulk != "bar" {
		t.Fatalf("expected bar, got %+v", r)
	}

	r, err = client.Exec(t.Context(), &grpcpb.Command{Name: "GET", Args: []string{"nonexistent"}})
	if err != nil {
		t.Fatal(err)
	}
	if r.Type != resp.TypeNull {
		t.Fatalf("expected null, got %+v", r)
	}
}

func TestGRPCUnknownCommand(t *testing.T) {
	n := network.NewGRPC(":0")
	setupNetwork(t, n)
	defer n.Close()

	conn, err := grpc.NewClient(n.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	client := grpcpb.NewFluxCacheClient(conn)
	r, err := client.Exec(t.Context(), &grpcpb.Command{Name: "UNKNOWN"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Type != resp.TypeError {
		t.Fatalf("expected error response, got %+v", r)
	}
}
