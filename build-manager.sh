#!/bin/bash
# Build script for EQ Patch Manager

set -e

echo "==================================="
echo "Building EQ Patch Manager"
echo "==================================="

# Get the directory where this script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

echo ""
echo "1. Building Windows executable..."
cd manager

# Build for Windows (cross-compile)
echo "   Compiling for Windows (amd64)..."
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc \
  go build -ldflags="-H windowsgui" -o ../manager.exe .

if [ $? -eq 0 ]; then
  echo "   ✓ Windows build complete: manager.exe"
else
  echo "   ✗ Windows build failed"
  exit 1
fi

echo ""
echo "2. Building Linux executable (for testing)..."
go build -o ../manager-linux .

if [ $? -eq 0 ]; then
  echo "   ✓ Linux build complete: manager-linux"
else
  echo "   ✗ Linux build failed"
  exit 1
fi

cd ..

echo ""
echo "==================================="
echo "Build Complete!"
echo "==================================="
echo ""
echo "Output files:"
echo "  - manager.exe (Windows)"
echo "  - manager-linux (Linux)"
echo ""
echo "Manager executable size:"
ls -lh manager.exe manager-linux 2>/dev/null || true
echo ""
