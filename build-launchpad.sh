#!/bin/bash

echo "════════════════════════════════════════"
echo "  Building EverQuest LaunchPad"
echo "════════════════════════════════════════"
echo ""

cd client

# Get dependencies
echo "Getting dependencies..."
go get fyne.io/fyne/v2@latest
go mod tidy

# Build for Windows with icon
echo ""
echo "Building LaunchPad.exe for Windows..."

# Install fyne command for bundling resources
go install fyne.io/fyne/v2/cmd/fyne@latest

# Build with icon
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc \
  go build -ldflags="-H windowsgui" -o LaunchPad.exe launchpad.go

if [ -f "LaunchPad.exe" ]; then
    echo "✓ LaunchPad.exe built successfully"
else
    echo "✗ Failed to build LaunchPad.exe"
    echo ""
    echo "Note: Building GUI Windows apps from Linux requires:"
    echo "  sudo apt-get install gcc-mingw-w64-x86-64"
    exit 1
fi

cd ..

echo ""
echo "════════════════════════════════════════"
echo "✓ Build complete!"
echo "════════════════════════════════════════"
echo ""
echo "LaunchPad: ./client/LaunchPad.exe"
echo ""
