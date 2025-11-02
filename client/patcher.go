package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

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
	GameExe      string `json:"game_exe"`
	GameArgs     string `json:"game_args"`
}

const (
	configFile = "patcher-config.json"
)

func main() {
	fmt.Println("═══════════════════════════════════════")
	fmt.Println("  Simple EverQuest Patcher v1.0")
	fmt.Println("═══════════════════════════════════════\n")

	// Load configuration
	config, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		fmt.Println("\nCreating default configuration file...")
		createDefaultConfig()
		fmt.Println("Please edit patcher-config.json with your server URL and game settings.")
		pause()
		os.Exit(1)
	}

	fmt.Printf("Server: %s\n", config.ServerURL)
	fmt.Printf("Game: %s %s\n\n", config.GameExe, config.GameArgs)

	// Download manifest
	fmt.Println("Downloading manifest...")
	manifest, err := downloadManifest(config.ServerURL)
	if err != nil {
		fmt.Printf("✗ Error downloading manifest: %v\n", err)
		pause()
		os.Exit(1)
	}
	fmt.Printf("✓ Manifest loaded (%d files)\n\n", len(manifest.Files))

	// Check files and build download list
	fmt.Println("Checking files...")
	toDownload := []FileEntry{}
	for _, file := range manifest.Files {
		localPath := file.Path

		// Check if file exists
		info, err := os.Stat(localPath)
		if os.IsNotExist(err) {
			fmt.Printf("  [MISSING] %s\n", file.Path)
			toDownload = append(toDownload, file)
			continue
		}

		// Check if size matches
		if info.Size() != file.Size {
			fmt.Printf("  [SIZE MISMATCH] %s\n", file.Path)
			toDownload = append(toDownload, file)
			continue
		}

		// Check MD5 hash
		localMD5, err := calculateMD5(localPath)
		if err != nil || localMD5 != file.MD5 {
			fmt.Printf("  [HASH MISMATCH] %s\n", file.Path)
			toDownload = append(toDownload, file)
			continue
		}

		// File is OK
		fmt.Printf("  [OK] %s\n", file.Path)
	}

	// Download files if needed
	if len(toDownload) > 0 {
		fmt.Printf("\n%d file(s) need updating\n", len(toDownload))
		fmt.Println("\nDownloading files...")

		for i, file := range toDownload {
			fmt.Printf("[%d/%d] %s...", i+1, len(toDownload), file.Path)

			err := downloadFile(config.ServerURL, file.Path)
			if err != nil {
				fmt.Printf(" ✗ FAILED: %v\n", err)
				pause()
				os.Exit(1)
			}

			fmt.Println(" ✓")
		}

		fmt.Println("\n✓ All files updated!")
	} else {
		fmt.Println("\n✓ All files are up to date!")
	}

	// Launch game
	fmt.Println("\nLaunching game...")
	err = launchGame(config)
	if err != nil {
		fmt.Printf("✗ Error launching game: %v\n", err)
		pause()
		os.Exit(1)
	}

	fmt.Println("✓ Game launched successfully!")
	time.Sleep(2 * time.Second)
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

func createDefaultConfig() {
	config := Config{
		ServerURL: "http://example.com/patches",
		GameExe:   "eqgame.exe",
		GameArgs:  "patchme",
	}

	data, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile(configFile, data, 0644)
	fmt.Printf("Created %s\n", configFile)
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
	// Construct URL
	url := strings.TrimRight(serverURL, "/") + "/" + filePath

	// Download file
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	// Create directory if needed
	dir := filepath.Dir(filePath)
	if dir != "." {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}

	// Create temporary file
	tmpFile := filePath + ".tmp"
	out, err := os.Create(tmpFile)
	if err != nil {
		return err
	}

	// Copy data
	_, err = io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		os.Remove(tmpFile)
		return err
	}

	// Rename to final name
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

	// On Windows, don't wait for the game to exit
	if runtime.GOOS == "windows" {
		return cmd.Start()
	}

	return cmd.Run()
}

func pause() {
	if runtime.GOOS == "windows" {
		fmt.Println("\nPress Enter to exit...")
		fmt.Scanln()
	}
}
