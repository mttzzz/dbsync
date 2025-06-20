#!/bin/bash

# Test script with linting
set -e

VERBOSE=false
if [ "$1" = "-v" ] || [ "$1" = "--verbose" ]; then
    VERBOSE=true
fi

echo "🔍 Running Go tests with linting..."

# Запускаем форматирование
echo ""
echo "📝 Checking code formatting..."
if ! go fmt ./... | grep -q '^$'; then
    echo "❌ Code formatting issues found"
    exit 1
fi
echo "✅ Code formatting OK"

# Запускаем go vet
echo ""
echo "🔍 Running go vet..."
if ! go vet ./...; then
    echo "❌ go vet found issues"
    exit 1
fi
echo "✅ go vet OK"

# Запускаем staticcheck
echo ""
echo "🔧 Running staticcheck..."
if ! staticcheck ./...; then
    echo "❌ staticcheck found issues"
    exit 1
fi
echo "✅ staticcheck OK"

# Запускаем тесты
echo ""
echo "🧪 Running tests..."
if [ "$VERBOSE" = true ]; then
    go test -v ./...
else
    go test ./...
fi

echo ""
echo "✅ All checks passed!"
