.PHONY: build install test clean release dev lint

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u +%Y-%m-%d)
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.Date=$(DATE)"

build:
	go build $(LDFLAGS) -o bin/termiflow ./cmd/termiflow

install:
	go install $(LDFLAGS) ./cmd/termiflow

test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run

clean:
	rm -rf bin/ dist/ coverage.out coverage.html

# Cross-compilation for releases
release:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/termiflow-linux-amd64 ./cmd/termiflow
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/termiflow-linux-arm64 ./cmd/termiflow
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/termiflow-darwin-amd64 ./cmd/termiflow
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/termiflow-darwin-arm64 ./cmd/termiflow
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/termiflow-windows-amd64.exe ./cmd/termiflow

# Development helpers
dev:
	go run ./cmd/termiflow $(ARGS)

db-reset:
	rm -f ~/.local/share/termiflow/termiflow.db

# Download dependencies
deps:
	go mod download
	go mod tidy
