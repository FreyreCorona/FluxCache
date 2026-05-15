.PHONY: all test test-race test-v bench bench-store bench-persistence build run clean

all: fmt vet build test

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
