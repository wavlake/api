#!/bin/bash

# Pre-commit setup script for Go projects
# This helps catch build errors before committing and pushing to GitHub

set -e

echo "🔧 Setting up pre-commit hooks for Go project..."

# Create .git/hooks directory if it doesn't exist
mkdir -p .git/hooks

# Create pre-commit hook
cat > .git/hooks/pre-commit << 'EOF'
#!/bin/bash

# Pre-commit hook for Go projects
# Runs build checks, formatting, and linting before allowing commits

set -e

echo "🔍 Running pre-commit checks..."

# 1. Check if Go files have been modified
go_files_changed=$(git diff --cached --name-only --diff-filter=ACM | grep '\.go$' || true)

if [ -z "$go_files_changed" ]; then
    echo "✅ No Go files changed, skipping Go checks"
    exit 0
fi

echo "📁 Go files changed:"
echo "$go_files_changed"

# 2. Run go mod tidy
echo "📦 Running go mod tidy..."
go mod tidy

# 3. Run go fmt on changed files
echo "🎨 Running go fmt..."
for file in $go_files_changed; do
    if [ -f "$file" ]; then
        go fmt "$file"
        # Add formatted files back to staging
        git add "$file"
    fi
done

# 4. Run go vet
echo "🔍 Running go vet..."
go vet ./...

# 5. Test build
echo "🏗️  Testing build..."
go build ./...

# 6. Run tests if they exist
if ls *_test.go 1> /dev/null 2>&1; then
    echo "🧪 Running tests..."
    go test ./...
fi

# 7. Check for common issues
echo "🔧 Checking for common issues..."

# Check for TODO/FIXME comments in staged files
todo_comments=$(git diff --cached | grep -E "^\+.*TODO|^\+.*FIXME" || true)
if [ -n "$todo_comments" ]; then
    echo "⚠️  Warning: Found TODO/FIXME comments in staged changes:"
    echo "$todo_comments"
    echo "Consider addressing these before committing."
fi

# Check for debug prints
debug_prints=$(git diff --cached | grep -E "^\+.*(fmt\.Print|log\.Print|console\.log)" || true)
if [ -n "$debug_prints" ]; then
    echo "⚠️  Warning: Found debug print statements:"
    echo "$debug_prints"
    read -p "Continue with commit? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

echo "✅ All pre-commit checks passed!"

EOF

# Make the hook executable
chmod +x .git/hooks/pre-commit

echo "✅ Pre-commit hook installed!"

# Optional: Install golangci-lint for more comprehensive linting
echo ""
echo "🔧 Would you like to install golangci-lint for advanced linting? (y/N)"
read -p "Install golangci-lint? " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "📦 Installing golangci-lint..."
    
    # Install golangci-lint
    if command -v brew &> /dev/null; then
        brew install golangci-lint
    else
        curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.55.2
    fi
    
    # Add golangci-lint to pre-commit hook
    sed -i.bak '/# 6. Run tests/i\
# 5.5. Run golangci-lint\
echo "🔍 Running golangci-lint..."\
golangci-lint run\
' .git/hooks/pre-commit
    
    echo "✅ golangci-lint installed and added to pre-commit hook!"
fi

# Create .golangci.yml config file
cat > .golangci.yml << 'EOF'
# golangci-lint configuration
run:
  timeout: 5m
  issues-exit-code: 1
  tests: true

output:
  format: colored-line-number
  print-issued-lines: true
  print-linter-name: true

linters-settings:
  govet:
    check-shadowing: true
  gofmt:
    simplify: true
  goimports:
    local-prefixes: github.com/wavlake/api
  golint:
    min-confidence: 0.8
  misspell:
    locale: US

linters:
  enable:
    - bodyclose
    - deadcode
    - depguard
    - dogsled
    - errcheck
    - gofmt
    - goimports
    - golint
    - govet
    - ineffassign
    - misspell
    - structcheck
    - typecheck
    - unconvert
    - unparam
    - unused
    - varcheck
  disable:
    - gosec # Can be too strict for some use cases

issues:
  exclude-use-default: false
  exclude:
    # Exclude some lints for test files
    - path: _test\.go
      linters:
        - golint
        - errcheck
EOF

echo ""
echo "📝 Created .golangci.yml configuration file"

echo ""
echo "🎉 Pre-commit setup complete!"
echo ""
echo "Now your commits will automatically:"
echo "  ✅ Format Go code with go fmt"
echo "  ✅ Run go mod tidy"
echo "  ✅ Check for build errors"
echo "  ✅ Run go vet for common issues"
echo "  ✅ Run tests (if present)"
echo "  ✅ Warn about debug statements"
if command -v golangci-lint &> /dev/null; then
    echo "  ✅ Run golangci-lint for advanced linting"
fi
echo ""
echo "💡 To bypass pre-commit hooks (not recommended):"
echo "   git commit --no-verify"