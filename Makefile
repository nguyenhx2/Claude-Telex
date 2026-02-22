.PHONY: build run clean release-dry lint test

BIN := claude-telex

build:
	go build -ldflags="-s -w" -o $(BIN)$(shell go env GOEXE) ./cmd/claude-telex

run:
	go run ./cmd/claude-telex

test:
	go test ./...

clean:
	rm -f $(BIN) $(BIN).exe
	rm -rf dist/

release-dry:
	goreleaser release --snapshot --clean

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

.DEFAULT_GOAL := build
