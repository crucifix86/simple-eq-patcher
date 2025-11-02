package main

import (
	"crypto/md5"
	"encoding/json"
	_ "embed"
	"fmt"
	"image/color"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

//go:embed loadscreen.jpg
var backgroundImage []byte

type FileEntry struct {
	Path string `json:"path"`
	MD5  string `json:"md5"`
	Size int64  `json:"size"`
}

type Manifest struct {
	Version string      `json:"version"`
	Files   []FileEntry `json:"files"`
}

type Config struct {
	ServerURL     string `json:"server_url"`
	ServerName    string `json:"server_name"`
	LauncherTitle string `json:"launcher_title"`
	WebsiteURL    string `json:"website_url"`
	WebsiteLabel  string `json:"website_label"`
	GameExe       string `json:"game_exe"`
	GameArgs      string `json:"game_args"`
}

type NewsItem struct {
	Text      string            `json:"text"`
	Formatted string            `json:"formatted"`
	Color     string            `json:"color"`
	Style     map[string]string `json:"style"`
}

type NewsConfig struct {
	Items          []*NewsItem `json:"items"`
	RotationTime   int         `json:"rotation_time"`
	FadeTime       float64     `json:"fade_time"`
	Enabled        bool        `json:"enabled"`
	BackgroundBlur bool        `json:"background_blur"`
}

const (
	configFile        = "patcher-config.json"
	localManifestFile = ".patcher-manifest.json"
)

// Directories managed by the patcher (these mirror the EQ client structure)
var managedDirs = []string{
	"ActorEffects",
	"Atlas",
	"AudioTriggers",
	"EnvEmitterEffects",
	"RenderEffects",
	"Resources",
	"SpellEffects",
	"help",
	"maps",
	"sounds",
	"storyline",
	"uifiles",
	"userdata",
	"voice",
}

var (
	config      *Config
	statusLabel *widget.Label
	progressBar *widget.ProgressBar
	playButton  *widget.Button
	exitButton  *widget.Button
)

func main() {
	myApp := app.New()

	// Get launcher directory and change to it
	// This ensures all file operations happen in the correct location
	exePath, err := os.Executable()
	if err == nil {
		launcherDir := filepath.Dir(exePath)
		os.Chdir(launcherDir)
	}

	// Load configuration first to get launcher title
	config, err = loadConfig()
	if err != nil {
		config = createDefaultConfig()
	}

	// Use configurable title
	windowTitle := config.LauncherTitle
	if windowTitle == "" {
		windowTitle = "EverQuest LaunchPad"
	}
	myWindow := myApp.NewWindow(windowTitle)

	// Load background image
	bg := canvas.NewImageFromReader(strings.NewReader(string(backgroundImage)), "background")
	bg.FillMode = canvas.ImageFillStretch

	// Create UI elements with EQ-style colors
	titleLabel := canvas.NewText("EverQuest", theme.ForegroundColor())
	titleLabel.TextSize = 28
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	titleLabel.Alignment = fyne.TextAlignCenter

	serverLabel := canvas.NewText(config.ServerName, theme.ForegroundColor())
	serverLabel.TextSize = 16
	serverLabel.Alignment = fyne.TextAlignCenter

	// Create news fader (centered, below server name)
	newsFader := createNewsFader(config.ServerURL)

	statusLabel = widget.NewLabel("Initializing...")
	statusLabel.Alignment = fyne.TextAlignCenter
	statusLabel.Importance = widget.MediumImportance
	statusLabel.TextStyle = fyne.TextStyle{Bold: true}

	progressBar = widget.NewProgressBar()
	progressBar.Hide()

	playButton = widget.NewButton("PLAY", func() {
		go launchGameOnly(myWindow)
	})
	playButton.Importance = widget.HighImportance
	playButton.Disable() // Disabled until update check completes

	exitButton = widget.NewButton("Exit", func() {
		myApp.Quit()
	})

	// Graphics Settings button (at top)
	graphicsButton := widget.NewButton("Graphics Settings", func() {
		showGraphicsDialog(myWindow)
	})

	// Website button (if configured)
	var websiteButton *widget.Button
	if config.WebsiteURL != "" {
		buttonLabel := config.WebsiteLabel
		if buttonLabel == "" {
			buttonLabel = "Visit Website"
		}
		websiteButton = widget.NewButton(buttonLabel, func() {
			openBrowser(config.WebsiteURL)
		})
	}

	// Create button layout per user specs:
	// - Graphics Settings at top (centered)
	// - PLAY button on left
	// - Website button (if configured) on left before Exit
	// - Exit button at bottom left
	// Add spacing between buttons for better visual layout
	leftButtons := container.NewVBox(
		playButton,
		layout.NewSpacer(),
	)
	if websiteButton != nil {
		leftButtons.Add(websiteButton)
		leftButtons.Add(layout.NewSpacer())
	}
	leftButtons.Add(exitButton)

	// Create centered content with better spacing
	centerContent := container.NewVBox(
		layout.NewSpacer(),
		container.NewCenter(titleLabel),
		container.NewCenter(serverLabel),
		container.NewCenter(newsFader),
		layout.NewSpacer(),
		layout.NewSpacer(),
		container.NewCenter(statusLabel),
		container.NewCenter(progressBar),
		layout.NewSpacer(),
	)

	// Create overlay container with new layout
	overlay := container.NewBorder(
		// Top: Graphics Settings button
		container.NewCenter(graphicsButton),
		// Bottom: empty
		nil,
		// Left: Play, Website (optional), Exit buttons
		container.NewVBox(
			layout.NewSpacer(),
			leftButtons,
			layout.NewSpacer(),
		),
		// Right: empty
		nil,
		// Center: Title, Status, Progress
		centerContent,
	)

	// Stack background and overlay
	content := container.NewStack(bg, overlay)

	myWindow.SetContent(content)
	myWindow.Resize(fyne.NewSize(600, 400))
	myWindow.SetFixedSize(true)
	myWindow.CenterOnScreen()

	// Check for updates on startup
	go checkForUpdatesOnStartup(myWindow)

	myWindow.ShowAndRun()
}

func checkForUpdatesOnStartup(win fyne.Window) {
	statusLabel.SetText("Checking for updates...")
	progressBar.Show()
	progressBar.SetValue(0)

	// Download manifest
	manifest, err := downloadManifest(config.ServerURL)
	if err != nil {
		// Can't connect - allow playing anyway
		statusLabel.SetText("⚠️ Update check failed - Ready to play")
		progressBar.Hide()
		playButton.Enable()
		return
	}

	// Check if launcher itself needs updating
	launcherNeedsUpdate := checkLauncherUpdates(manifest)
	if launcherNeedsUpdate {
		progressBar.Hide()
		statusLabel.SetText("📦 Launcher update available")

		dialog.ShowInformation(
			"Launcher Update Available",
			fmt.Sprintf("A new version of the launcher is available!\n\nPlease download the latest version:\n%s/eq-patcher-client.zip\n\nExtract and replace your current files.", config.ServerURL),
			win,
		)
		// Continue checking game files anyway
	}

	statusLabel.SetText("Checking files...")
	progressBar.SetValue(0.3)

	// Check which files need updating
	toDownload := []FileEntry{}
	for _, file := range manifest.Files {
		localPath := file.Path

		// Check if file exists
		info, err := os.Stat(localPath)
		if os.IsNotExist(err) {
			toDownload = append(toDownload, file)
			continue
		}

		// Check size and hash
		if info.Size() != file.Size {
			toDownload = append(toDownload, file)
			continue
		}

		localMD5, err := calculateMD5(localPath)
		if err != nil || localMD5 != file.MD5 {
			toDownload = append(toDownload, file)
			continue
		}
	}

	// Check for obsolete files (files not in manifest)
	toDelete := findObsoleteFiles(manifest)

	progressBar.Hide()

	totalChanges := len(toDownload) + len(toDelete)

	if totalChanges > 0 {
		// Updates available - ask user
		changeMsg := ""
		if len(toDownload) > 0 && len(toDelete) > 0 {
			changeMsg = fmt.Sprintf("%d file(s) to update, %d file(s) to remove", len(toDownload), len(toDelete))
		} else if len(toDownload) > 0 {
			changeMsg = fmt.Sprintf("%d file(s) to update", len(toDownload))
		} else {
			changeMsg = fmt.Sprintf("%d file(s) to remove", len(toDelete))
		}

		statusLabel.SetText(fmt.Sprintf("📦 %d change(s) available", totalChanges))

		dialog.ShowConfirm(
			"Updates Available",
			fmt.Sprintf("%s\n\nWould you like to apply updates now?\n\n(You can also play without updating)", changeMsg),
			func(update bool) {
				if update {
					// Apply updates (download new files and remove obsolete ones)
					go performUpdate(win, manifest, toDownload, toDelete)
				} else {
					// Skip updates
					statusLabel.SetText("✓ Ready to play (updates skipped)")
					playButton.Enable()
				}
			},
			win,
		)
	} else {
		// No updates needed - save current manifest as our local record
		saveLocalManifest(manifest)
		statusLabel.SetText("✓ Up to date - Ready to play")
		playButton.Enable()
	}
}

func performUpdate(win fyne.Window, manifest *Manifest, toDownload []FileEntry, toDelete []string) {
	playButton.Disable()
	progressBar.Show()
	progressBar.SetValue(0)

	totalOperations := len(toDownload) + len(toDelete)
	currentOp := 0

	// Download new/updated files
	for _, file := range toDownload {
		progress := float64(currentOp) / float64(totalOperations)
		progressBar.SetValue(progress)
		statusLabel.SetText(fmt.Sprintf("📥 Downloading %s (%d/%d)", filepath.Base(file.Path), currentOp+1, totalOperations))

		err := downloadFile(config.ServerURL, file.Path)
		if err != nil {
			statusLabel.SetText("⚠️ Download failed")
			progressBar.Hide()
			showError(win, fmt.Sprintf("Failed to download %s: %v", file.Path, err))
			playButton.Enable()
			return
		}
		currentOp++
	}

	// Delete obsolete files
	for _, filePath := range toDelete {
		progress := float64(currentOp) / float64(totalOperations)
		progressBar.SetValue(progress)
		statusLabel.SetText(fmt.Sprintf("🗑️ Removing %s (%d/%d)", filepath.Base(filePath), currentOp+1, totalOperations))

		err := os.Remove(filePath)
		if err != nil {
			// Don't fail the entire update if we can't delete a file
			// Just log it and continue
			fmt.Printf("Warning: Could not delete %s: %v\n", filePath, err)
		}
		currentOp++
	}

	// Save the server manifest as our local record
	saveLocalManifest(manifest)

	progressBar.SetValue(1.0)
	statusLabel.SetText("✓ All files updated - Ready to play")
	progressBar.Hide()
	playButton.Enable()
}

func launchGameOnly(win fyne.Window) {
	playButton.Disable()
	statusLabel.SetText("🎮 Launching EverQuest...")

	err := launchGame(config)
	if err != nil {
		showError(win, fmt.Sprintf("Failed to launch game: %v", err))
		playButton.Enable()
		statusLabel.SetText("Ready to play")
		return
	}

	// Exit launcher
	os.Exit(0)
}

func performPatchAndLaunch(win fyne.Window) {
	playButton.Disable()
	statusLabel.SetText("Connecting to patch server...")
	progressBar.Show()
	progressBar.SetValue(0)

	// Download manifest
	manifest, err := downloadManifest(config.ServerURL)
	if err != nil {
		// Can't connect to patch server - ask if they want to play anyway
		statusLabel.SetText("⚠️ Connection failed")
		progressBar.Hide()

		manifestURL := strings.TrimRight(config.ServerURL, "/") + "/manifest.json"
		dialog.ShowConfirm(
			"Patch Server Unavailable",
			fmt.Sprintf("Could not connect to patch server:\n\nURL: %s\n\nError: %v\n\nWould you like to launch the game anyway?\n\n(You may be missing latest updates)", manifestURL, err),
			func(playAnyway bool) {
				if playAnyway {
					// Skip patching, just launch
					statusLabel.SetText("Launching EverQuest...")
					err := launchGame(config)
					if err != nil {
						showError(win, fmt.Sprintf("Failed to launch game: %v", err))
						playButton.Enable()
						statusLabel.SetText("Ready to play")
						return
					}
					// Exit launcher
					os.Exit(0)
				} else {
					// User chose not to play
					playButton.Enable()
					statusLabel.SetText("Ready to play")
				}
			},
			win,
		)
		return
	}

	statusLabel.SetText(fmt.Sprintf("✓ Connected - Checking %d files...", len(manifest.Files)))
	progressBar.SetValue(0.1)

	// Check files
	toDownload := []FileEntry{}
	for _, file := range manifest.Files {
		localPath := file.Path

		// Check if file exists
		info, err := os.Stat(localPath)
		if os.IsNotExist(err) {
			toDownload = append(toDownload, file)
			continue
		}

		// Check size and hash
		if info.Size() != file.Size {
			toDownload = append(toDownload, file)
			continue
		}

		localMD5, err := calculateMD5(localPath)
		if err != nil || localMD5 != file.MD5 {
			toDownload = append(toDownload, file)
			continue
		}
	}

	// Download files if needed
	if len(toDownload) > 0 {
		statusLabel.SetText(fmt.Sprintf("📥 Downloading %d file(s)...", len(toDownload)))
		progressBar.SetValue(0.2)

		for i, file := range toDownload {
			progress := 0.2 + (float64(i) / float64(len(toDownload)) * 0.7)
			progressBar.SetValue(progress)
			statusLabel.SetText(fmt.Sprintf("📥 Downloading %s (%d/%d)", filepath.Base(file.Path), i+1, len(toDownload)))

			err := downloadFile(config.ServerURL, file.Path)
			if err != nil {
				// Download failed - ask if they want to continue anyway
				statusLabel.SetText("⚠️ Download failed")
				progressBar.Hide()

				dialog.ShowConfirm(
					"Download Failed",
					fmt.Sprintf("Failed to download %s:\n\n%v\n\nWould you like to launch the game anyway?\n\n(Some files may be outdated or missing)", filepath.Base(file.Path), err),
					func(playAnyway bool) {
						if playAnyway {
							// Skip remaining downloads, just launch
							statusLabel.SetText("Launching EverQuest...")
							err := launchGame(config)
							if err != nil {
								showError(win, fmt.Sprintf("Failed to launch game: %v", err))
								playButton.Enable()
								statusLabel.SetText("Ready to play")
								return
							}
							// Exit launcher
							os.Exit(0)
						} else {
							// User chose not to play
							playButton.Enable()
							statusLabel.SetText("Ready to play")
						}
					},
					win,
				)
				return
			}
		}

		progressBar.SetValue(0.9)
		statusLabel.SetText("✓ All files updated!")
	} else {
		progressBar.SetValue(0.9)
		statusLabel.SetText("✓ All files up to date!")
	}

	// Launch game
	progressBar.SetValue(1.0)
	statusLabel.SetText("🎮 Launching EverQuest...")
	err = launchGame(config)
	if err != nil {
		showError(win, fmt.Sprintf("Failed to launch game: %v", err))
		playButton.Enable()
		progressBar.Hide()
		statusLabel.SetText("Ready to play")
		return
	}

	// Exit launcher
	statusLabel.SetText("✓ Game launched successfully!")
	os.Exit(0)
}

func loadConfig() (*Config, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func createDefaultConfig() *Config {
	config := &Config{
		ServerURL:     "http://example.com/patches",
		ServerName:    "EverQuest Emulator Server",
		LauncherTitle: "EverQuest LaunchPad",
		WebsiteURL:    "https://www.example.com",
		WebsiteLabel:  "Visit Website",
		GameExe:       "eqgame.exe",
		GameArgs:      "patchme",
	}

	data, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile(configFile, data, 0644)

	return config
}

func downloadManifest(serverURL string) (*Manifest, error) {
	url := strings.TrimRight(serverURL, "/") + "/manifest.json"

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var manifest Manifest
	err = json.NewDecoder(resp.Body).Decode(&manifest)
	if err != nil {
		return nil, err
	}

	return &manifest, nil
}

func downloadFile(serverURL, filePath string) error {
	url := strings.TrimRight(serverURL, "/") + "/" + filePath

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	dir := filepath.Dir(filePath)
	if dir != "." {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}

	tmpFile := filePath + ".tmp"
	out, err := os.Create(tmpFile)
	if err != nil {
		return err
	}

	_, err = io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		os.Remove(tmpFile)
		return err
	}

	err = os.Rename(tmpFile, filePath)
	if err != nil {
		os.Remove(tmpFile)
		return err
	}

	return nil
}

func calculateMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func launchGame(config *Config) error {
	// Get the directory where the launcher is located
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not determine launcher location: %v", err)
	}
	launcherDir := filepath.Dir(exePath)

	// Look for game exe in the same directory as the launcher
	gameExePath := filepath.Join(launcherDir, config.GameExe)

	// Check if game exe exists
	if _, err := os.Stat(gameExePath); os.IsNotExist(err) {
		return fmt.Errorf("game executable not found: %s\n\nMake sure %s is in the same folder as LaunchPad.exe", gameExePath, config.GameExe)
	}

	args := []string{}
	if config.GameArgs != "" {
		args = strings.Fields(config.GameArgs)
	}

	cmd := exec.Command(gameExePath, args...)
	// Set working directory to the game directory
	cmd.Dir = launcherDir

	if runtime.GOOS == "windows" {
		return cmd.Start()
	}

	return cmd.Run()
}

