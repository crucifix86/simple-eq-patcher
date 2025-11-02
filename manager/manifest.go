package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"
)

// ManifestFile represents a file in the manifest
type ManifestFile struct {
	Path string `json:"path"`
	MD5  string `json:"md5"`
	Size int64  `json:"size"`
}

// Manifest represents the patch manifest structure
type Manifest struct {
	Version   string          `json:"version"`
	Generated string          `json:"generated,omitempty"`
	Files     []*ManifestFile `json:"files"`
}

// ManifestManager handles manifest operations
type ManifestManager struct {
	conn     *ConnectionManager
	manifest *Manifest
}

// NewManifestManager creates a new manifest manager
func NewManifestManager(conn *ConnectionManager) *ManifestManager {
	return &ManifestManager{
		conn: conn,
	}
}

// LoadManifest downloads and parses the manifest from the server
func (mm *ManifestManager) LoadManifest(remotePath string) error {
	if !mm.conn.IsConnected() {
		return fmt.Errorf("not connected to server")
	}

	// Download manifest.json to temp file
	tempFile := "/tmp/manifest.json"
	err := mm.conn.DownloadFile(remotePath+"/manifest.json", tempFile)
	if err != nil {
		return fmt.Errorf("failed to download manifest: %v", err)
	}

	// Parse manifest
	manifest, err := LoadManifestFromFile(tempFile)
	if err != nil {
		return fmt.Errorf("failed to parse manifest: %v", err)
	}

	mm.manifest = manifest
	return nil
}

// RebuildManifest executes the manifest-builder on the remote server
func (mm *ManifestManager) RebuildManifest(remotePath string) (string, error) {
	if !mm.conn.IsConnected() {
		return "", fmt.Errorf("not connected to server")
	}

	// Execute manifest-builder command
	command := fmt.Sprintf("cd %s && ./manifest-builder .", remotePath)
	output, err := mm.conn.ExecuteCommand(command)
	if err != nil {
		return output, fmt.Errorf("failed to rebuild manifest: %v", err)
	}

	// Reload the manifest
	err = mm.LoadManifest(remotePath)
	if err != nil {
		return output, fmt.Errorf("manifest rebuilt but failed to reload: %v", err)
	}

	return output, nil
}

// GetManifestSummary returns a human-readable summary of the manifest
func (mm *ManifestManager) GetManifestSummary() string {
	if mm.manifest == nil {
		return "No manifest loaded"
	}

	var totalSize int64
	filesByFolder := make(map[string]int)

	for _, file := range mm.manifest.Files {
		totalSize += file.Size

		// Determine folder
		folder := "Root"
		if len(file.Path) > 0 {
			// Get first part of path
			for i, c := range file.Path {
				if c == '/' || c == '\\' {
					folder = file.Path[:i]
					break
				}
			}
		}

		filesByFolder[folder]++
	}

	summary := fmt.Sprintf("Manifest Version: %s\n", mm.manifest.Version)
	if mm.manifest.Generated != "" {
		summary += fmt.Sprintf("Generated: %s\n", mm.manifest.Generated)
	}
	summary += fmt.Sprintf("Total Files: %d\n", len(mm.manifest.Files))
	summary += fmt.Sprintf("Total Size: %s\n\n", FormatFileSize(totalSize))
	summary += "Files by Folder:\n"

	for folder, count := range filesByFolder {
		summary += fmt.Sprintf("  %s: %d files\n", folder, count)
	}

	return summary
}

// GetManifestJSON returns the manifest as formatted JSON
func (mm *ManifestManager) GetManifestJSON() (string, error) {
	if mm.manifest == nil {
		return "", fmt.Errorf("no manifest loaded")
	}

	data, err := json.MarshalIndent(mm.manifest, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// VerifyFile checks if a file exists in the manifest
func (mm *ManifestManager) VerifyFile(path string) (*ManifestFile, bool) {
	if mm.manifest == nil {
		return nil, false
	}

	for _, file := range mm.manifest.Files {
		if file.Path == path {
			return file, true
		}
	}

	return nil, false
}

// GetFilesInFolder returns all files in a specific folder
func (mm *ManifestManager) GetFilesInFolder(folder string) []*ManifestFile {
	if mm.manifest == nil {
		return nil
	}

	var files []*ManifestFile
	for _, file := range mm.manifest.Files {
		// Check if file is in the specified folder
		if folder == "" {
			// Root folder - files with no path separator
			hasSlash := false
			for _, c := range file.Path {
				if c == '/' || c == '\\' {
					hasSlash = true
					break
				}
			}
			if !hasSlash {
				files = append(files, file)
			}
		} else {
			// Check if path starts with folder
			if len(file.Path) > len(folder) && file.Path[:len(folder)] == folder {
				files = append(files, file)
			}
		}
	}

	return files
}

// LoadManifestFromFile loads a manifest from a local JSON file
func LoadManifestFromFile(path string) (*Manifest, error) {
	data, err := readFileBytes(path)
	if err != nil {
		return nil, err
	}

	var manifest Manifest
	err = json.Unmarshal(data, &manifest)
	if err != nil {
		return nil, err
	}

	return &manifest, nil
}

// CreateEmptyManifest creates a new empty manifest
func CreateEmptyManifest() *Manifest {
	return &Manifest{
		Version:   "1.0",
		Generated: time.Now().Format(time.RFC3339),
		Files:     make([]*ManifestFile, 0),
	}
}

// readFileBytes is a helper to read file bytes (wrapper for testing)
func readFileBytes(path string) ([]byte, error) {
	// Use ioutil for consistency
	data, err := ioutil.ReadFile(path)
	return data, err
}
