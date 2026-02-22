---
description: How to build and test Claude Telex
---

## Build

// turbo-all

1. Build binary (Windows):
```
go build -ldflags="-s -w -H windowsgui" -o claude-telex.exe ./cmd/claude-telex
```

2. Build binary (macOS/Linux):
```
go build -ldflags="-s -w" -o claude-telex ./cmd/claude-telex
```

3. Or use Make:
```
make build
```

## Test

// turbo-all

1. Run all tests:
```
go test ./...
```

2. Run tests with verbose output:
```
go test -v ./...
```

3. Run patcher tests only:
```
go test -v ./internal/patcher/...
```

## Release

1. Create a snapshot release (dry run):
```
goreleaser release --snapshot --clean
```

## Lint

1. Run linter:
```
golangci-lint run ./...
```