func checkLauncherUpdates(manifest *Manifest) bool {
	launcherFiles := []string{"LaunchPad.exe", "patcher.exe", "patcher-config.json"}

	for _, file := range manifest.Files {
		// Check if this is a launcher file
		isLauncherFile := false
		for _, lf := range launcherFiles {
			if file.Path == lf {
				isLauncherFile = true
				break
			}
		}

		if !isLauncherFile {
			continue
		}

		// Get launcher directory
		exePath, err := os.Executable()
		if err != nil {
			continue
		}
		launcherDir := filepath.Dir(exePath)
		localPath := filepath.Join(launcherDir, file.Path)

		// Check if file exists
		info, err := os.Stat(localPath)
		if os.IsNotExist(err) {
			return true // Launcher file missing
		}

		// Check size
		if info.Size() != file.Size {
			return true // Launcher file different size
		}

		// Check MD5
		localMD5, err := calculateMD5(localPath)
		if err != nil || localMD5 != file.MD5 {
			return true // Launcher file different hash
		}
	}

	return false
}

// findObsoleteFiles finds files that were previously installed by the patcher but are no longer in the manifest
func findObsoleteFiles(serverManifest *Manifest) []string {
	obsolete := []string{}

	// Load local manifest (tracks files we've previously downloaded)
	localManifest := loadLocalManifest()
	if localManifest == nil {
		// No local manifest = first run, nothing to delete
		return obsolete
	}

	// Launcher files that should never be deleted
	launcherFiles := map[string]bool{
		"LaunchPad.exe":        true,
		"patcher.exe":          true,
		"patcher-config.json":  true,
		".patcher-manifest.json": true,
	}

	// Create a map of all files in server manifest for quick lookup
	serverFiles := make(map[string]bool)
	for _, file := range serverManifest.Files {
		serverFiles[filepath.ToSlash(file.Path)] = true
	}

	// Check each file we previously downloaded
	for _, file := range localManifest.Files {
		normalizedPath := filepath.ToSlash(file.Path)

		// Skip launcher files
		if launcherFiles[normalizedPath] {
			continue
		}

		// If file is not in server manifest, mark for deletion
		if !serverFiles[normalizedPath] {
			obsolete = append(obsolete, file.Path)
		}
	}

	return obsolete
}

