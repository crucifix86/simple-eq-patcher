#!/bin/bash

set -e

echo "════════════════════════════════════════════════════════"
echo "  Simple EQ Patcher - Ubuntu 24.04 Server Installation"
echo "════════════════════════════════════════════════════════"
echo ""

# Determine if we need sudo
if [ "$EUID" -eq 0 ]; then
   SUDO=""
   echo "🔑 Running as root"
else
   SUDO="sudo"
   echo "🔑 Running as regular user (will use sudo when needed)"
fi

# Check Ubuntu version
echo "📋 Checking system..."
if ! grep -q "Ubuntu 24" /etc/os-release 2>/dev/null; then
    echo "⚠️  Warning: This script is designed for Ubuntu 24.04"
    echo "   It may work on other versions, but is untested"
    read -p "   Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Install dependencies
echo ""
echo "📦 Installing dependencies..."
if ! command -v go &> /dev/null; then
    echo "  Installing Go..."
    $SUDO apt-get update
    $SUDO apt-get install -y golang-go
else
    echo "  ✓ Go already installed: $(go version)"
fi

if ! command -v nginx &> /dev/null; then
    echo "  Installing nginx..."
    $SUDO apt-get install -y nginx
else
    echo "  ✓ nginx already installed"
fi

if ! command -v zip &> /dev/null; then
    echo "  Installing zip..."
    $SUDO apt-get install -y zip
else
    echo "  ✓ zip already installed"
fi

# Build server tools only (client executables are pre-compiled)
echo ""
echo "🔨 Building server manifest-builder..."
cd server
go build -o manifest-builder manifest-builder.go
if [ $? -eq 0 ]; then
    echo "  ✓ Server tool built: server/manifest-builder"
else
    echo "  ✗ Failed to build server tool"
    exit 1
fi
cd ..

echo "  ✓ Client executables (pre-compiled):"
echo "    - client/LaunchPad.exe (GUI launcher)"
echo "    - client/patcher.exe (CLI fallback)"

if [ ! -f "./server/manifest-builder" ] || [ ! -f "./client/patcher.exe" ]; then
    echo "✗ Build failed!"
    exit 1
fi

echo "  ✓ Tools built successfully"

# Set up patch directory
echo ""
echo "📁 Setting up patch directory..."
PATCH_DIR="/var/www/eq-patches"

if [ -d "$PATCH_DIR" ]; then
    echo "  ⚠️  Directory $PATCH_DIR already exists"
    read -p "  Keep existing files? (Y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Nn]$ ]]; then
        $SUDO rm -rf "$PATCH_DIR"
        $SUDO mkdir -p "$PATCH_DIR"
    fi
else
    $SUDO mkdir -p "$PATCH_DIR"
fi

# Set ownership (only if not root)
if [ "$EUID" -ne 0 ]; then
    $SUDO chown $USER:$USER "$PATCH_DIR"
fi
echo "  ✓ Patch directory: $PATCH_DIR"

# Copy manifest builder to patch directory
echo ""
echo "📋 Installing manifest builder..."
cp ./server/manifest-builder "$PATCH_DIR/"
chmod +x "$PATCH_DIR/manifest-builder"
echo "  ✓ Manifest builder installed"

# Create example files
if [ ! -f "$PATCH_DIR/manifest.json" ]; then
    echo ""
    echo "📝 Creating example files..."
    echo "This is a test file. Replace with your actual EQ files." > "$PATCH_DIR/README.txt"
    cd "$PATCH_DIR"
    ./manifest-builder "$PATCH_DIR"
    cd - > /dev/null
fi

# Configure nginx
echo ""
echo "🌐 Configuring nginx..."
NGINX_CONF="/etc/nginx/sites-available/eq-patcher"

if [ -f "$NGINX_CONF" ]; then
    echo "  ⚠️  nginx config already exists: $NGINX_CONF"
    read -p "  Overwrite? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "  Skipping nginx configuration"
        SKIP_NGINX=1
    fi
fi

if [ -z "$SKIP_NGINX" ]; then
    $SUDO tee "$NGINX_CONF" > /dev/null << 'EOF'
server {
    listen 80;
    server_name _;

    # EQ Patcher endpoint
    location /patches {
        alias /var/www/eq-patches;
        autoindex off;

        # CORS headers for web-based patchers
        add_header Access-Control-Allow-Origin *;

        # Cache control
        add_header Cache-Control "public, max-age=300";

        # Security headers
        add_header X-Content-Type-Options nosniff;
    }

    # Client bundle download (recommended)
    location /download/eq-patcher-client.zip {
        alias /var/www/eq-patches/eq-patcher-client.zip;
        add_header Content-Disposition "attachment; filename=eq-patcher-client.zip";
    }

    # Individual file downloads (optional)
    location /download/LaunchPad.exe {
        alias /var/www/eq-patches/LaunchPad.exe;
    }

    location /download/patcher.exe {
        alias /var/www/eq-patches/patcher.exe;
    }

    location /download/patcher-config.json {
        alias /var/www/eq-patches/patcher-config.json;
    }
}
EOF

    # Enable site
    if [ ! -L "/etc/nginx/sites-enabled/eq-patcher" ]; then
        $SUDO ln -sf "$NGINX_CONF" /etc/nginx/sites-enabled/
    fi

    # Test nginx config
    if $SUDO nginx -t 2>/dev/null; then
        $SUDO systemctl reload nginx
        echo "  ✓ nginx configured and reloaded"
    else
        echo "  ✗ nginx configuration error"
        echo "  Check: nginx -t"
        exit 1
    fi
