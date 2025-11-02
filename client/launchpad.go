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
	ServerURL    string `json:"server_url"`
	ServerName   string `json:"server_name"`
	GameExe      string `json:"game_exe"`
	GameArgs     string `json:"game_args"`
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
	myWindow := myApp.NewWindow("EverQuest LaunchPad")

	// Load configuration
	var err error
	config, err = loadConfig()
	if err != nil {
		config = createDefaultConfig()
	}

	// Load background image
	bg := canvas.NewImageFromReader(strings.NewReader(string(backgroundImage)), "background")
	bg.FillMode = canvas.ImageFillStretch

	// Create UI elements with EQ-style colors
	titleLabel := canvas.NewText("EverQuest", theme.ForegroundColor())
	titleLabel.TextSize = 24
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	titleLabel.Alignment = fyne.TextAlignCenter

	serverLabel := canvas.NewText(config.ServerName, theme.ForegroundColor())
	serverLabel.TextSize = 14
	serverLabel.Alignment = fyne.TextAlignCenter

	statusLabel = widget.NewLabel("Ready to play")
	statusLabel.Alignment = fyne.TextAlignCenter

	progressBar = widget.NewProgressBar()
	progressBar.Hide()

	playButton = widget.NewButton("PLAY", func() {
		go performPatchAndLaunch(myWindow)
	})
	playButton.Importance = widget.HighImportance

	exitButton = widget.NewButton("Exit", func() {
		myApp.Quit()
	})

	// Create overlay container with semi-transparent background
	overlay := container.NewVBox(
		layout.NewSpacer(),
		container.NewCenter(titleLabel),
		container.NewCenter(serverLabel),
		layout.NewSpacer(),
		container.NewCenter(statusLabel),
		container.NewCenter(progressBar),
		layout.NewSpacer(),
		container.NewCenter(container.NewGridWithColumns(2, playButton, exitButton)),
		layout.NewSpacer(),
	)

	// Stack background and overlay
	content := container.NewStack(bg, overlay)

	myWindow.SetContent(content)
	myWindow.Resize(fyne.NewSize(600, 400))
	myWindow.SetFixedSize(true)
	myWindow.CenterOnScreen()
	myWindow.ShowAndRun()
}

func performPatchAndLaunch(win fyne.Window) {
	playButton.Disable()
	statusLabel.SetText("Checking for updates...")
	progressBar.Show()

	// Download manifest
	manifest, err := downloadManifest(config.ServerURL)
	if err != nil {
		showError(win, fmt.Sprintf("Failed to check for updates: %v", err))
		playButton.Enable()
		progressBar.Hide()
		statusLabel.SetText("Ready to play")
		return
	}

	statusLabel.SetText(fmt.Sprintf("Checking %d files...", len(manifest.Files)))

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
		statusLabel.SetText(fmt.Sprintf("Downloading %d file(s)...", len(toDownload)))

		for i, file := range toDownload {
			progress := float64(i) / float64(len(toDownload))
			progressBar.SetValue(progress)
			statusLabel.SetText(fmt.Sprintf("Downloading %s (%d/%d)", filepath.Base(file.Path), i+1, len(toDownload)))

			err := downloadFile(config.ServerURL, file.Path)
			if err != nil {
				showError(win, fmt.Sprintf("Failed to download %s: %v", file.Path, err))
				playButton.Enable()
				progressBar.Hide()
				statusLabel.SetText("Ready to play")
				return
			}
		}

		progressBar.SetValue(1.0)
		statusLabel.SetText("All files updated!")
	} else {
		statusLabel.SetText("All files up to date!")
	}

	// Launch game
	statusLabel.SetText("Launching EverQuest...")
	err = launchGame(config)
	if err != nil {
		showError(win, fmt.Sprintf("Failed to launch game: %v", err))
		playButton.Enable()
		progressBar.Hide()
		statusLabel.SetText("Ready to play")
		return
	}

	// Exit launcher
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
		ServerURL:  "http://example.com/patches",
		ServerName: "EverQuest Server",
		GameExe:    "eqgame.exe",
		GameArgs:   "patchme",
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
	args := []string{}
	if config.GameArgs != "" {
		args = strings.Fields(config.GameArgs)
	}

	cmd := exec.Command(config.GameExe, args...)

	if runtime.GOOS == "windows" {
		return cmd.Start()
	}

	return cmd.Run()
}

func showError(win fyne.Window, message string) {
	dialog.ShowError(fmt.Errorf("%s", message), win)
}
