.PHONY: build run clean release-dry lint

BIN := claude-vi

build:
	go build -ldflags="-s -w" -o $(BIN)$(shell go env GOEXE) ./cmd/claude-vi

run:
	go run ./cmd/claude-vi

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
