# Claude Telex - Project Rules

## Overview

Claude Telex is a Go CLI tool that patches Claude Code's `cli.js` to fix Vietnamese TELEX input issues. The patcher intercepts `\x7f` (DEL) characters sent by Vietnamese IMEs and processes them one-by-one instead of in batches.

## Build & Run

```bash
# Build (Windows)
go build -ldflags="-s -w -H windowsgui" -o claude-telex.exe ./cmd/claude-telex

# Build (macOS/Linux)
go build -ldflags="-s -w" -o claude-telex ./cmd/claude-telex

# Dev mode (with console output)
go run ./cmd/claude-telex

# Test
go test ./...
```

## Project Structure

```
cmd/claude-telex/    # Entry point, single-instance lock
internal/
  patcher/           # Core: find cli.js, extract vars, inject fix
  tray/              # System tray icon management
  settings/          # HTTP settings server (:9315)
  icon/              # Programmatic icon rendering
  hotkey/            # Global hotkey (Ctrl+Alt+V)
  autostart/         # OS-level auto-start registration
  state/             # JSON config persistence
assets/ui/           # Embedded HTML settings UI
```

## Conventions

- Follow **Effective Go** standards (see `.agents/skills/effective-go/`)
- Package names: lowercase, single-word
- Error handling: always check errors, wrap with `fmt.Errorf("context: %w", err)`
- Variable names from minified JS: use descriptive comments
- Tests: table-driven, in `*_test.go` alongside source
- Regex patterns for minified JS: document what they match

## Key Patterns

- **Patch marker**: `/* Vietnamese IME fix */` - injected to detect already-patched files
- **Variable extraction**: regex-based extraction of dynamic var names from minified JS
- **Backup-restore**: always create `.bak` before patching, restore on failure
