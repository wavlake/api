#!/bin/bash

# Install Git hooks for the Wavlake API project
# Run this script to set up pre-commit hooks for all developers

set -e

HOOKS_DIR=".git/hooks"
PROJECT_ROOT="$(git rev-parse --show-toplevel)"

echo "🔧 Installing Git hooks for Wavlake API..."

# Create pre-commit hook
cat > "$HOOKS_DIR/pre-commit" << 'EOF'
#!/bin/bash

# Wavlake API Pre-commit Hook
# This hook runs before every commit to ensure code quality

set -e  # Exit on any error

echo "🔍 Running pre-commit checks..."

# Check if we're in a Go project
if [ ! -f "go.mod" ]; then
    echo "❌ go.mod not found. This doesn't appear to be a Go project."
    exit 1
fi

# Store original directory
ORIGINAL_DIR=$(pwd)

# Change to project root
cd "$(git rev-parse --show-toplevel)"

echo "📦 Tidying Go modules..."
go mod tidy

echo "🔧 Building all packages..."
if ! go build ./...; then
    echo "❌ Build failed! Please fix compilation errors before committing."
    echo "💡 Run 'go build ./...' to see detailed error messages"
    exit 1
fi

echo "🏗️  Building main server binary..."
if ! go build -o server ./cmd/server; then
    echo "❌ Server build failed! Please fix the main server before committing."
    echo "💡 Run 'go build -o server ./cmd/server' to see detailed error messages"
    exit 1
fi

# Clean up the built binary
rm -f server

echo "📋 Running go vet..."
if ! go vet ./...; then
    echo "❌ go vet found issues! Please fix them before committing."
    echo "💡 Run 'go vet ./...' to see detailed issues"
    exit 1
fi

echo "🧹 Checking gofmt..."
UNFORMATTED=$(gofmt -l . | grep -v vendor/ | head -10)
if [ -n "$UNFORMATTED" ]; then
    echo "❌ Some files are not gofmt'd. Please run: gofmt -w ."
    echo "Files that need formatting:"
    echo "$UNFORMATTED"
    echo "💡 Run 'gofmt -w .' to auto-format all files"
    exit 1
fi

# Optional: Check for common issues
echo "🔍 Checking for common issues..."

# Check for TODO/FIXME comments in committed files
STAGED_FILES=$(git diff --cached --name-only --diff-filter=A | grep '\.go$' || true)
if [ -n "$STAGED_FILES" ]; then
    TODO_COUNT=$(echo "$STAGED_FILES" | xargs grep -l "TODO\|FIXME" | wc -l | tr -d ' ')
    if [ "$TODO_COUNT" -gt 0 ]; then
        echo "⚠️  Warning: Found $TODO_COUNT files with TODO/FIXME comments"
        echo "💡 Consider addressing these before committing"
    fi
fi

# Return to original directory
cd "$ORIGINAL_DIR"

echo "✅ All pre-commit checks passed!"
echo "🚀 Proceeding with commit..."
EOF

# Make hooks executable
chmod +x "$HOOKS_DIR/pre-commit"

echo "✅ Pre-commit hook installed successfully!"
echo ""
echo "The hook will now run automatically before every commit and check:"
echo "  📦 Go module tidiness"
echo "  🔧 Package compilation" 
echo "  🏗️  Server binary build"
echo "  📋 Code vet analysis"
echo "  🧹 Code formatting"
echo ""
echo "To bypass the hook in emergencies, use: git commit --no-verify"
echo ""
echo "💡 Other developers can run this script to get the same hooks:"
echo "   ./scripts/install-git-hooks.sh"