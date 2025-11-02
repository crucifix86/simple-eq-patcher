# Quick Start Guide

## For Server Administrators

### 1️⃣ One-Time Setup

```bash
# Create patch directory
sudo mkdir -p /var/www/eq-patches
sudo chown $USER:$USER /var/www/eq-patches

# Configure nginx (if not already done)
sudo nano /etc/nginx/sites-available/default
```

Add to nginx config:
```nginx
location /patches {
    alias /var/www/eq-patches;
    autoindex off;
    add_header Access-Control-Allow-Origin *;
}
```

```bash
# Reload nginx
sudo nginx -t
sudo systemctl reload nginx

# Test web server
curl http://localhost/patches/
```

### 2️⃣ Add Files to Patch

```bash
cd /var/www/eq-patches

# Copy any files you want players to have
# IMPORTANT: Maintain directory structure!

# Example: Root files
cp ~/everquest_rof2/spells_us.txt .
cp ~/everquest_rof2/dbg.txt .

# Example: Subdirectories (create as needed)
mkdir -p Resources
cp ~/everquest_rof2/Resources/*.txt Resources/

mkdir -p UI/default
cp ~/everquest_rof2/UI/default/*.xml UI/default/
```

### 3️⃣ Generate Manifest

```bash
/home/doug/simple-eq-patcher/server/manifest-builder /var/www/eq-patches
```

You should see:
```
Scanning directory: /var/www/eq-patches
  Added: spells_us.txt (123456 bytes, md5: a1b2c3d4)
  Added: dbg.txt (987 bytes, md5: e5f6g7h8)
  ...

✓ Manifest created: /var/www/eq-patches/manifest.json
✓ Total files: 3
```

### 4️⃣ Test from Server

```bash
curl http://localhost/patches/manifest.json
```

Should return JSON with file list.

### 5️⃣ Distribute to Players

Give players:
1. `patcher.exe` (from `/home/doug/simple-eq-patcher/client/patcher.exe`)
2. Pre-configured `patcher-config.json`:

```json
{
  "server_url": "http://YOUR_SERVER_IP/patches",
  "game_exe": "eqgame.exe",
  "game_args": "patchme"
}
```

Or just give them `patcher.exe` and tell them to edit config after first run.

## For Players

### Installation

1. Download `patcher.exe` from your server admin
2. Copy to your EverQuest folder (same folder as `eqgame.exe`)
3. Double-click `patcher.exe`

First run creates `patcher-config.json` - edit it with server URL from your admin.

### Every Time You Play

1. Double-click `patcher.exe`
2. It checks for updates
3. Downloads any new/changed files
4. Launches EverQuest

That's it!

## Daily Server Workflow

When you update files (spells, zones, etc.):

```bash
# 1. Copy updated files to patch directory
cp /opt/eqemu/server/spells_us.txt /var/www/eq-patches/

# 2. Regenerate manifest
/home/doug/simple-eq-patcher/server/manifest-builder /var/www/eq-patches

# 3. Done!
```

Players will automatically get the updates next time they run the patcher.

## Example: Real World Usage

### Scenario: You updated spells and added a new zone

```bash
cd /var/www/eq-patches

# Copy updated spells
cp /opt/eqemu/server/spells_us.txt .

# Copy new zone files
cp /opt/eqemu/zones/newzone.eqg .
cp /opt/eqemu/zones/newzone_chr.txt .
cp /opt/eqemu/zones/newzone.zon .

# Regenerate manifest
/home/doug/simple-eq-patcher/server/manifest-builder /var/www/eq-patches
```

Output:
```
Scanning directory: /var/www/eq-patches
  Added: spells_us.txt (125000 bytes, md5: NEW_HASH)      ← Updated
  Added: dbg.txt (987 bytes, md5: OLD_HASH)               ← Unchanged
  Added: newzone.eqg (2500000 bytes, md5: NEW_HASH)       ← New file
  Added: newzone_chr.txt (150 bytes, md5: NEW_HASH)       ← New file
  Added: newzone.zon (50000 bytes, md5: NEW_HASH)         ← New file

✓ Manifest created: /var/www/eq-patches/manifest.json
✓ Total files: 5
```

Next time players run patcher:
```
Downloading manifest...
✓ Manifest loaded (5 files)

Checking files...
  [OK] dbg.txt
  [HASH MISMATCH] spells_us.txt
  [MISSING] newzone.eqg
  [MISSING] newzone_chr.txt
  [MISSING] newzone.zon

4 file(s) need updating

Downloading files...
[1/4] spells_us.txt... ✓
[2/4] newzone.eqg... ✓
[3/4] newzone_chr.txt... ✓
[4/4] newzone.zon... ✓

✓ All files updated!

Launching game...
✓ Game launched successfully!
```

## Pro Tips

### Tip 1: Test Before Distributing

Create a test directory and use the Linux patcher:
```bash
mkdir ~/eq-test
cd ~/eq-test

# Create test config
cat > patcher-config.json << EOF
{
  "server_url": "http://localhost/patches",
  "game_exe": "echo",
  "game_args": "Game would launch here"
}
EOF

# Run patcher
/home/doug/simple-eq-patcher/client/patcher-linux
```

### Tip 2: Version Your Patches

Keep backups:
```bash
cp /var/www/eq-patches/manifest.json /var/www/eq-patches/manifest.json.backup-$(date +%Y%m%d)
```

### Tip 3: Automate with Cron

```bash
# Auto-regenerate manifest every hour
0 * * * * /home/doug/simple-eq-patcher/server/manifest-builder /var/www/eq-patches
```

### Tip 4: Monitor Download Stats

```bash
# Check nginx access logs
sudo tail -f /var/log/nginx/access.log | grep patches
```

## Troubleshooting

**Problem:** Players can't download manifest

**Solution:**
```bash
# Check web server
curl http://YOUR_SERVER_IP/patches/manifest.json

# Check firewall
sudo ufw status
sudo ufw allow 80/tcp

# Check nginx
sudo systemctl status nginx
```

**Problem:** Files downloading but game doesn't see them

**Solution:**
- Make sure patcher is in the SAME directory as eqgame.exe
- Files download to current directory where patcher runs

**Problem:** Some files won't download

**Solution:**
```bash
# Check file permissions
ls -la /var/www/eq-patches/

# Fix permissions
sudo chown -R www-data:www-data /var/www/eq-patches/
sudo chmod -R 644 /var/www/eq-patches/*
```

## Need Help?

Check the main README.md for detailed documentation.
