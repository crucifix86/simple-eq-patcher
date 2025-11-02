# Simple EverQuest Patcher

A lightweight, manifest-based patcher for EverQuest servers that mirrors your client directory structure.

## 🎯 Design Philosophy

Unlike complex patchers with YAML configs and client types, this patcher is **dead simple**:

- **Server Side**: Directory structure mirrors the EQ client exactly
- **Manifest-based**: Single `manifest.json` tracks all files with MD5 hashes
- **Pure HTTP**: No special protocols, just standard HTTP downloads
- **Cross-platform**: Written in Go, compiles to single executable

## 📁 Architecture

### Server Structure
```
/var/www/eq-patches/
├── manifest.json          ← Generated file list with MD5 hashes
├── spells_us.txt          ← Files in client root
├── dbg.txt
├── Resources/             ← Subdirectories mirror client
│   └── SomeFile.txt
├── UI/
│   └── default/
│       └── ui_file.xml
└── [any other files/dirs]
```

### Client Behavior
1. Downloads `manifest.json` from server
2. Compares each file in manifest with local files
3. Downloads only files that are:
   - Missing
   - Wrong size
   - Different MD5 hash
4. Launches the game

## 🚀 Quick Start

### Server Setup

1. **Create patch directory** (mirrors your EQ client structure):
   ```bash
   mkdir -p /var/www/eq-patches
   cd /var/www/eq-patches
   ```

2. **Copy files you want to patch** (keep directory structure):
   ```bash
   # Example: Copy custom spells file
   cp /path/to/custom/spells_us.txt .

   # Example: Copy custom zone files
   cp /path/to/zones/*.eqg .

   # Example: Copy UI files (maintain structure)
   mkdir -p UI/default
   cp /path/to/ui/EQUI_*.xml UI/default/
   ```

3. **Generate manifest**:
   ```bash
   /home/doug/simple-eq-patcher/server/manifest-builder /var/www/eq-patches
   ```

   Output:
   ```
   Scanning directory: /var/www/eq-patches
     Added: spells_us.txt (123456 bytes, md5: a1b2c3d4)
     Added: customzone.eqg (987654 bytes, md5: e5f6g7h8)
     Added: UI/default/EQUI_Inventory.xml (45678 bytes, md5: i9j0k1l2)

   ✓ Manifest created: /var/www/eq-patches/manifest.json
   ✓ Total files: 3
   ```

4. **Serve via HTTP**:

   **Option A: Nginx** (recommended for production)
   ```nginx
   server {
       listen 80;
       server_name yourserver.com;

       location /patches {
           alias /var/www/eq-patches;
           autoindex off;
       }
   }
   ```

   **Option B: Quick test with Go**
   ```bash
   cd /var/www/eq-patches
   python3 -m http.server 8080
   # or
   go run -m http.server
   ```

### Client Setup

1. **Copy patcher to EQ directory**:
   ```
   C:\EverQuest\
   ├── eqgame.exe
   ├── patcher.exe          ← Your new patcher
   └── patcher-config.json  ← Auto-created on first run
   ```

2. **Run patcher once** to create config:
   ```
   patcher.exe
   ```

   It will create `patcher-config.json` and exit.

3. **Edit patcher-config.json**:
   ```json
   {
     "server_url": "http://yourserver.com/patches",
     "game_exe": "eqgame.exe",
     "game_args": "patchme"
   }
   ```

4. **Run patcher** - it will:
   - Download manifest
   - Check all files
   - Download any missing/changed files
   - Launch the game

## 📖 Usage Examples

### Example 1: Add Custom Spells

**On Server:**
```bash
cd /var/www/eq-patches
cp /opt/eqemu/server/spells_us.txt .
/home/doug/simple-eq-patcher/server/manifest-builder /var/www/eq-patches
```

**Players:** Run `patcher.exe` - automatically downloads new spells

### Example 2: Add Custom Zone

**On Server:**
```bash
cd /var/www/eq-patches
cp /opt/eqemu/zones/customzone.eqg .
cp /opt/eqemu/zones/customzone_chr.txt .
/home/doug/simple-eq-patcher/server/manifest-builder /var/www/eq-patches
```

**Players:** Run `patcher.exe` - downloads zone files

### Example 3: Update Multiple Files

**On Server:**
```bash
cd /var/www/eq-patches

# Update spells
cp /opt/eqemu/server/spells_us.txt .

# Update zones
cp /opt/eqemu/zones/*.eqg .

# Update UI files (maintain directory structure)
mkdir -p UI/default
cp /opt/eqemu/ui/*.xml UI/default/

# Regenerate manifest
/home/doug/simple-eq-patcher/server/manifest-builder /var/www/eq-patches
```

**Players:** Run `patcher.exe` - downloads only changed files

## 🔧 Building from Source

```bash
cd /home/doug/simple-eq-patcher
./build.sh
```

