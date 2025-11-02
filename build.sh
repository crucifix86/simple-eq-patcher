#!/bin/bash

echo "════════════════════════════════════════"
echo "  Building Simple EQ Patcher"
echo "════════════════════════════════════════"
echo ""

# Build server manifest builder (Linux)
echo "Building server manifest-builder..."
cd server
go build -o manifest-builder manifest-builder.go
if [ $? -eq 0 ]; then
    echo "✓ Server tool built: server/manifest-builder"
else
    echo "✗ Failed to build server tool"
    exit 1
fi
cd ..

# Build CLI patcher (Windows) - fallback
echo ""
echo "Building CLI patcher for Windows..."
cd client
GOOS=windows GOARCH=amd64 go build -o patcher.exe patcher.go
if [ $? -eq 0 ]; then
    echo "✓ CLI patcher built: client/patcher.exe"
else
    echo "✗ Failed to build CLI patcher"
    exit 1
fi
cd ..

# Build GUI LaunchPad (Windows)
echo ""
echo "Building GUI LaunchPad.exe for Windows..."
cd client

# Get Fyne dependencies
echo "  Getting Fyne dependencies..."
go get fyne.io/fyne/v2@latest
go mod tidy

# Generate Windows resource file with icon using windres
echo "  Embedding icon..."
x86_64-w64-mingw32-windres icon.rc -O coff -o rsrc_windres.syso

# Rename to standard name for Go to pick up automatically
mv rsrc_windres.syso rsrc.syso

# Build with mingw
echo "  Compiling LaunchPad.exe..."
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc \
  go build -ldflags="-H windowsgui -s -w" -o LaunchPad.exe launchpad.go graphics.go browser.go

if [ $? -eq 0 ]; then
    echo "✓ GUI LaunchPad built: client/LaunchPad.exe"
else
    echo "⚠ Failed to build GUI LaunchPad (will use CLI patcher instead)"
fi
cd ..

# Build client patcher (Linux for testing)
echo ""
echo "Building CLI patcher for Linux (testing)..."
cd client
go build -o patcher-linux patcher.go
if [ $? -eq 0 ]; then
    echo "✓ Linux patcher built: client/patcher-linux"
else
    echo "✗ Failed to build Linux patcher"
fi
cd ..

echo ""
echo "════════════════════════════════════════"
echo "✓ Build complete!"
echo "════════════════════════════════════════"
echo ""
echo "Server tool: ./server/manifest-builder"
echo "Client tools:"
echo "  - LaunchPad.exe (GUI launcher)"
echo "  - patcher.exe (CLI fallback)"
echo ""
