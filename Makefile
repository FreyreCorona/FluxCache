PROTOC      ?= protoc
PROTOC_GEN_GO      ?= protoc-gen-go
PROTOC_GEN_GO_GRPC ?= protoc-gen-go-grpc
PROTO_DIR   := proto
PROTO_OUT   := network/grpcpb

.PHONY: all proto test test-race test-v bench bench-store bench-persistence build run clean

all: fmt vet build test

proto:
	@command -v $(PROTOC) >/dev/null 2>&1 || { echo "Error: protoc not found. Install from https://github.com/protocolbuffers/protobuf/releases"; exit 1; }
	@command -v $(PROTOC_GEN_GO) >/dev/null 2>&1 || { echo "Error: protoc-gen-go not found. Run: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"; exit 1; }
	@command -v $(PROTOC_GEN_GO_GRPC) >/dev/null 2>&1 || { echo "Error: protoc-gen-go-grpc not found. Run: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"; exit 1; }
	$(PROTOC) --proto_path=$(PROTO_DIR) \
		--go_out=$(PROTO_OUT) --go_opt=paths=source_relative \
		--go-grpc_out=$(PROTO_OUT) --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/fluxcache.proto

test:
	go test ./...

test-race:
	go test -race ./...

test-v:
	go test -v ./...

bench:
	go test -bench=. -benchmem ./...

bench-store:
	go test -bench=. -benchmem ./tests/store/

bench-persistence:
	go test -bench=. -benchmem ./tests/persistence/

build:
	go build -o fluxcache ./

run:
	go run .

clean:
	rm -f fluxcache
	rm -f $(PROTO_OUT)/*.pb.go
