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
/var/www/html/eq-patches/
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

## 🚀 Quick Start (Ubuntu 24.04)

### 1. Install on Your Server

**First time install:**
```bash
git clone https://github.com/crucifix86/simple-eq-patcher.git
cd simple-eq-patcher
./install.sh
```

**Update to latest version:**
```bash
cd simple-eq-patcher
./install.sh --update
```
This pulls latest from git and rebuilds everything automatically!

**Done!** The installer automatically:
- Installs Go, nginx, and zip
- Builds server manifest-builder tool
- Copies pre-compiled client executables
- Creates `/var/www/html/eq-patches`
- Configures nginx
- Opens firewall port 80

### 2. Add Your Custom Files

**⚠️ IMPORTANT:** Only copy files that are **unique to YOUR server** - not the whole EQ client!

```bash
cd /var/www/html/eq-patches

# Copy ONLY your custom/modified files
cp /path/to/custom/spells_us.txt .
cp /path/to/custom/dbg.txt .

# Custom zones (only ones you added/modified)
cp /path/to/custom/customzone.eqg .
cp /path/to/custom/customzone_chr.txt .

# Custom UI files (maintain directory structure)
mkdir -p UI/default
cp /path/to/custom/EQUI_*.xml UI/default/
```

**What NOT to copy:**
- ❌ The entire EQ client (players already have this)
- ❌ Standard/vanilla zone files
- ❌ Game executables (eqgame.exe, etc.)
- ❌ Anything unchanged from retail

### 3. Generate Manifest

```bash
cd /var/www/html/eq-patches
./update-patches.sh
```

Output:
```
Scanning directory: /var/www/html/eq-patches
  Added: spells_us.txt (123456 bytes, md5: a1b2c3d4)
  Added: customzone.eqg (987654 bytes, md5: e5f6g7h8)

✓ Manifest created: /var/www/html/eq-patches/manifest.json
✓ Total files: 2
```

### 4. Test Your Patch Server

```bash
curl http://YOUR_SERVER_IP/eq-patches/manifest.json
```

### 5. Distribute to Players

**Give players the client bundle (easiest):**
- Download: `http://YOUR_SERVER_IP/eq-patches/eq-patcher-client.zip`
- Contains everything they need:
  - `LaunchPad.exe` - GUI launcher
  - `patcher.exe` - CLI fallback
  - `patcher-config.json` - Pre-configured for your server

**Or individual files from `/var/www/html/eq-patches/`:**
1. `LaunchPad.exe` - GUI launcher (recommended)
2. `patcher-config.json` - Configuration file
3. `patcher.exe` - CLI fallback (optional)

**Player instructions:**
1. Download `eq-patcher-client.zip` from your server
2. Extract the ZIP to their EverQuest folder (same folder as `eqgame.exe`)
3. (Optional) Edit `patcher-config.json` to customize settings
4. Double-click `LaunchPad.exe`
5. Click "PLAY" button to patch and launch game
6. Use "Graphics Settings" button to configure display options
7. Click "Compatibility Fix Wizard" if having fullscreen/DPI issues
8. Click website button to visit server Discord/forums

**Features:**
- ✅ Auto-update check on startup (not when clicking play!)
- ✅ Launcher self-update detection (notifies when new version available)
- ✅ Graphics Settings with resolution, texture quality, effects
- ✅ Compatibility Fix Wizard (fixes fullscreen/DPI issues on modern Windows)
- ✅ Configurable website button (Discord, forums, etc.)
- ✅ Customizable launcher title and server name
- ✅ EverQuest-themed UI with background image
- ✅ Progress bar shows connection, checking, downloading stages

**Note:** `LaunchPad.exe` is a full-featured GUI application. If players have issues, they can use the CLI `patcher.exe` instead.

## 📖 Daily Usage

### When You Update Your Server Files

```bash
cd /var/www/html/eq-patches

# Copy updated files
cp /opt/eqemu/server/spells_us.txt .

# Regenerate manifest
./update-patches.sh
```

**Done!** Players automatically get updates next time they run the patcher.

### Example: Updated Spells and Added Custom Zone

```bash
cd /var/www/html/eq-patches

# Updated spells file
cp /opt/eqemu/server/spells_us.txt .

# New custom zone files (only copy YOUR custom zones!)
cp /opt/eqemu/zones/mycustomzone.eqg .
cp /opt/eqemu/zones/mycustomzone_chr.txt .
cp /opt/eqemu/zones/mycustomzone.zon .

# Regenerate manifest
./update-patches.sh
```

Next time players run `patcher.exe`:
- Downloads new spells_us.txt
- Downloads 3 new zone files
- Launches game

## 🔧 Building from Source (Developers Only)

**Note:** Pre-compiled executables are included in the repository. You only need to build if you're modifying the code.

**Requirements:**
- Go compiler
- mingw-w64 cross-compiler: `sudo apt-get install gcc-mingw-w64-x86-64`

```bash
cd /home/doug/simple-eq-patcher
./build.sh
```

Output:
- `server/manifest-builder` - Linux binary for server
- `client/LaunchPad.exe` - Windows GUI launcher (23MB, includes graphics)
- `client/patcher.exe` - Windows CLI fallback (7MB, command-line)
- `client/patcher-linux` - Linux binary for testing

## 📋 Workflow

### One-Time Setup
1. Run `./install.sh` on your Ubuntu server - **DONE!**
2. Copy your custom files to `/var/www/html/eq-patches`
3. Run `./update-patches.sh`
4. Give `LaunchPad.exe` + `patcher-config.json` to players

