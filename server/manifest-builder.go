package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: manifest-builder <directory-to-scan>")
		fmt.Println("Example: manifest-builder /var/www/eq-patches")
		os.Exit(1)
	}

	rootDir := os.Args[1]

	// Check if directory exists
	if _, err := os.Stat(rootDir); os.IsNotExist(err) {
		fmt.Printf("Error: Directory does not exist: %s\n", rootDir)
		os.Exit(1)
	}

	fmt.Printf("Scanning directory: %s\n", rootDir)

	manifest := Manifest{
		Version: "1.0",
		Files:   []FileEntry{},
	}

	// Walk directory tree
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories, manifest, update script, README, and launcher files
		// Launcher files should NOT be in manifest (can't update themselves while running)
		baseName := filepath.Base(path)
		if info.IsDir() ||
		   baseName == "manifest.json" ||
		   baseName == "update-patches.sh" ||
		   baseName == "manifest-builder" ||
		   baseName == "README.txt" ||
		   baseName == "LaunchPad.exe" ||
		   baseName == "patcher.exe" ||
		   baseName == "patcher-config.json" ||
		   baseName == "manager.exe" ||
		   baseName == "news.json" ||
		   baseName == "eq-patcher-client.zip" {
			return nil
		}

		// Calculate relative path
		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}

		// Convert to forward slashes for cross-platform compatibility
		relPath = filepath.ToSlash(relPath)

		// Calculate MD5
		hash, err := calculateMD5(path)
		if err != nil {
			fmt.Printf("Warning: Could not hash %s: %v\n", relPath, err)
			return nil
		}

		entry := FileEntry{
			Path: relPath,
			MD5:  hash,
			Size: info.Size(),
		}

		manifest.Files = append(manifest.Files, entry)
		fmt.Printf("  Added: %s (%d bytes, md5: %s)\n", relPath, info.Size(), hash[:8])

		return nil
	})

	if err != nil {
		fmt.Printf("Error walking directory: %v\n", err)
		os.Exit(1)
	}

	// Write manifest to file
	manifestPath := filepath.Join(rootDir, "manifest.json")
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		fmt.Printf("Error creating JSON: %v\n", err)
		os.Exit(1)
	}

	err = os.WriteFile(manifestPath, data, 0644)
	if err != nil {
		fmt.Printf("Error writing manifest: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✓ Manifest created: %s\n", manifestPath)
	fmt.Printf("✓ Total files: %d\n", len(manifest.Files))
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
