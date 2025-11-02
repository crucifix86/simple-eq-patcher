package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// FileItem represents a file or directory
type FileItem struct {
	Name     string
	Path     string
	IsDir    bool
	Size     int64
	Children []*FileItem
}

// UploadQueueItem represents a file queued for upload
type UploadQueueItem struct {
	LocalPath     string
	RemotePath    string
	Size          int64
	Uploaded      int64
	Status        string // "pending", "uploading", "completed", "failed", "paused"
	Error         string
	DestFolder    string // Which EQ folder this goes to
}

// UploadQueue manages the upload queue
type UploadQueue struct {
	Items    []*UploadQueueItem
	Current  int
	IsPaused bool
}

// NewUploadQueue creates a new upload queue
func NewUploadQueue() *UploadQueue {
	return &UploadQueue{
		Items:    make([]*UploadQueueItem, 0),
		Current:  0,
		IsPaused: false,
	}
}

// AddFile adds a file to the upload queue
func (uq *UploadQueue) AddFile(localPath, remotePath string, size int64) {
	item := &UploadQueueItem{
		LocalPath:  localPath,
		RemotePath: remotePath,
		Size:       size,
		Uploaded:   0,
		Status:     "pending",
	}
	uq.Items = append(uq.Items, item)
}

// GetNextPending returns the next pending item
func (uq *UploadQueue) GetNextPending() *UploadQueueItem {
	for _, item := range uq.Items {
		if item.Status == "pending" {
			return item
		}
	}
	return nil
}

// GetStats returns queue statistics
func (uq *UploadQueue) GetStats() (total, completed, failed, pending int) {
	total = len(uq.Items)
	for _, item := range uq.Items {
		switch item.Status {
		case "completed":
			completed++
		case "failed":
			failed++
		case "pending":
			pending++
		}
	}
	return
}

// Clear removes all items from the queue
func (uq *UploadQueue) Clear() {
	uq.Items = make([]*UploadQueueItem, 0)
	uq.Current = 0
}

// GetLocalFiles recursively scans a local directory
func GetLocalFiles(path string) (*FileItem, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	item := &FileItem{
		Name:  filepath.Base(path),
		Path:  path,
		IsDir: info.IsDir(),
		Size:  info.Size(),
	}

	if info.IsDir() {
		files, err := ioutil.ReadDir(path)
		if err != nil {
			return nil, err
		}

		for _, f := range files {
			childPath := filepath.Join(path, f.Name())
			child, err := GetLocalFiles(childPath)
			if err != nil {
				continue // Skip files we can't read
			}
			item.Children = append(item.Children, child)
		}
	}

	return item, nil
}

// EQFolderStructure defines the standard EverQuest client folder structure
var EQFolderStructure = []string{
	"",                      // Root level (spells_us.txt, dbg.txt, etc.)
	"ActorEffects",          // Actor effect files
	"Atlas/Default",         // Atlas files
	"AudioTriggers/default", // Audio trigger files
	"AudioTriggers/shared",
	"EnvEmitterEffects",       // Environment emitter effects
	"Resources",               // Zone files (.s3d, .eqg)
	"Resources/Precipitation", // Weather effects
	"Resources/Sky",           // Sky files
	"Resources/SlideShow",     // Loading screens
	"Resources/WaterSwap",     // Water textures
	"RenderEffects/MPL",       // Multi-pass lighting
	"RenderEffects/SPL",       // Single-pass lighting
	"SpellEffects",            // Spell effect files
	"help",                    // Help files
	"help/tips",
	"maps",      // Map files (.txt)
	"sounds",    // Sound files
	"storyline", // Storyline files
	"uifiles",   // UI files - NOTE: This is the actual folder name
	"uifiles/default",
	"userdata", // User data
	"voice/default",
}

// GetEQFolderForFile determines which EQ folder a file should go to based on extension
func GetEQFolderForFile(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	base := strings.ToLower(filepath.Base(filename))

	// Root level files
	if base == "spells_us.txt" || base == "dbg.txt" || base == "spells_us_str.txt" {
		return ""
	}

	// UI files - check for EQUI_ prefix or UI-related extensions
	if strings.HasPrefix(base, "equi_") || ext == ".xml" || ext == ".tga" {
		return "uifiles/default"
	}

	// Zone files
	if ext == ".s3d" || ext == ".eqg" || ext == ".zon" || ext == ".txt" && strings.Contains(base, "_chr") {
		return "Resources"
	}

	// Map files
	if ext == ".txt" && !strings.Contains(base, "_chr") {
		// Could be map or help file, default to maps
		return "maps"
	}

	// Spell effects
	if ext == ".eff" || ext == ".pts" {
		return "SpellEffects"
	}

	// Sound files
	if ext == ".wav" || ext == ".mp3" || ext == ".ogg" {
		return "sounds"
	}

	// Actor effects
	if ext == ".acf" {
		return "ActorEffects"
	}

	// Default to root if unknown
	return ""
}

// ValidateRemotePath ensures the remote path is within the allowed EQ patch directory
func ValidateRemotePath(basePath, targetPath string) bool {
	// Normalize paths
	absBase, _ := filepath.Abs(basePath)
	absTarget, _ := filepath.Abs(targetPath)

	// Check if target is within base
	rel, err := filepath.Rel(absBase, absTarget)
	if err != nil {
		return false
	}

	// Reject paths that escape the base directory
	if strings.HasPrefix(rel, "..") {
		return false
	}

	return true
}

// FormatFileSize formats bytes to human-readable size
func FormatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// GetFileExtensionHelp returns help text for file types
func GetFileExtensionHelp(ext string) string {
	help := map[string]string{
		".txt": "Text files (spells, maps, zone chr files)",
		".xml": "UI definition files (EQUI_*.xml)",
		".tga": "UI texture files",
		".s3d": "Zone archive files",
		".eqg": "Zone geometry files",
		".zon": "Zone configuration files",
		".eff": "Spell effect files",
		".pts": "Particle system files",
		".wav": "Audio files",
		".mp3": "Audio files",
	}

	if h, ok := help[strings.ToLower(ext)]; ok {
		return h
	}
	return "Unknown file type"
}
