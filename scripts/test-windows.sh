#!/bin/bash
# Windows compatibility test script for codai project
# Fixes cross-platform issues in WSL/Windows environment

set -e

echo "🔧 Setting up Windows-compatible test environment..."

# Create local temporary directory to avoid cross-platform path issues
TEMP_DIR=$(pwd)/temp_go_build
mkdir -p "$TEMP_DIR"

# Set environment variables to use local temporary directory
export GOTMPDIR="$TEMP_DIR"
export TMP="$TEMP_DIR" 
export TEMP="$TEMP_DIR"

# Ensure Go environment is properly set
if [ -z "$GOPATH" ] && [ -z "$GOMODCACHE" ]; then
    export GOPATH=$(go env GOPATH)
    export GOMODCACHE=$(go env GOMODCACHE)
fi

echo "📁 Using temporary directory: $TEMP_DIR"
echo "🧪 Running Go tests..."

# Run all tests with proper environment
if go test -v ./...; then
    echo "✅ All tests passed successfully!"
    EXIT_CODE=0
else
    echo "❌ Some tests failed!"
    EXIT_CODE=1
fi

# Clean up temporary directory
echo "🧹 Cleaning up temporary files..."
rm -rf "$TEMP_DIR"

echo "📊 Test execution completed."
exit $EXIT_CODE