### Daily Updates (When You Change Server Files)
1. Copy updated files to `/var/www/html/eq-patches`
2. Run `./update-patches.sh`
3. Players auto-download on next launcher run

### Player Experience
1. Copy `LaunchPad.exe` + `patcher-config.json` to EverQuest folder
2. Double-click `LaunchPad.exe` whenever they want to play
3. Launcher shows progress, downloads your custom files, and launches game

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

### Customizing the Launcher

Edit `patcher-config.json`:
```json
{
  "server_url": "http://yourserver.com/eq-patches",
  "server_name": "My EverQuest Server",
  "launcher_title": "My Server - LaunchPad",
  "website_url": "https://discord.gg/yourserver",
  "website_label": "Join Discord",
  "game_exe": "eqgame.exe",
  "game_args": "patchme"
}
```

**Configuration Options:**
- `server_url` - Patch server URL (where manifest.json is located)
- `server_name` - Displayed in launcher (under EverQuest title)
- `launcher_title` - Window title for the launcher
- `website_url` - URL for the website button (Discord, forums, etc.)
- `website_label` - Text shown on website button
- `game_exe` - Game executable name (usually eqgame.exe)
- `game_args` - Launch arguments (e.g., "patchme" or "patchme /login:loginserver.com")

### Custom Game Launch Arguments

For custom login servers:
```json
{
  "game_args": "patchme /login:loginserver.myserver.com"
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
- Test manually: `http://yourserver.com/eq-patches/manifest.json`
- Check firewall allows port 80/443

**Files not updating:**
- Regenerate manifest: `manifest-builder /var/www/html/eq-patches`
- Check file permissions on server
- Delete local file and re-run patcher

**Game won't launch:**
- Check `game_exe` path in config
- Verify `eqgame.exe` exists
- Check `game_args` are correct

## 🎓 Complete Example Setup

### On Your Ubuntu Server:

```bash
# 1. Install (one command!)
git clone https://github.com/crucifix86/simple-eq-patcher.git
cd simple-eq-patcher
./install.sh

# 2. Copy YOUR custom files only
cd /var/www/html/eq-patches
cp /opt/eqemu/server/spells_us.txt .
cp /opt/eqemu/server/dbg.txt .

# 3. Generate manifest
./update-patches.sh

# 4. Test it works
curl http://YOUR_SERVER_IP/eq-patches/manifest.json
```

### Give to Players:

- `LaunchPad.exe` (GUI launcher from `/var/www/html/eq-patches/LaunchPad.exe`)
- `patcher-config.json` (from `/var/www/html/eq-patches/patcher-config.json`)
- `patcher.exe` (CLI fallback - optional)

Players copy `LaunchPad.exe` and `patcher-config.json` to their `C:\EverQuest\` folder and double-click `LaunchPad.exe`!

## 📦 Distribution

**Give players ONE file (easiest):**
- `eq-patcher-client.zip` - Download from `http://YOUR_SERVER_IP/eq-patches/eq-patcher-client.zip`

**Contains everything:**
- `LaunchPad.exe` - Full-featured GUI launcher
- `patcher.exe` - CLI fallback (for troubleshooting)
- `patcher-config.json` - Pre-configured with your server settings

**Players just:**
1. Download the ZIP
2. Extract to `C:\EverQuest\` (or their EQ folder)
3. Double-click `LaunchPad.exe`

**Result:**
```
C:\EverQuest\
├── eqgame.exe (their existing game)
├── LaunchPad.exe (your launcher - replaces retail)
├── patcher.exe (CLI fallback)
└── patcher-config.json (server config)
```

**LaunchPad Features:**
- ✨ Auto-patching with progress bar
- 🎮 Graphics Settings (resolution, textures, effects, shaders)
- 🔧 Compatibility Fix Wizard (DPI/fullscreen issues)
- 🌐 Configurable website button (Discord/forums)
- 🎨 EverQuest-themed UI with background
- 🏷️ Customizable server branding
- 📐 Clean layout with proper button spacing

Just double-click `LaunchPad.exe` to patch and play!

## 🎮 Player Features

### Graphics Settings Menu
- **Resolution Selector**: Common resolutions from 800x600 to 4K
- **Fullscreen Toggle**: Enable/disable fullscreen mode
- **Texture Quality**: Low, Medium, High, Ultra
- **Spell Effects**: Off, Low, Medium, High
- **Advanced Options**: Grass, Dynamic Lighting, Vertex/Pixel Shaders
- **Reset to Defaults**: One-click restoration of recommended settings

### Compatibility Fix Wizard
Having issues with fullscreen or DPI scaling on modern Windows? The Compatibility Fix Wizard can help!

**Full Compatibility (Best for Fullscreen)**
- Disables DX Maximized Windowed Mode
- Sets DPI Unaware mode (prevents scaling)
- Enables High DPI Aware flag
- Best for traditional fullscreen gaming

**DPI Awareness Only (Best for Windowed)**
- Enables High DPI Aware flag only
- Works best with borderless windowed mode
- Less aggressive than full compatibility

**Remove All Settings**
- Restores Windows default behavior
- Use if compatibility fixes cause issues

## 🔐 Security Notes

- Uses HTTP (add HTTPS in nginx for encryption)
- MD5 for file integrity (not cryptographic security)
- No authentication (add nginx basic auth if needed)

## 📜 License

Free to use and modify for your EverQuest server.

---

**That's it!** Dead simple patcher that just works.
