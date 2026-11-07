.PHONY: build test lint install clean

BIN_EXT := $(if $(filter windows,$(shell go env GOOS)),.exe,)

build:
	go build -trimpath -o bin/camel-cli$(BIN_EXT) ./cmd/camel-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/camel-cli

clean:
	rm -rf bin/

build-mcp:
	go build -trimpath -o bin/camel-mcp$(BIN_EXT) ./cmd/camel-mcp

install-mcp:
	go install ./cmd/camel-mcp

build-all: build build-mcp
