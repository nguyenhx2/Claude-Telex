---
name: Effective Go
description: Comprehensive Go coding standards and idioms based on go.dev/doc/effective_go. Apply when writing, reviewing, or refactoring Go code.
---

# Effective Go

Reference: https://go.dev/doc/effective_go

Apply these standards whenever writing or reviewing Go code.

## Formatting

- **Always run `gofmt`** (or `go fmt`). All Go code must be `gofmt`-compliant.
- Use **tabs for indentation**, spaces for alignment.
- No line length limit, but keep lines readable. Break long lines at natural points.
- Opening braces `{` must be on the **same line** as the control structure (if, for, switch, func). Go inserts semicolons; a brace on the next line causes compilation errors.

```go
// ✅ Correct
if err != nil {
    return err
}

// ❌ Wrong — semicolon inserted before brace
if err != nil
{
    return err
}
```

## Naming

### General Rules
- Names are **semantically meaningful**: uppercase first letter = exported (public), lowercase = unexported (private).
- Use **MixedCaps** or **mixedCaps**, never underscores in names.
- Keep names **short and descriptive**. Local variables can be very short (`i`, `r`, `buf`). Exported names should be more descriptive.

### Package Names
- **lowercase, single-word**, no underscores or mixedCaps: `http`, `bufio`, `fmt`
- The package name is part of the qualified name — avoid redundancy:
  - `bufio.Reader` ✅ not `bufio.BufReader` ❌
  - `ring.New()` ✅ not `ring.NewRing()` ❌

### Getters/Setters
- Getter: `Owner()` ✅ not `GetOwner()` ❌
- Setter: `SetOwner()` ✅

### Interface Names
- Single-method interfaces: method name + `-er` suffix: `Reader`, `Writer`, `Formatter`, `Stringer`
- Honor canonical names: if your type implements `String()`, call it `String`, not `ToString`.

## Control Structures

### If
- Accept optional initializer: `if err := f(); err != nil { ... }`
- Avoid unnecessary else — handle error and return early:

```go
// ✅ Idiomatic
f, err := os.Open(name)
if err != nil {
    return err
}
defer f.Close()
// use f
```

### For
- Go's only loop keyword. Three forms:
```go
for i := 0; i < n; i++ { }  // C-style
for condition { }              // while
for { }                        // infinite
```
- Range over slices, maps, channels, strings:
```go
for key, value := range m { }
for i, v := range slice { }
for _, v := range slice { }   // ignore index
```

### Switch
- Cases don't fall through by default (no `break` needed).
- Cases can be comma-separated: `case 'a', 'e', 'i':`.
- Switch with no condition = `switch true`, acts as if/else chain.

### Type Switch
```go
switch v := val.(type) {
case *bytes.Buffer:
    // v is *bytes.Buffer
case string:
    // v is string
}
```

## Functions

### Multiple Return Values
- Return both result and error: `func Open(name string) (*File, error)`
- Always check errors — never ignore the `error` return value.

### Named Results
- Use named results for documentation, not to save lines:
```go
func ReadFull(r Reader, buf []byte) (n int, err error) { ... }
```

### Defer
- Deferred functions run in LIFO order when the enclosing function returns.
- Use for cleanup: `defer f.Close()`, `defer mu.Unlock()`
- Arguments are evaluated at `defer` time, not execution time.

## Data

### `new` vs `make`
- `new(T)` → allocates zeroed `T`, returns `*T`. Use for structs, basic types.
- `make(T, args)` → initializes slices, maps, channels. Returns `T` (not pointer).

```go
p := new(SyncedBuffer) // *SyncedBuffer, zero value, ready to use
s := make([]int, 10)   // []int with len=10
m := make(map[string]int) // initialized map
ch := make(chan int, 5)    // buffered channel
```

### Composite Literals
- Use field labels for clarity:
```go
return &File{fd: fd, name: name}  // ✅ clear
return &File{fd, name, nil, 0}    // ❌ fragile
```

### Slices
- Slices wrap arrays; copies share underlying data until modified.
- Use `append` to grow: `slice = append(slice, elem)`
- Avoid pre-allocating unless you know the size.

### Maps
- Key must be comparable (==). Value can be anything.
- Check existence: `v, ok := m[key]`
- Delete: `delete(m, key)`

## Methods

