package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

// NewsItem represents a single news entry
type NewsItem struct {
	Text      string            `json:"text"`
	Formatted string            `json:"formatted"` // HTML-like formatting tags
	Color     string            `json:"color"`     // Hex color code
	Style     map[string]string `json:"style"`     // Bold, Italic, etc.
}

// NewsConfig contains all news items and settings
type NewsConfig struct {
	Items          []*NewsItem `json:"items"`
	RotationTime   int         `json:"rotation_time"` // Seconds per item
	FadeTime       float64     `json:"fade_time"`     // Fade transition duration
	Enabled        bool        `json:"enabled"`
	BackgroundBlur bool        `json:"background_blur"`
}

// NewNewsConfig creates a default news configuration
func NewNewsConfig() *NewsConfig {
	return &NewsConfig{
		Items:          make([]*NewsItem, 0),
		RotationTime:   5,
		FadeTime:       0.5,
		Enabled:        true,
		BackgroundBlur: false,
	}
}

// AddItem adds a news item to the config
func (nc *NewsConfig) AddItem(text, color string, bold, italic bool) {
	item := &NewsItem{
		Text:  text,
		Color: color,
		Style: make(map[string]string),
	}

	// Build formatted text with style tags
	formatted := text
	if bold {
		formatted = fmt.Sprintf("<b>%s</b>", formatted)
		item.Style["bold"] = "true"
	}
	if italic {
		formatted = fmt.Sprintf("<i>%s</i>", formatted)
		item.Style["italic"] = "true"
	}
	if color != "" && color != "#FFFFFF" {
		formatted = fmt.Sprintf("<color=%s>%s</color>", color, formatted)
	}

	item.Formatted = formatted
	nc.Items = append(nc.Items, item)
}

// RemoveItem removes a news item by index
func (nc *NewsConfig) RemoveItem(index int) error {
	if index < 0 || index >= len(nc.Items) {
		return fmt.Errorf("invalid index")
	}

	nc.Items = append(nc.Items[:index], nc.Items[index+1:]...)
	return nil
}

// MoveItemUp moves an item up in the list
func (nc *NewsConfig) MoveItemUp(index int) error {
	if index <= 0 || index >= len(nc.Items) {
		return fmt.Errorf("cannot move item up")
	}

	nc.Items[index], nc.Items[index-1] = nc.Items[index-1], nc.Items[index]
	return nil
}

// MoveItemDown moves an item down in the list
func (nc *NewsConfig) MoveItemDown(index int) error {
	if index < 0 || index >= len(nc.Items)-1 {
		return fmt.Errorf("cannot move item down")
	}

	nc.Items[index], nc.Items[index+1] = nc.Items[index+1], nc.Items[index]
	return nil
}

// SaveToFile saves the news config to a JSON file
func (nc *NewsConfig) SaveToFile(path string) error {
	data, err := json.MarshalIndent(nc, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	err = ioutil.WriteFile(path, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	return nil
}

// LoadFromFile loads news config from a JSON file
func LoadNewsConfigFromFile(path string) (*NewsConfig, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	var config NewsConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	return &config, nil
}

// GetPreviewText returns a plain text preview of an item
func (ni *NewsItem) GetPreviewText() string {
	text := ni.Text
	if ni.Style["bold"] == "true" {
		text = "[BOLD] " + text
	}
	if ni.Style["italic"] == "true" {
		text = "[ITALIC] " + text
	}
	if ni.Color != "" && ni.Color != "#FFFFFF" {
		text = fmt.Sprintf("[%s] %s", ni.Color, text)
	}
	return text
}

// ValidateNewsConfig checks if the news config is valid
func ValidateNewsConfig(nc *NewsConfig) error {
	if len(nc.Items) == 0 {
		return fmt.Errorf("no news items")
	}

	if nc.RotationTime < 1 {
		return fmt.Errorf("rotation time must be at least 1 second")
	}

	if nc.FadeTime < 0 || nc.FadeTime > 5 {
		return fmt.Errorf("fade time must be between 0 and 5 seconds")
	}

	for i, item := range nc.Items {
		if item.Text == "" {
			return fmt.Errorf("item %d has empty text", i+1)
		}
	}

	return nil
}

// Common color presets for news items
var ColorPresets = map[string]string{
	"White":       "#FFFFFF",
	"Light Blue":  "#87CEEB",
	"Gold":        "#FFD700",
	"Light Green": "#90EE90",
	"Orange":      "#FFA500",
	"Pink":        "#FFB6C1",
	"Yellow":      "#FFFF00",
	"Red":         "#FF6B6B",
	"Purple":      "#DA70D6",
}

// GetColorPresetNames returns list of preset color names
func GetColorPresetNames() []string {
	names := []string{
		"White", "Light Blue", "Gold", "Light Green",
		"Orange", "Pink", "Yellow", "Red", "Purple",
	}
	return names
}