// loadLocalManifest loads the local manifest that tracks files we've downloaded
func loadLocalManifest() *Manifest {
	data, err := os.ReadFile(localManifestFile)
	if err != nil {
		return nil
	}

	var manifest Manifest
	err = json.Unmarshal(data, &manifest)
	if err != nil {
		return nil
	}

	return &manifest
}

// saveLocalManifest saves the current server manifest as our local record
func saveLocalManifest(manifest *Manifest) {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(localManifestFile, data, 0644)
}

func showError(win fyne.Window, message string) {
	dialog.ShowError(fmt.Errorf("%s", message), win)
}

// downloadNews fetches the news.json from the server
func downloadNews(serverURL string) (*NewsConfig, error) {
	url := strings.TrimRight(serverURL, "/") + "/news.json"

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var newsConfig NewsConfig
	err = json.NewDecoder(resp.Body).Decode(&newsConfig)
	if err != nil {
		return nil, err
	}

	return &newsConfig, nil
}

// createNewsFader creates a rotating news fader widget
func createNewsFader(serverURL string) *canvas.Text {
	newsLabel := canvas.NewText("", theme.ForegroundColor())
	newsLabel.TextSize = 14
	newsLabel.Alignment = fyne.TextAlignCenter
	newsLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Try to download news
	go func() {
		newsConfig, err := downloadNews(serverURL)
		if err != nil || newsConfig == nil || len(newsConfig.Items) == 0 || !newsConfig.Enabled {
			// No news or error - hide the label
			newsLabel.Text = ""
			newsLabel.Refresh()
			return
		}

		// Start rotating through news items
		currentIndex := 0
		rotationTime := newsConfig.RotationTime
		if rotationTime < 1 {
			rotationTime = 5 // Default to 5 seconds
		}

		// Set first news item
		if len(newsConfig.Items) > 0 {
			newsLabel.Text = newsConfig.Items[0].Text
			if newsConfig.Items[0].Color != "" && newsConfig.Items[0].Color != "#FFFFFF" {
				newsLabel.Color = parseHexColor(newsConfig.Items[0].Color)
			}
			newsLabel.Refresh()
		}

		// Rotate through items using a ticker
		ticker := time.NewTicker(time.Duration(rotationTime) * time.Second)
		go func() {
			for range ticker.C {
				currentIndex = (currentIndex + 1) % len(newsConfig.Items)
				item := newsConfig.Items[currentIndex]

				newsLabel.Text = item.Text
				if item.Color != "" && item.Color != "#FFFFFF" {
					newsLabel.Color = parseHexColor(item.Color)
				} else {
					newsLabel.Color = theme.ForegroundColor()
				}
				newsLabel.Refresh()
			}
		}()
	}()

	return newsLabel
}

// parseHexColor converts a hex color string to color.Color
func parseHexColor(hex string) color.Color {
	// Remove # if present
	if strings.HasPrefix(hex, "#") {
		hex = hex[1:]
	}

	// Parse hex values
	var r, g, b uint8
	if len(hex) == 6 {
		fmt.Sscanf(hex[0:2], "%x", &r)
		fmt.Sscanf(hex[2:4], "%x", &g)
		fmt.Sscanf(hex[4:6], "%x", &b)
		return color.RGBA{R: r, G: g, B: b, A: 255}
	}

	return theme.ForegroundColor()
}