### Pointer vs Value Receivers
- **Value receivers**: read-only methods, small types, immutable.
- **Pointer receivers**: methods that modify state, large structs, consistency.
- Rule: **value methods** can be called on values and pointers; **pointer methods** can only be called on pointers.

```go
type ByteSlice []byte

func (s ByteSlice) Len() int     { return len(s) }  // value receiver
func (p *ByteSlice) Append(data []byte) { *p = append(*p, data...) }  // pointer receiver
```

## Interfaces

- Interfaces define behavior, not data: "if something can do this, use it here."
- Keep interfaces small: 1-2 methods is ideal.
- Accept interfaces, return concrete types:

```go
// ✅ Accept interface
func Copy(dst io.Writer, src io.Reader) (int64, error)

// ✅ Return concrete
func NewBuffer(buf []byte) *Buffer
```

### Type Assertions
```go
str, ok := val.(string)
if !ok {
    // val is not a string
}
```

### Interface Compliance Check
```go
var _ json.Marshaler = (*MyType)(nil) // compile-time check
```

## Embedding

- Embed interfaces in interfaces, types in structs.
- Embedded methods are promoted — the outer type "inherits" them.
- Use for composition, NOT inheritance.

```go
type ReadWriter struct {
    *Reader  // promoted Read method
    *Writer  // promoted Write method
}
```

## Concurrency

### Core Philosophy
**"Do not communicate by sharing memory; share memory by communicating."**

### Goroutines
```go
go func() {
    result := heavyWork()
    ch <- result
}()
```
- Goroutines are cheap — launch thousands if needed.
- Always ensure goroutines can exit (avoid leaks).

### Channels
```go
ch := make(chan int)       // unbuffered — synchronous
ch := make(chan int, 100)  // buffered
ch <- value                // send
v := <-ch                  // receive
```
- Use channels to coordinate goroutines.
- Use `select` for multiplexing multiple channels.

### Patterns
```go
// Fan-out/fan-in
for i := 0; i < numWorkers; i++ {
    go worker(jobs, results)
}

// Timeout
select {
case result := <-ch:
    use(result)
case <-time.After(5 * time.Second):
    fmt.Println("timeout")
}

// Done channel for cancellation
done := make(chan struct{})
go func() {
    defer close(done)
    work()
}()
<-done
```

## Error Handling

### Error Values
- Errors are values — `error` is an interface with `Error() string`.
- Create sentinel errors: `var ErrNotFound = errors.New("not found")`
- Create custom types for rich errors:

```go
type PathError struct {
    Op   string
    Path string
    Err  error
}

func (e *PathError) Error() string {
    return e.Op + " " + e.Path + ": " + e.Err.Error()
}
```

### Error Wrapping
```go
return fmt.Errorf("open config: %w", err)  // wrap with context
errors.Is(err, os.ErrNotExist)             // check wrapped
errors.As(err, &pathErr)                   // unwrap to type
```

### Panic/Recover
- `panic` only for truly unrecoverable errors (programmer mistakes).
- `recover` only in deferred functions. Convert panic to error at API boundaries:

```go
func safeCall() (err error) {
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("internal error: %v", r)
        }
    }()
    riskyWork()
    return nil
}
```

## Zero Values

Design types so that zero values are useful:
- `bytes.Buffer` zero value = empty buffer ready to use
- `sync.Mutex` zero value = unlocked mutex
- Use `var wg sync.WaitGroup` directly, no constructor needed.

## Testing

- Test files: `*_test.go` in the same package.
- Test functions: `func TestXxx(t *testing.T)`
- Table-driven tests are idiomatic:

```go
tests := []struct {
    name     string
    input    string
    expected string
}{
    {"basic", "hello", "HELLO"},
    {"empty", "", ""},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        got := Transform(tt.input)
        if got != tt.expected {
            t.Errorf("got %q, want %q", got, tt.expected)
        }
    })
}
```

## Project Structure

```
project/
├── cmd/appname/main.go    # Entry points
├── internal/              # Private packages
│   ├── service/
│   └── handler/
├── pkg/                   # Public library packages (if any)
├── go.mod
├── go.sum
└── README.md
```

- `internal/` is for packages that should NOT be importable by external projects.
- `cmd/` for each binary.
- Keep `main.go` thin — delegate to internal packages.
