# CRUSH.md - Codai Development Guide

## Build Commands
```bash
# Build project
make build

# Run tests with sequence
make test

# Run single test (examples)
go test -v ./code_analyzer -run TestGeneratePrompt
go test -v ./code_analyzer -run TestFunctionName

# Clean build artifacts
make clean

# Install globally
make install
```

## Code Style
- **Imports**: Standard lib first, external libs second, internal packages last with blank lines between groups
- **Error handling**: Return wrapped errors with `fmt.Errorf("context: %v", err)` or `fmt.Errorf("context: %w", err)`
- **Naming**: Use camelCase for variables/functions, PascalCase for exported types, descriptive names
- **Files**: One package per directory, use underscores for file names with multiple words
- **Structs**: Prefix with export keyword for public types, use clear field names
- **Variables**: Prefer `var` block for related variables at package level
- **Comments**: Start with `//` and match function/type name for exported items

## Testing
- Use testify assertions with `suite` package
- Test files named `*_test.go`
- Table-driven tests preferred
- Temporary directories created per test with cleanup