VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

.PHONY: build install test lint clean

build:
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o bin/gx ./cmd/gx

install:
	go install -ldflags "-s -w -X main.version=$(VERSION)" ./cmd/gx

test:
	go test -v ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/
