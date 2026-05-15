//go:generate protoc --proto_path=../proto --go_out=grpcpb --go_opt=paths=source_relative --go-grpc_out=grpcpb --go-grpc_opt=paths=source_relative ../proto/fluxcache.proto

package network

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/FreyreCorona/FluxCache/network/grpcpb"
	"github.com/FreyreCorona/FluxCache/resp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// GRPC is a gRPC network transport.
type GRPC struct {
	addr string
	srv  *grpc.Server
	ln   net.Listener
}

// NewGRPC creates a new gRPC transport that listens on the given address.
func NewGRPC(addr string) *GRPC {
	return &GRPC{addr: addr}
}

// Listen starts the gRPC server and begins serving requests.
func (g *GRPC) Listen(handlers map[string]HandlerFunc, onWrite WriteFunc) error {
	ln, err := net.Listen("tcp", g.addr)
	if err != nil {
		return fmt.Errorf("grpc: listen: %w", err)
	}
	g.ln = ln

	g.srv = grpc.NewServer()
	grpcpb.RegisterFluxCacheServer(g.srv, &grpcService{
		handlers: handlers,
		onWrite:  onWrite,
	})
	reflection.Register(g.srv)

	return g.srv.Serve(ln)
}

// Addr returns the address the server is listening on.
func (g *GRPC) Addr() net.Addr {
	if g.ln != nil {
		return g.ln.Addr()
	}
	return nil
}

// Close gracefully stops the gRPC server.
func (g *GRPC) Close() error {
	if g.srv != nil {
		g.srv.GracefulStop()
	}
	return nil
}

type grpcService struct {
	grpcpb.UnimplementedFluxCacheServer
	handlers map[string]HandlerFunc
	onWrite  WriteFunc
}

func (s *grpcService) Exec(ctx context.Context, cmd *grpcpb.Command) (*grpcpb.ResponseValue, error) {
	name := strings.ToUpper(cmd.GetName())

	handler, ok := s.handlers[name]
	if !ok {
		return &grpcpb.ResponseValue{
			Type: resp.TypeError,
			Str:  fmt.Sprintf("unknown command: %s", name),
		}, nil
	}

	args := make([]resp.Value, len(cmd.GetArgs()))
	for i, a := range cmd.GetArgs() {
		args[i] = resp.Value{Type: resp.TypeBulk, Bulk: a}
	}

	if s.onWrite != nil {
		s.onWrite(name, args)
	}

	result := handler(args)
	return respValueToProto(result), nil
}

func respValueToProto(v resp.Value) *grpcpb.ResponseValue {
	pb := &grpcpb.ResponseValue{Type: v.Type}
	switch v.Type {
	case resp.TypeString:
		pb.Str = v.Str
	case resp.TypeBulk:
		pb.Bulk = v.Bulk
	case resp.TypeError:
		pb.Str = v.Str
	case resp.TypeInteger:
		pb.Num = int64(v.Num)
	case resp.TypeArray:
		arr := make([]*grpcpb.ResponseValue, len(v.Array))
		for i, item := range v.Array {
			arr[i] = respValueToProto(item)
		}
		pb.Array = arr
	}
	return pb
}
