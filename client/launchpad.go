package main

import (
	"crypto/md5"
	"encoding/json"
	_ "embed"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

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

const (
	configFile = "patcher-config.json"
)

var (
	config      *Config
	statusLabel *widget.Label
	progressBar *widget.ProgressBar
	playButton  *widget.Button
	exitButton  *widget.Button
)

func main() {
	myApp := app.New()

	// Load configuration first to get launcher title
	var err error
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

	progressBar.Hide()

	if len(toDownload) > 0 {
		// Updates available - ask user
		statusLabel.SetText(fmt.Sprintf("📦 %d update(s) available", len(toDownload)))

		dialog.ShowConfirm(
			"Updates Available",
			fmt.Sprintf("%d file(s) need to be updated.\n\nWould you like to download updates now?\n\n(You can also play without updating)", len(toDownload)),
			func(update bool) {
				if update {
					// Download updates
					go performUpdate(win, toDownload)
				} else {
					// Skip updates
					statusLabel.SetText("✓ Ready to play (updates skipped)")
					playButton.Enable()
				}
			},
			win,
		)
	} else {
		// No updates needed
		statusLabel.SetText("✓ Up to date - Ready to play")
		playButton.Enable()
	}
}

func performUpdate(win fyne.Window, toDownload []FileEntry) {
	playButton.Disable()
	statusLabel.SetText(fmt.Sprintf("📥 Downloading %d file(s)...", len(toDownload)))
	progressBar.Show()
	progressBar.SetValue(0)

	for i, file := range toDownload {
		progress := float64(i) / float64(len(toDownload))
		progressBar.SetValue(progress)
		statusLabel.SetText(fmt.Sprintf("📥 Downloading %s (%d/%d)", filepath.Base(file.Path), i+1, len(toDownload)))

		err := downloadFile(config.ServerURL, file.Path)
		if err != nil {
			statusLabel.SetText("⚠️ Download failed")
			progressBar.Hide()
			showError(win, fmt.Sprintf("Failed to download %s: %v", file.Path, err))
			playButton.Enable()
			return
		}
	}

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

func showError(win fyne.Window, message string) {
	dialog.ShowError(fmt.Errorf("%s", message), win)
}