fi

# Check firewall
echo ""
echo "🔥 Checking firewall..."
if command -v ufw &> /dev/null; then
    if $SUDO ufw status | grep -q "Status: active"; then
        if ! $SUDO ufw status | grep -q "80/tcp"; then
            read -p "  Open port 80 in firewall? (Y/n) " -n 1 -r
            echo
            if [[ ! $REPLY =~ ^[Nn]$ ]]; then
                $SUDO ufw allow 80/tcp
                echo "  ✓ Port 80 opened"
            fi
        else
            echo "  ✓ Port 80 already open"
        fi
    else
        echo "  ℹ️  ufw firewall not active"
    fi
else
    echo "  ℹ️  ufw not installed"
fi

# Get server IP (force IPv4)
echo ""
echo "🌍 Detecting server IP..."
SERVER_IP=$(curl -4 -s ifconfig.me 2>/dev/null || curl -4 -s icanhazip.com 2>/dev/null || echo "YOUR_SERVER_IP")
if [[ "$SERVER_IP" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "  Server IPv4: $SERVER_IP"
else
    echo "  ⚠️  Could not detect IPv4 address: $SERVER_IP"
    echo "  Please manually update patcher-config.json with your server's IPv4 address"
fi

# Copy client files for distribution
echo ""
echo "📦 Preparing client files..."
if [ -f "./client/LaunchPad.exe" ]; then
    cp ./client/LaunchPad.exe "$PATCH_DIR/"
    echo "  ✓ LaunchPad.exe (GUI) copied to $PATCH_DIR"
fi
cp ./client/patcher.exe "$PATCH_DIR/"
echo "  ✓ patcher.exe (CLI fallback) copied to $PATCH_DIR"

# Create sample client config
cat > "$PATCH_DIR/patcher-config.json" << EOF
{
  "server_url": "http://$SERVER_IP/patches",
  "server_name": "EverQuest Emulator Server",
  "launcher_title": "EverQuest LaunchPad",
  "website_url": "https://www.yourserver.com",
  "website_label": "Visit Website",
  "game_exe": "eqgame.exe",
  "game_args": "patchme"
}
EOF
echo "  ✓ Sample patcher-config.json created"

# Create client bundle ZIP
echo ""
echo "📦 Creating client bundle..."
cd "$PATCH_DIR"
rm -f eq-patcher-client.zip
zip -q eq-patcher-client.zip LaunchPad.exe patcher.exe patcher-config.json
if [ $? -eq 0 ]; then
    ZIP_SIZE=$(du -h eq-patcher-client.zip | cut -f1)
    echo "  ✓ Client bundle created: eq-patcher-client.zip ($ZIP_SIZE)"
    echo "  Contains:"
    echo "    - LaunchPad.exe (GUI launcher)"
    echo "    - patcher.exe (CLI fallback)"
    echo "    - patcher-config.json (pre-configured)"
else
    echo "  ✗ Failed to create client bundle"
fi
cd - > /dev/null

# Create usage script
cat > "$PATCH_DIR/update-patches.sh" << 'EOF'
#!/bin/bash
# Quick script to regenerate manifest after adding files
cd "$(dirname "$0")"
./manifest-builder "$(pwd)"
echo ""
echo "✓ Manifest updated!"
echo "  Players will download updates on next patcher run"
EOF
chmod +x "$PATCH_DIR/update-patches.sh"
echo "  ✓ Created update-patches.sh helper script"

# Installation complete
echo ""
echo "════════════════════════════════════════════════════════"
echo "✅ Installation Complete!"
echo "════════════════════════════════════════════════════════"
echo ""
echo "📂 Patch Directory: $PATCH_DIR"
echo "🌐 Patch URL: http://$SERVER_IP/patches"
echo ""
echo "📋 Next Steps:"
echo ""
echo "1️⃣  Add your EQ files to patch directory:"
echo "   cd $PATCH_DIR"
echo "   cp /path/to/spells_us.txt ."
echo "   cp /path/to/zones/*.eqg ."
echo ""
echo "2️⃣  Update manifest:"
echo "   cd $PATCH_DIR"
echo "   ./update-patches.sh"
echo ""
echo "3️⃣  Distribute to players:"
echo ""
echo "   📦 Client Bundle (recommended - all-in-one):"
echo "      http://$SERVER_IP/download/eq-patcher-client.zip"
echo ""
echo "   Or individual files:"
echo "      http://$SERVER_IP/download/LaunchPad.exe"
echo "      http://$SERVER_IP/download/patcher.exe"
echo "      http://$SERVER_IP/download/patcher-config.json"
echo ""
echo "📖 Full documentation: README.md"
echo "🚀 Quick start guide: QUICKSTART.md"
echo ""
echo "🧪 Test patch server:"
echo "   curl http://$SERVER_IP/patches/manifest.json"
echo ""
