# AGENTS.md

This document contains guidelines and commands for agentic coding agents working in this Go dependency injection repository.

## Build/Test/Lint Commands

### Running Tests
- **Run all tests**: `go test ./...`
- **Run tests in current package**: `go test`
- **Run single test**: `go test -run TestFunctionName`
- **Run tests with verbose output**: `go test -v`
- **Run tests with coverage**: `go test -cover`
- **Run specific test file**: `go test -v injector_test.go`

### Building
- **Build all packages**: `go build ./...`
- **Build current package**: `go build`
- **Build with race detection**: `go build -race`

### Linting/Validation
- **Run go vet**: `go vet ./...`
- **Run go fmt**: `go fmt ./...`
- **Check for unused imports**: `go mod tidy`

## Code Style Guidelines

### Import Organization
- Group imports in three sections: standard library, third-party libraries, and local packages
- Use blank lines between groups
- Sort imports alphabetically within each group
- Example:
```go
import (
    "fmt"
    "reflect"
    "time"

    "github.com/facebookgo/structtag"
    "github.com/hq-cml/go-tools/injector/implmap"
    orderMap "github.com/hq-cml/go-tools/order-map"
)
```

### Naming Conventions
- **Package names**: lowercase, single word when possible (`injector`, `implmap`)
- **Constants**: UPPER_SNAKE_CASE for exported constants
- **Variables**: camelCase, with descriptive names
- **Functions**: PascalCase for exported, camelCase for unexported
- **Structs**: PascalCase, with descriptive names (`Graph`, `Object`, `Startable`)
- **Interfaces**: Usually end with `-able` suffix for behavioral interfaces (`Startable`, `Closeable`, `Injectable`)

### Type and Variable Guidelines
- Use `interface{}` for generic object storage
- Use `reflect.Type` and `reflect.Value` for reflection operations
- Pointer types should be used for mutable objects and interfaces
- Use `sync.RWMutex` for concurrent access protection

### Error Handling
- Always return errors as the last return value
- Use `fmt.Errorf` for creating formatted errors
- Include context in error messages (field names, types, object names)
- Example:
```go
return nil, fmt.Errorf("dependency field=%s,injectTag=%s not found in object %s:%v", 
    fieldType.Name, injectTag, name, reflectType)
```

### Function Structure
- Keep functions focused and relatively small
- Use meaningful parameter names
- Include lock/unlock patterns for concurrent access
- Example mutex pattern:
```go
func (g *Graph) Find(name string) (*Object, bool) {
    g.mu.RLock()
    defer g.mu.RUnlock()
    return g.find(name)
}
```

### Struct Tag Usage
- Use `inject:""` for injection by type (must be a pointer to a struct)
- Use `inject:"serviceName"` for injection by name
- Use `singleton:"true"` or `singleton:"false"` for singleton behavior (default: false)
- Use `cannil:"true"` or `nilable:"true"` for nullable fields (default: false)

### Documentation
- Exported functions and types should have Go doc comments
- Use Chinese comments in this codebase (following existing pattern)
- Keep comments concise and relevant
- Example:
```go
// newGraph 创建新的依赖注入图
// 本质上它是一个Map（带插入顺序的map），它的Key有两种情况：
// refType => *Object
// tagString => *Object  // 这里的tagString是`inject`
func newGraph() *Graph {
```

### Testing Guidelines
- Test files should end with `_test.go`
- Test functions should start with `Test_` or `Test`
- Use table-driven tests when testing multiple scenarios
- Include setup/teardown in `InitDefault()` and `Close()` calls
- Example test structure:
```go
func Test_Demo1(t *testing.T) {
    InitDefault()
    defer Close()
    
    // Test code here
}
```

### Reflection Usage
- This codebase heavily uses reflection for dependency injection
- Use `reflect.TypeOf()` and `reflect.ValueOf()` frequently
- Check for pointer types using `Kind() == reflect.Ptr`
- Use `Elem()` to get underlying types
- Use `CanInterface()` and `CanSet()` before accessing fields

### Concurrency Patterns
- Use `sync.RWMutex` for protecting shared state
- Separate internal methods (no locking) from public methods (with locking)
- Use naming convention: `_methodName` for internal, `MethodName` for public
- Example:
```go
// Internal method (no lock)
func (g *Graph) _len() int {
    return g.container.Len()
}

// Public method (with lock)
func (g *Graph) Len() int {
    g.mu.RLock()
    defer g.mu.RUnlock()
    return g._len()
}
```

## Project Structure
- `injector.go` - Main dependency injection logic
- `define.go` - Core type definitions and interfaces
- `global.go` - Global instance and convenience functions
- `implmap/` - Implementation mapping registry
- `*_test.go` - Test files with examples

## Dependencies
- Uses `github.com/facebookgo/structtag` for struct tag parsing
- Uses custom `github.com/hq-cml/go-tools/order-map` for ordered map operations
- Minimum Go version: 1.13 (but project tested with Go 1.20.2)

## Common Patterns
- Registration methods with variants: `Register`, `RegisterSingle`, `RegisterNoFill`
- Panic variants: `RegisterOrFail`, `RegisterOrFailSingle`
- Global convenience functions that delegate to internal graph instance
- Start/Close lifecycle management for registered objects