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

# Build client patcher (Windows)
echo ""
echo "Building client patcher for Windows..."
cd client
GOOS=windows GOARCH=amd64 go build -o patcher.exe patcher.go
if [ $? -eq 0 ]; then
    echo "✓ Client patcher built: client/patcher.exe"
else
    echo "✗ Failed to build client patcher"
    exit 1
fi
cd ..

# Build client patcher (Linux for testing)
echo ""
echo "Building client patcher for Linux (testing)..."
cd client
go build -o patcher-linux patcher.go
if [ $? -eq 0 ]; then
    echo "✓ Client patcher built: client/patcher-linux"
else
    echo "✗ Failed to build client patcher for Linux"
fi
cd ..

echo ""
echo "════════════════════════════════════════"
echo "✓ Build complete!"
echo "════════════════════════════════════════"
echo ""
echo "Server tool: ./server/manifest-builder"
echo "Client tool: ./client/patcher.exe"
echo ""
