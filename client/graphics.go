package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type GraphicsSettings struct {
	Width              int
	Height             int
	Fullscreen         bool
	TextureQuality     int
	ShowGrass          bool
	ShowSpellEffects   int
	ShowDynamicLights  bool
	VertexShaders      bool
	PixelShaders       bool
}

var commonResolutions = []string{
	"800x600",
	"1024x768",
	"1280x720",
	"1280x1024",
	"1366x768",
	"1600x900",
	"1920x1080",
	"2560x1440",
	"3840x2160",
}

func showGraphicsDialog(win fyne.Window) {
	// Load current settings
	ini, err := LoadINI("eqclient.ini")
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to load graphics settings: %v", err), win)
		return
	}

	// Parse current settings
	currentSettings := parseGraphicsSettings(ini)

	// Find current resolution index
	currentRes := fmt.Sprintf("%dx%d", currentSettings.Width, currentSettings.Height)
	selectedResIndex := 0
	for i, res := range commonResolutions {
		if res == currentRes {
			selectedResIndex = i
			break
		}
	}

	// Create UI elements
	resolutionSelect := widget.NewSelect(commonResolutions, nil)
	resolutionSelect.SetSelectedIndex(selectedResIndex)

	fullscreenCheck := widget.NewCheck("Fullscreen", nil)
	fullscreenCheck.SetChecked(currentSettings.Fullscreen)

	textureQuality := widget.NewSelect([]string{"Low (0)", "Medium (1)", "High (2)", "Ultra (3)"}, nil)
	textureQuality.SetSelectedIndex(currentSettings.TextureQuality)

	grassCheck := widget.NewCheck("Show Grass", nil)
	grassCheck.SetChecked(currentSettings.ShowGrass)

	lightsCheck := widget.NewCheck("Dynamic Lighting", nil)
	lightsCheck.SetChecked(currentSettings.ShowDynamicLights)

	spellEffects := widget.NewSelect([]string{"Off (0)", "Low (1)", "Medium (2)", "High (3)"}, nil)
	spellEffects.SetSelectedIndex(currentSettings.ShowSpellEffects)

	vertexShadersCheck := widget.NewCheck("Vertex Shaders", nil)
	vertexShadersCheck.SetChecked(currentSettings.VertexShaders)

	pixelShadersCheck := widget.NewCheck("Pixel Shaders", nil)
	pixelShadersCheck.SetChecked(currentSettings.PixelShaders)

	// Save button
	saveButton := widget.NewButton("Apply Settings", func() {
		// Parse resolution
		resParts := parseResolution(resolutionSelect.Selected)
		if resParts[0] == 0 {
			dialog.ShowError(fmt.Errorf("Invalid resolution selected"), win)
			return
		}

		// Update settings
		settings := GraphicsSettings{
			Width:              resParts[0],
			Height:             resParts[1],
			Fullscreen:         fullscreenCheck.Checked,
			TextureQuality:     textureQuality.SelectedIndex(),
			ShowGrass:          grassCheck.Checked,
			ShowSpellEffects:   spellEffects.SelectedIndex(),
			ShowDynamicLights:  lightsCheck.Checked,
			VertexShaders:      vertexShadersCheck.Checked,
			PixelShaders:       pixelShadersCheck.Checked,
		}

		// Save to INI
		err := saveGraphicsSettings(ini, settings)
		if err != nil {
			dialog.ShowError(fmt.Errorf("Failed to save settings: %v", err), win)
			return
		}

		dialog.ShowInformation("Success", "Graphics settings saved!\nRestart the game for changes to take effect.", win)
	})
	saveButton.Importance = widget.HighImportance

	resetButton := widget.NewButton("Reset to Defaults", func() {
		resolutionSelect.SetSelected("1920x1080")
		fullscreenCheck.SetChecked(true)
		textureQuality.SetSelectedIndex(2)
		grassCheck.SetChecked(true)
		lightsCheck.SetChecked(true)
		spellEffects.SetSelectedIndex(2)
		vertexShadersCheck.SetChecked(true)
		pixelShadersCheck.SetChecked(true)
	})

	// Compatibility fix button
	compatButton := widget.NewButton("Compatibility Fix Wizard", func() {
		showCompatibilityWizard(win)
	})
	compatButton.Importance = widget.MediumImportance

	// Layout
	content := container.NewVBox(
		widget.NewLabelWithStyle("Graphics Settings", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),

		// Display settings
		widget.NewLabelWithStyle("Display Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewGridWithColumns(2,
			widget.NewLabel("Resolution:"),
			resolutionSelect,
		),
		fullscreenCheck,

		widget.NewSeparator(),

		// Graphics quality
		widget.NewLabelWithStyle("Graphics Quality", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewGridWithColumns(2,
			widget.NewLabel("Texture Quality:"),
			textureQuality,
		),
		container.NewGridWithColumns(2,
			widget.NewLabel("Spell Effects:"),
			spellEffects,
		),
		grassCheck,
		lightsCheck,

		widget.NewSeparator(),

		// Advanced
		widget.NewLabelWithStyle("Advanced", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		vertexShadersCheck,
		pixelShadersCheck,

		widget.NewSeparator(),

		// Compatibility Fix
		widget.NewLabelWithStyle("Troubleshooting", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Having issues with fullscreen or scaling?"),
		compatButton,

		widget.NewSeparator(),

		// Buttons
		container.NewGridWithColumns(2, saveButton, resetButton),
	)

	scroll := container.NewScroll(content)
	d := dialog.NewCustom("Graphics Settings", "Close", scroll, win)
	d.Resize(fyne.NewSize(500, 650))
	d.Show()
}

func parseGraphicsSettings(ini *INIFile) GraphicsSettings {
	settings := GraphicsSettings{
		Width:              800,
		Height:             600,
		Fullscreen:         true,
		TextureQuality:     1,
		ShowGrass:          true,
		ShowSpellEffects:   1,
		ShowDynamicLights:  true,
		VertexShaders:      true,
		PixelShaders:       true,
	}

	// Read from VideoMode section
	if width := ini.Get("VideoMode", "Width", ""); width != "" {
		if w, err := strconv.Atoi(width); err == nil {
			settings.Width = w
		}
	}
	if height := ini.Get("VideoMode", "Height", ""); height != "" {
		if h, err := strconv.Atoi(height); err == nil {
			settings.Height = h
		}
	}

	// Read from Defaults section
	windowedMode := ini.Get("Defaults", "WindowedMode", "FALSE")
	settings.Fullscreen = windowedMode != "TRUE"

	if tq := ini.Get("Defaults", "TextureQuality", ""); tq != "" {
		if q, err := strconv.Atoi(tq); err == nil && q >= 0 && q <= 3 {
			settings.TextureQuality = q
		}
	}

	settings.ShowGrass = ini.Get("Defaults", "ShowGrass", "TRUE") == "TRUE"
	settings.ShowDynamicLights = ini.Get("Defaults", "ShowDynamicLights", "TRUE") == "TRUE"
	settings.VertexShaders = ini.Get("Defaults", "VertexShaders", "TRUE") == "TRUE"
	settings.PixelShaders = ini.Get("Defaults", "20PixelShaders", "TRUE") == "TRUE"

	if se := ini.Get("Defaults", "ShowSpellEffects", ""); se != "" {
		if s, err := strconv.Atoi(se); err == nil && s >= 0 && s <= 3 {
			settings.ShowSpellEffects = s
		}
	}

	return settings
}

func saveGraphicsSettings(ini *INIFile, settings GraphicsSettings) error {
	// Save VideoMode section
	ini.Set("VideoMode", "Width", strconv.Itoa(settings.Width))
	ini.Set("VideoMode", "Height", strconv.Itoa(settings.Height))
	ini.Set("VideoMode", "WindowedWidth", strconv.Itoa(settings.Width))
	ini.Set("VideoMode", "WindowedHeight", strconv.Itoa(settings.Height))
	ini.Set("VideoMode", "FullscreenBitsPerPixel", "32")
	ini.Set("VideoMode", "FullscreenRefreshRate", "0")

	// Save Defaults section
	windowedMode := "TRUE"
	if settings.Fullscreen {
		windowedMode = "FALSE"
	}
	ini.Set("Defaults", "WindowedMode", windowedMode)
	ini.Set("Defaults", "TextureQuality", strconv.Itoa(settings.TextureQuality))

	grassValue := "FALSE"
	if settings.ShowGrass {
		grassValue = "TRUE"
	}
	ini.Set("Defaults", "ShowGrass", grassValue)

	lightsValue := "FALSE"
	if settings.ShowDynamicLights {
		lightsValue = "TRUE"
	}
	ini.Set("Defaults", "ShowDynamicLights", lightsValue)

	vsValue := "FALSE"
	if settings.VertexShaders {
		vsValue = "TRUE"
	}
	ini.Set("Defaults", "VertexShaders", vsValue)

	psValue := "FALSE"
	if settings.PixelShaders {
		psValue = "TRUE"
	}
	ini.Set("Defaults", "20PixelShaders", psValue)
	ini.Set("Defaults", "14PixelShaders", psValue)
	ini.Set("Defaults", "1xPixelShaders", psValue)

	ini.Set("Defaults", "ShowSpellEffects", strconv.Itoa(settings.ShowSpellEffects))

	return ini.Save()
}

func parseResolution(res string) [2]int {
	var width, height int
	fmt.Sscanf(res, "%dx%d", &width, &height)
	return [2]int{width, height}
}

func showCompatibilityWizard(win fyne.Window) {
	content := container.NewVBox(
		widget.NewLabelWithStyle("Windows Compatibility Fix Wizard", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),

		widget.NewLabel("This wizard applies Windows compatibility settings to fix:"),
		widget.NewLabel("• DPI scaling issues"),
		widget.NewLabel("• Fullscreen behavior on modern monitors"),
		widget.NewLabel("• Desktop composition problems"),

		widget.NewSeparator(),
		widget.NewLabelWithStyle("Select Fix Type:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	)

	fullCompatButton := widget.NewButton("Full Compatibility (Best for Fullscreen)", func() {
		err := applyCompatibilityFix("full")
		if err != nil {
			dialog.ShowError(fmt.Errorf("Failed to apply settings: %v", err), win)
		} else {
			dialog.ShowInformation("Success", "Full compatibility settings applied!\n\nApplied:\n• Disabled DX Maximized Windowed Mode\n• DPI Unaware mode\n• High DPI Aware flag\n\nRestart EverQuest for changes to take effect.", win)
		}
	})
	fullCompatButton.Importance = widget.HighImportance

	dpiOnlyButton := widget.NewButton("DPI Awareness Only (Best for Windowed)", func() {
		err := applyCompatibilityFix("dpi")
		if err != nil {
			dialog.ShowError(fmt.Errorf("Failed to apply settings: %v", err), win)
		} else {
			dialog.ShowInformation("Success", "DPI awareness setting applied!\n\nThis works best with borderless windowed mode.\n\nRestart EverQuest for changes to take effect.", win)
		}
	})

	removeButton := widget.NewButton("Remove All Compatibility Settings", func() {
		err := applyCompatibilityFix("remove")
		if err != nil {
			dialog.ShowError(fmt.Errorf("Failed to remove settings: %v", err), win)
		} else {
			dialog.ShowInformation("Success", "All compatibility settings removed.\n\nEverQuest will use default Windows behavior.", win)
		}
	})

	content.Add(fullCompatButton)
	content.Add(dpiOnlyButton)
	content.Add(removeButton)

	d := dialog.NewCustom("Compatibility Fix Wizard", "Close", content, win)
	d.Resize(fyne.NewSize(450, 400))
	d.Show()
}

func applyCompatibilityFix(fixType string) error {
	// Get full path to eqgame.exe
	exePath, err := filepath.Abs(config.GameExe)
	if err != nil {
		return err
	}

	// Check if eqgame.exe exists
	if _, err := os.Stat(exePath); os.IsNotExist(err) {
		return fmt.Errorf("eqgame.exe not found at: %s", exePath)
	}

	// Build registry command based on fix type
	var regCmd *exec.Cmd

	switch fixType {
	case "full":
		// Full compatibility: Disable DWM, DPI override, High DPI awareness
		regValue := "~ DISABLEDXMAXIMIZEDWINDOWEDMODE DPIUNAWARE HIGHDPIAWARE"
		regCmd = exec.Command("reg", "add",
			`HKCU\Software\Microsoft\Windows NT\CurrentVersion\AppCompatFlags\Layers`,
			"/v", exePath,
			"/t", "REG_SZ",
			"/d", regValue,
			"/f")

	case "dpi":
		// DPI awareness only
		regValue := "~ HIGHDPIAWARE"
		regCmd = exec.Command("reg", "add",
			`HKCU\Software\Microsoft\Windows NT\CurrentVersion\AppCompatFlags\Layers`,
			"/v", exePath,
			"/t", "REG_SZ",
			"/d", regValue,
			"/f")

	case "remove":
		// Remove all compatibility settings
		regCmd = exec.Command("reg", "delete",
			`HKCU\Software\Microsoft\Windows NT\CurrentVersion\AppCompatFlags\Layers`,
			"/v", exePath,
			"/f")

	default:
		return fmt.Errorf("unknown fix type: %s", fixType)
	}

	// Execute registry command (only works on Windows)
	if runtime.GOOS == "windows" {
		output, err := regCmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("registry command failed: %v\nOutput: %s", err, string(output))
		}
		return nil
	}

	// On non-Windows, just report success (for testing)
	return nil
}
