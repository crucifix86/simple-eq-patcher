#!/bin/bash

set -e

echo "════════════════════════════════════════════════════════"
echo "  Simple EQ Patcher - Ubuntu 24.04 Server Installation"
echo "════════════════════════════════════════════════════════"
echo ""

# Check if running as root
if [ "$EUID" -eq 0 ]; then
   echo "⚠️  Please run as regular user, not root"
   echo "   The script will ask for sudo when needed"
   exit 1
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
    sudo apt-get update
    sudo apt-get install -y golang-go
else
    echo "  ✓ Go already installed: $(go version)"
fi

if ! command -v nginx &> /dev/null; then
    echo "  Installing nginx..."
    sudo apt-get install -y nginx
else
    echo "  ✓ nginx already installed"
fi

# Build the tools
echo ""
echo "🔨 Building patcher tools..."
./build.sh

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
        sudo rm -rf "$PATCH_DIR"
        sudo mkdir -p "$PATCH_DIR"
    fi
else
    sudo mkdir -p "$PATCH_DIR"
fi

sudo chown $USER:$USER "$PATCH_DIR"
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
    sudo tee "$NGINX_CONF" > /dev/null << 'EOF'
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

    # Optional: Serve patcher files for download
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
        sudo ln -sf "$NGINX_CONF" /etc/nginx/sites-enabled/
    fi

    # Test nginx config
    if sudo nginx -t 2>/dev/null; then
        sudo systemctl reload nginx
        echo "  ✓ nginx configured and reloaded"
    else
        echo "  ✗ nginx configuration error"
        echo "  Check: sudo nginx -t"
        exit 1
    fi
fi

# Check firewall
echo ""
echo "🔥 Checking firewall..."
if command -v ufw &> /dev/null; then
    if sudo ufw status | grep -q "Status: active"; then
        if ! sudo ufw status | grep -q "80/tcp"; then
            read -p "  Open port 80 in firewall? (Y/n) " -n 1 -r
            echo
            if [[ ! $REPLY =~ ^[Nn]$ ]]; then
                sudo ufw allow 80/tcp
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

# Get server IP
echo ""
echo "🌍 Detecting server IP..."
SERVER_IP=$(curl -s ifconfig.me 2>/dev/null || echo "YOUR_SERVER_IP")
echo "  Server IP: $SERVER_IP"

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
  "game_exe": "eqgame.exe",
  "game_args": "patchme"
}
EOF
echo "  ✓ Sample patcher-config.json created"

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
echo "   - LaunchPad.exe (GUI launcher, recommended)"
echo "   - patcher.exe (CLI fallback)"
echo "   - patcher-config.json (already configured)"
echo ""
echo "   Download URLs:"
echo "     http://$SERVER_IP/download/LaunchPad.exe"
echo "     http://$SERVER_IP/download/patcher.exe"
echo ""
echo "📖 Full documentation: README.md"
echo "🚀 Quick start guide: QUICKSTART.md"
echo ""
echo "🧪 Test patch server:"
echo "   curl http://$SERVER_IP/patches/manifest.json"
echo ""