Output:
- `server/manifest-builder` - Linux binary for server
- `client/patcher.exe` - Windows binary for players
- `client/patcher-linux` - Linux binary for testing

## 📋 Workflow

### Initial Setup
1. Create patch directory on server
2. Build patcher tools
3. Configure web server

### Daily Updates
1. Copy updated files to `/var/www/eq-patches` (maintain structure)
2. Run `manifest-builder /var/www/eq-patches`
3. Done! Players auto-download on next patcher run

### Player Experience
1. Download `patcher.exe` from your server
2. Place in EQ directory
3. Run once to configure
4. Run anytime to patch and play

## 🔍 How It Works

### Manifest Structure
```json
{
  "version": "1.0",
  "files": [
    {
      "path": "spells_us.txt",
      "md5": "a1b2c3d4e5f6...",
      "size": 123456
    },
    {
      "path": "UI/default/EQUI_Inventory.xml",
      "md5": "1a2b3c4d5e6f...",
      "size": 45678
    }
  ]
}
```

### Patcher Logic
1. Download `manifest.json`
2. For each file in manifest:
   - Check if exists locally
   - If exists: compare size, then MD5
   - If missing/different: add to download queue
3. Download queued files
4. Launch game

## ✅ Advantages

- **Simple**: No complex configs, just mirror your directory structure
- **Efficient**: Only downloads changed files (MD5 comparison)
- **Reliable**: Hash-based verification prevents corruption
- **Fast**: Single manifest file, pure HTTP downloads
- **Portable**: Single .exe, no dependencies
- **Transparent**: Players see exactly what's being patched

## 🆚 Comparison with Other Patchers

| Feature | Simple Patcher | Complex Patcher |
|---------|---------------|-----------------|
| Server setup | Copy files + run builder | YAML configs, client types |
| Directory structure | Mirrors client | Custom structure |
| Manifest | Single JSON | Multiple YAML files |
| Client config | 3-line JSON | Complex settings |
| Dependencies | None | .NET Framework, libraries |
| Size | ~2MB | ~10-20MB |
| Cross-platform | Yes (Go) | Windows only (C#) |

## 🛠️ Advanced Configuration

### Custom Game Launch Arguments

Edit `patcher-config.json`:
```json
{
  "server_url": "http://yourserver.com/patches",
  "game_exe": "eqgame.exe",
  "game_args": "patchme /login:loginserver.com"
}
```

### Multiple Patch Servers

Create different config files:
- `patcher-config-test.json`
- `patcher-config-live.json`

Copy and rename as needed.

### Exclude Files from Manifest

The manifest builder automatically excludes:
- Directories (only files are added)
- `manifest.json` itself

To exclude more, modify `manifest-builder.go`:
```go
// Skip unwanted files
if strings.Contains(path, "temp") || strings.HasSuffix(path, ".log") {
    return nil
}
```

## 📝 Troubleshooting

**Patcher can't connect:**
- Check `server_url` in config
- Test manually: `http://yourserver.com/patches/manifest.json`
- Check firewall allows port 80/443

**Files not updating:**
- Regenerate manifest: `manifest-builder /var/www/eq-patches`
- Check file permissions on server
- Delete local file and re-run patcher

**Game won't launch:**
- Check `game_exe` path in config
- Verify `eqgame.exe` exists
- Check `game_args` are correct

## 🎓 Example: Complete Setup

**Server (Linux):**
```bash
# 1. Create patch directory
sudo mkdir -p /var/www/eq-patches
sudo chown $USER:$USER /var/www/eq-patches

# 2. Copy your custom files
cd /var/www/eq-patches
cp ~/everquest_rof2/spells_us.txt .
cp ~/everquest_rof2/globalload.txt .

# 3. Generate manifest
/home/doug/simple-eq-patcher/server/manifest-builder /var/www/eq-patches

# 4. Serve via nginx (already configured at port 80)
curl http://localhost/patches/manifest.json
```

**Client (Windows):**
```
1. Copy patcher.exe to C:\EverQuest\
2. Run patcher.exe (creates config)
3. Edit patcher-config.json:
   {
     "server_url": "http://84.46.251.4/patches",
     "game_exe": "eqgame.exe",
     "game_args": "patchme"
   }
4. Run patcher.exe
5. Play!
```

## 📦 Distribution

**What to give players:**
- `patcher.exe` (single file)
- `patcher-config.json` (pre-configured with your server URL)

**Or:**
- Just `patcher.exe`
- Instructions to edit config on first run

## 🔐 Security Notes

- Uses HTTP (add HTTPS in nginx for encryption)
- MD5 for file integrity (not cryptographic security)
- No authentication (add nginx basic auth if needed)

## 📜 License

Free to use and modify for your EverQuest server.

---

**That's it!** Dead simple patcher that just works.
