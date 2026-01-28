.PHONY: all build run clean test

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Binary name
BINARY_NAME=proxy-client
BINARY_UNIX=$(BINARY_NAME)-linux
BINARY_DARWIN=$(BINARY_NAME)-darwin
BINARY_WINDOWS=$(BINARY_NAME).exe

all: test build

build:
	$(GOBUILD) -o $(BINARY_NAME) -v client.go

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v client.go

build-darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BINARY_DARWIN) -v client.go

build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BINARY_WINDOWS) -v client.go

build-all: build-linux build-darwin build-windows

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
	rm -f $(BINARY_DARWIN)
	rm -f $(BINARY_WINDOWS)

run:
	$(GOBUILD) -o $(BINARY_NAME) -v client.go
	./$(BINARY_NAME)

deps:
	$(GOMOD) download
	$(GOMOD) verify

test:
	$(GOTEST) -v ./...

# Example: run with custom config
run-socks5:
	SERVER_URL=wss://example.com/ws \
	AUTH_TOKEN=test-token \
	LISTEN_ADDR=127.0.0.1:1080 \
	PROXY_MODE=socks5 \
	./$(BINARY_NAME)

run-http:
	SERVER_URL=wss://example.com/ws \
	AUTH_TOKEN=test-token \
	LISTEN_ADDR=127.0.0.1:8080 \
	PROXY_MODE=http \
	./$(BINARY_NAME)

run-connect:
	SERVER_URL=wss://example.com/ws \
	AUTH_TOKEN=test-token \
	LISTEN_ADDR=127.0.0.1:8080 \
	PROXY_MODE=connect \
	./$(BINARY_NAME)
