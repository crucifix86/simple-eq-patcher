# Quick Start Guide

## For Server Administrators (Ubuntu 24.04)

### 1️⃣ Install (One Command!)

```bash
git clone https://github.com/crucifix86/simple-eq-patcher.git
cd simple-eq-patcher
./install.sh
```

**That's it!** The installer does everything:
- Installs Go and nginx
- Builds tools
- Creates `/var/www/eq-patches`
- Configures nginx
- Opens firewall

### 2️⃣ Add YOUR Custom Files

**⚠️ ONLY copy files unique to YOUR server - not the whole EQ client!**

```bash
cd /var/www/eq-patches

# Copy ONLY your custom/modified files
cp /path/to/custom/spells_us.txt .
cp /path/to/custom/dbg.txt .

# Custom zones (only ones YOU created/modified!)
cp /path/to/custom/mycustomzone.eqg .

# Custom UI (if you have it)
mkdir -p UI/default
cp /path/to/custom/EQUI_*.xml UI/default/
```

**Don't copy:**
- ❌ Entire EQ client
- ❌ Vanilla/retail files
- ❌ Game executables

### 3️⃣ Generate Manifest

```bash
cd /var/www/eq-patches
./update-patches.sh
```

Output:
```
Scanning directory: /var/www/eq-patches
  Added: spells_us.txt (123456 bytes, md5: a1b2c3d4)
  Added: mycustomzone.eqg (987654 bytes, md5: e5f6g7h8)

✓ Manifest created: manifest.json
✓ Total files: 2
```

### 4️⃣ Test It Works

```bash
curl http://YOUR_SERVER_IP/patches/manifest.json
```

Should show your files in JSON format.

### 5️⃣ Give to Players

**Files from `/var/www/eq-patches/`:**
1. `LaunchPad.exe` (GUI launcher with Play button - recommended)
2. `patcher-config.json` (already has your server IP!)
3. `patcher.exe` (CLI fallback - optional)

Players copy `LaunchPad.exe` and `patcher-config.json` to their `C:\EverQuest\` folder.

## For Players

### Setup (One Time)

1. Get `LaunchPad.exe` and `patcher-config.json` from your server admin
2. Copy both to your EverQuest folder (where `eqgame.exe` is)
3. Done!

### Every Time You Play

Double-click `LaunchPad.exe` - you'll see:
- A window showing your server name
- Progress bar for downloads
- "PLAY" button

The launcher automatically:
- Checks for updates
- Downloads any new/changed files
- Launches EverQuest when you click PLAY

## Daily Workflow (When You Update Server)

```bash
cd /var/www/eq-patches

# Copy updated files
cp /opt/eqemu/server/spells_us.txt .

# Regenerate manifest
./update-patches.sh

# Done!
```

Players auto-download updates next time they run patcher.

## Complete Example

### You updated spells and added a custom zone:

```bash
cd /var/www/eq-patches

# Updated spells
cp /opt/eqemu/server/spells_us.txt .

# Your NEW custom zone
cp /opt/eqemu/zones/mycustomzone.eqg .
cp /opt/eqemu/zones/mycustomzone_chr.txt .

# Regenerate
./update-patches.sh
```

### What players see:

**LaunchPad Window:**
```
┌─────────────────────────────────┐
│   EverQuest LaunchPad           │
├─────────────────────────────────┤
│                                 │
│   EverQuest Emulator Server     │
│                                 │
│   Downloading files... (2/3)    │
│   ████████████░░░░░░░  66%      │
│                                 │
│        [    PLAY    ]           │
│        [    Exit    ]           │
│                                 │
└─────────────────────────────────┘
```

Process:
1. Checks manifest
2. Downloads updated files with progress bar
3. Click PLAY when ready
4. Game launches!

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
