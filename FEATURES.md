# LaunchPad Features

## New in Latest Version

### Graphics Settings Button
Configure EverQuest graphics compatibility settings with ease:

**Guided Setup:**
- Full Compatibility (Fullscreen) - Disables DWM, DPI override
- DPI Awareness Only (Borderless Windowed)
- Remove All Compatibility Settings
- Show Current Settings

Applies Windows registry settings automatically - no manual editing required!

**Manual INI Edit:**
- View all EverQuest configuration files
- Edit eqclient.ini, eqgame.ini, character INIs
- Built-in text editor with syntax preservation
- Direct access to all graphics and game settings

### Configurable Website Button
Add a custom website button to your launcher!

Edit `patcher-config.json`:
```json
{
  "website_url": "https://www.yourserver.com",
  "website_label": "Visit Forums"
}
```

Opens player's default browser to your website, forums, or Discord.

### Customizable Launcher Title
Brand the launcher with your server name!

```json
{
  "launcher_title": "My EQ Server LaunchPad"
}
```

Window title changes to match your server branding.

## Configuration

### patcher-config.json

```json
{
  "server_url": "http://yourserver.com/patches",
  "server_name": "My EverQuest Server",
  "launcher_title": "EverQuest LaunchPad",
  "website_url": "https://www.myeqserver.com",
  "website_label": "Visit Website",
  "game_exe": "eqgame.exe",
  "game_args": "patchme"
}
```

**Fields:**
- `server_url` - URL where patch files are hosted
- `server_name` - Displayed in launcher window (under title)
- `launcher_title` - Window title bar text
- `website_url` - URL for website button (set to empty "" to hide button)
- `website_label` - Button text for website
- `game_exe` - EverQuest executable name
- `game_args` - Command line arguments (use "patchme" for emuservers)

## Graphics Settings Details

### Full Compatibility Mode
Best for traditional fullscreen on modern Windows:
- Disables Desktop Window Manager (DWM)
- Sets DPI Unaware mode (prevents scaling artifacts)
- Enables High DPI Aware flag
- Fixes fullscreen behavior on high-DPI displays

**Registry Location:**
`HKCU\Software\Microsoft\Windows NT\CurrentVersion\AppCompatFlags\Layers`

**Registry Value:**
`~ DISABLEDXMAXIMIZEDWINDOWEDMODE DPIUNAWARE HIGHDPIAWARE`

### DPI Awareness Only
Best for borderless windowed mode:
- Enables High DPI Aware flag only
- Keeps Desktop Window Manager enabled
- Better for multi-monitor setups
- Smoother alt-tabbing

**Registry Value:**
`~ HIGHDPIAWARE`

### Manual INI Files

**eqclient.ini** - Main graphics settings:
```ini
[VideoMode]
Width=1920
Height=1080
RefreshRate=60
Windowed=FALSE
```

**Common Settings:**
- `Width`, `Height` - Screen resolution
- `Windowed` - TRUE for windowed, FALSE for fullscreen
- `RefreshRate` - Monitor refresh rate
- `AntiAliasing` - 0-8 for AA levels

**eqgame.ini** - Game behavior:
```ini
[Defaults]
ShowGrassDetail=1
MaxFPS=60
```

**dbg.txt** - Debug/performance:
```
NoFlash=TRUE
NoAlpha=FALSE
```

## Player Instructions

### Using Graphics Settings

1. Click "Graphics Settings" button in LaunchPad
2. Choose "Guided Setup" or "Manual INI Edit"
3. **Guided**: Select compatibility mode and click Apply
4. **Manual**: Select file, edit settings, click Save Changes
5. Restart game for changes to take effect

### Common Issues

**Fullscreen won't work on modern Windows:**
- Use "Full Compatibility" guided setup
- This disables Windows DWM for the game

**Game is blurry on high-DPI monitor:**
- Use "DPI Awareness Only" setting
- Or manually edit DPI scaling in Windows compatibility properties

**Want to reset everything:**
- Use "Remove All Compatibility Settings"
- Or delete values from Registry Editor manually

## For Server Admins

### Customizing the Website Button

**Example: Discord Server**
```json
{
  "website_url": "https://discord.gg/yourserver",
  "website_label": "Join Discord"
}
```

**Example: Forums**
```json
{
  "website_url": "https://forums.yourserver.com",
  "website_label": "Visit Forums"
}
```

**Hide the button:**
```json
{
  "website_url": "",
  "website_label": ""
}
```

### Branding

```json
{
  "server_name": "Project 1999",
  "launcher_title": "Project 1999 LaunchPad"
}
```

Players will see "Project 1999" in the launcher window with "Project 1999 LaunchPad" in the title bar.

## Technical Notes

- Graphics settings use Windows Registry API (Windows only)
- Website button uses system default browser
- INI editor preserves file formatting
- All changes require game restart to take effect
- Compatible with all EverQuest client versions
