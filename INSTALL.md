# Installation Guide

## Ubuntu 24.04 Server (Recommended)

### Quick Install

```bash
git clone https://github.com/crucifix86/simple-eq-patcher.git
cd simple-eq-patcher
chmod +x install.sh
./install.sh
```

The installer will:
- ✅ Install Go (if not present)
- ✅ Install nginx (if not present)
- ✅ Build server and client tools
- ✅ Create `/var/www/eq-patches` directory
- ✅ Configure nginx to serve patches
- ✅ Open firewall port 80
- ✅ Create helper scripts

### What Gets Installed

```
/var/www/eq-patches/
├── manifest-builder          ← Server tool
├── update-patches.sh         ← Helper script
├── patcher.exe               ← Windows client (for distribution)
├── patcher-config.json       ← Pre-configured client config
├── manifest.json             ← Auto-generated file list
└── README.txt                ← Placeholder file
```

### Post-Installation

1. **Add your EQ files:**
   ```bash
   cd /var/www/eq-patches

   # Copy files (maintain directory structure!)
   cp /path/to/spells_us.txt .
   cp /path/to/zones/*.eqg .

   # Create subdirectories as needed
   mkdir -p UI/default
   cp /path/to/ui/*.xml UI/default/
   ```

2. **Generate manifest:**
   ```bash
   cd /var/www/eq-patches
   ./update-patches.sh
   ```

3. **Test patch server:**
   ```bash
   curl http://YOUR_SERVER_IP/patches/manifest.json
   ```

4. **Distribute to players:**
   - Download: `http://YOUR_SERVER_IP/download/patcher.exe`
   - Or manually copy from `/var/www/eq-patches/patcher.exe`
   - Include `patcher-config.json` (already has correct server URL)

## Manual Installation (Any Linux)

### Prerequisites

```bash
# Install Go (1.21 or later)
sudo apt-get update
sudo apt-get install golang-go

# Install nginx
sudo apt-get install nginx
```

### Build

```bash
git clone https://github.com/crucifix86/simple-eq-patcher.git
cd simple-eq-patcher
./build.sh
```

### Manual Setup

```bash
# Create patch directory
sudo mkdir -p /var/www/eq-patches
sudo chown $USER:$USER /var/www/eq-patches

# Copy manifest builder
cp server/manifest-builder /var/www/eq-patches/

# Configure nginx
sudo nano /etc/nginx/sites-available/eq-patcher
```

Add:
```nginx
server {
    listen 80;
    server_name _;

    location /patches {
        alias /var/www/eq-patches;
        autoindex off;
        add_header Access-Control-Allow-Origin *;
    }

    location /download/patcher.exe {
        alias /var/www/eq-patches/patcher.exe;
    }
}
```

Enable and reload:
```bash
sudo ln -s /etc/nginx/sites-available/eq-patcher /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

### Firewall

```bash
sudo ufw allow 80/tcp
```

## Windows (Client Development)

### Build from Source

1. Install Go: https://go.dev/dl/
2. Clone repo
3. Build:
   ```powershell
   cd client
   go build -o patcher.exe patcher.go
   ```

### Pre-built Binary

Just download `patcher.exe` from the releases page.

## Docker (Optional)

Coming soon - Docker container for patch server.

## Troubleshooting

### Build fails

```bash
# Make sure Go is installed
go version

# Clean and rebuild
rm -rf server/manifest-builder client/patcher*
./build.sh
```

### nginx errors

```bash
# Test config
sudo nginx -t

# Check logs
sudo tail -f /var/log/nginx/error.log

# Restart nginx
sudo systemctl restart nginx
```

### Permission denied

```bash
# Fix patch directory permissions
sudo chown -R $USER:$USER /var/www/eq-patches
chmod 755 /var/www/eq-patches
```

### Firewall blocking

```bash
# Check firewall status
sudo ufw status

# Open port 80
sudo ufw allow 80/tcp

# Or disable firewall (not recommended)
sudo ufw disable
```

### Can't access from internet

- Check VPS provider firewall/security groups
- Verify port 80 is open externally
- Test with: `curl http://YOUR_PUBLIC_IP/patches/manifest.json`

## Updating

```bash
cd simple-eq-patcher
git pull
./build.sh

# Copy new tools to patch directory
cp server/manifest-builder /var/www/eq-patches/
```

## Uninstall

```bash
# Remove patch directory
sudo rm -rf /var/www/eq-patches

# Remove nginx config
sudo rm /etc/nginx/sites-enabled/eq-patcher
sudo rm /etc/nginx/sites-available/eq-patcher
sudo systemctl reload nginx

# Close firewall port (optional)
sudo ufw delete allow 80/tcp

# Remove source
rm -rf ~/simple-eq-patcher
```

## Production Checklist

Before going live:

- [ ] Test manifest generation with real files
- [ ] Test client download and patching
- [ ] Configure HTTPS (Let's Encrypt recommended)
- [ ] Set up automated backups of `/var/www/eq-patches`
- [ ] Monitor nginx access logs
- [ ] Document your specific file layout for team
- [ ] Create update procedure documentation

## Support

- GitHub Issues: https://github.com/crucifix86/simple-eq-patcher/issues
- EQEmu Forums: https://www.eqemulator.org/forums/
