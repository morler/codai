.PHONY: build test test-windows clean install lint

# Default Go build with Windows compatibility
build:
	@mkdir -p ./temp_go_build
	@GOTMPDIR=$$(pwd)/temp_go_build TMP=$$(pwd)/temp_go_build TEMP=$$(pwd)/temp_go_build go build -v ./...
	@rm -rf ./temp_go_build

# Standard test command
test:
	go test -v ./...

# Windows-compatible test command that handles cross-platform issues
test-windows:
	@echo "🔧 Running Windows-compatible tests..."
	@./scripts/test-windows.sh

# Clean build artifacts and temporary files
clean:
	go clean -cache -modcache -testcache
	rm -rf temp_go_build
	rm -rf test_temp_*

# Install globally
install:
	go install github.com/meysamhadeli/codai@latest

# Run linter (if available)
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "⚠️  golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Development setup
dev-setup:
	@echo "🚀 Setting up development environment..."
	@go mod tidy
	@go mod download
	@echo "✅ Development setup complete!"