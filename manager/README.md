# EQ Patch Manager

A companion desktop application for the Simple EQ Patcher system. Allows server admins to easily manage patch files and news via an intuitive GUI.

## Features

### 1. SSH/SFTP Connection Management
- **Secure Connection**: Connect to your patch server via SSH
- **Credential Storage**: Save connection profiles for quick access
- **Connection Testing**: Test connectivity before uploading
- **Remote Path Validation**: Automatically verifies access to patch directory

### 2. File Upload Manager
- **Dual-Pane Interface**:
  - Left: Local file browser
  - Right: Remote EQ folder structure
- **Smart File Placement**: Automatically determines correct folder based on file extension
  - `spells_us.txt`, `dbg.txt` → Root
  - `EQUI_*.xml`, `*.tga` → `uifiles/default/`
  - `*.s3d`, `*.eqg` → `Resources/`
  - `*.txt` (maps) → `maps/`
  - `*.eff` → `SpellEffects/`
  - And more...
- **Upload Queue**: Add multiple files before uploading
- **Progress Tracking**: Visual progress bar for each upload
- **Resumable Uploads**: Automatically resumes if connection drops
- **Batch Operations**: Upload entire folders at once

### 3. News Editor
- **Rich Text Formatting**:
  - **Bold** text support
  - *Italic* text support
  - Color selection (9 preset colors)
- **News Rotation**:
  - Configurable rotation time (seconds)
  - Smooth fade transitions
  - Enable/disable news globally
- **Live Preview**: See how news will appear in launcher
- **Multiple Items**: Create multiple news items that rotate
- **Easy Publishing**: One-click publish to server

### 4. Manifest Management
- **View Manifest**: Load and view current manifest.json
- **Remote Rebuild**: Execute manifest-builder on server via SSH
- **Summary View**: See file counts and total size
- **JSON View**: View raw manifest data

## Installation

### Prerequisites
- Windows PC
- SSH access to your patch server
- EQ Patch Server already installed (see main README)

### Setup
1. Download `manager.exe` from your server:
   ```
   http://YOUR_SERVER/eq-patches/manager.exe
   ```

2. Double-click `manager.exe` to launch

3. No installation needed - it's a portable executable

## Usage Guide

### First Time Setup

1. **Connect to Server**:
   - Click "Connection" tab
   - Enter your server details:
     - Host: Your server IP or domain
     - Port: 22 (default SSH port)
     - Username: Your SSH username (usually `root`)
     - Password: Your SSH password
     - Remote Path: `/var/www/html/eq-patches`
   - Click "Connect"
   - Click "Save Profile" to remember settings

2. **Test Connection**:
   - Click "Test Connection" to verify SSH commands work
   - Should see output of `pwd && whoami`

### Uploading Files

1. **Select Files**:
   - Go to "File Upload" tab
   - Click "Select Files" or "Select Folder"
   - Browse to your custom EQ files
   - Supported types: `.txt`, `.xml`, `.tga`, `.s3d`, `.eqg`, `.eff`, `.wav`

2. **Choose Destination** (Optional):
   - Select destination folder from dropdown
   - Or leave blank for automatic placement

3. **Add to Queue**:
   - Click "Add to Upload Queue"
   - Files appear in upload queue with status

4. **Upload**:
   - Review files in queue
   - Click "Start Upload"
   - Watch progress for each file
   - Files upload with resume support

5. **Rebuild Manifest**:
   - Go to "Manifest" tab
   - Click "Rebuild Manifest"
   - Manifest is regenerated with new files

### Managing News

1. **Create News Item**:
   - Go to "News Editor" tab
   - Type your message in the editor
   - Select color (optional)
   - Check "Bold" or "Italic" (optional)
   - Click "Add News Item"

2. **Manage Items**:
   - See all news items in right panel
   - Remove unwanted items
   - Set rotation time (seconds between items)

3. **Publish**:
   - Click "Publish to Server"
   - News.json is uploaded to patch server
   - Players see news next time they launch

### Example News Messages
```
"Welcome to Aradune's Revenge!"
"Server maintenance tonight at 9 PM EST"
"New custom zone: Mistmoore Keep"
"Join us on Discord: discord.gg/yourserver"
```

## File Organization

The manager mirrors the EQ client folder structure:

```
/var/www/html/eq-patches/
├── spells_us.txt          ← Root level files
├── dbg.txt
├── uifiles/               ← UI files
│   └── default/
│       ├── EQUI_*.xml
│       └── *.tga
├── Resources/             ← Zone files
│   ├── customzone.eqg
│   └── customzone.s3d
├── maps/                  ← Map files
│   └── *.txt
├── SpellEffects/          ← Spell effects
│   └── *.eff
└── news.json              ← News config
```

**IMPORTANT**: Only upload files that are **unique to your server**, not the entire EQ client!

## Workflow Example

### Adding a Custom Zone

1. Connect to server
2. Upload files:
   - `customzone.eqg` → `Resources/`
   - `customzone_chr.txt` → Root
   - `customzone.txt` (map) → `maps/`
3. Rebuild manifest
4. Create news: "New custom zone available: Custom Zone!"
5. Publish news
6. Players get updates next launcher run

### Updating Spells

1. Connect to server
2. Upload `spells_us.txt` → Root (auto-detected)
3. Rebuild manifest
4. Players get update automatically

## Troubleshooting

### Connection Failed
- Check server IP/hostname
- Verify SSH port (default 22)
- Confirm username/password
- Check firewall allows SSH (port 22)

### Upload Failed
- Check file permissions on server
- Verify remote path exists
- Ensure sufficient disk space
- Try smaller files first

### News Not Showing
- Verify news.json was uploaded
- Check LaunchPad has news fader enabled
- Check server URL in patcher-config.json
- Test with: `curl http://YOUR_SERVER/eq-patches/news.json`

### Manifest Not Updating
- Verify manifest-builder exists on server
- Check execute permissions
- Try running manually: `cd /var/www/html/eq-patches && ./manifest-builder .`

## Advanced

### Multiple Servers
Create different connection profiles for test/live servers:
- Save Profile as "Test Server"
- Save Profile as "Live Server"
- Load profiles as needed

### Batch Operations
1. Organize files locally in folders matching EQ structure
2. Use "Select Folder" to add entire directory
3. Files automatically route to correct remote folders

### Resume Failed Uploads
If upload fails:
1. Don't clear queue
2. Click "Start Upload" again
3. Manager resumes from last byte uploaded

## Building from Source

```bash
cd manager
go mod tidy
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc \
  go build -ldflags="-H windowsgui" -o manager.exe .
```

Requires:
- Go 1.24+
- mingw-w64 for Windows builds

## Technical Details

**Language**: Go + Fyne v2 (GUI framework)
**Protocols**: SSH, SFTP
**Size**: ~30-40 MB (includes GUI framework)
**Platform**: Windows (Linux support planned)

**Libraries**:
- `golang.org/x/crypto/ssh` - SSH client
- `github.com/pkg/sftp` - SFTP file transfer
- `fyne.io/fyne/v2` - GUI framework

## Support

For issues or questions:
1. Check troubleshooting section above
2. Verify server setup (main README)
3. Test with CLI tools first
4. Check server logs

## License

Free to use and modify for your EverQuest server.

---

**Part of the Simple EQ Patcher project**
Repository: https://github.com/crucifix86/simple-eq-patcher
