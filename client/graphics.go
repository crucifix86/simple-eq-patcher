package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/sys/windows/registry"
)

func showGraphicsDialog(win fyne.Window) {
	// Create dialog content
	guidedButton := widget.NewButton("Guided Setup", func() {
		showGuidedGraphicsSetup(win)
	})
	guidedButton.Importance = widget.HighImportance

	manualButton := widget.NewButton("Manual INI Edit", func() {
		showManualINIDialog(win)
	})

	content := container.NewVBox(
		widget.NewLabel("Graphics Settings Configuration"),
		widget.NewLabel(""),
		widget.NewLabel("Choose how you want to configure graphics:"),
		widget.NewLabel(""),
		guidedButton,
		widget.NewLabel("Apply Windows compatibility settings automatically"),
		widget.NewLabel(""),
		manualButton,
		widget.NewLabel("View and edit eqclient.ini manually"),
	)

	d := dialog.NewCustom("Graphics Settings", "Close", content, win)
	d.Resize(fyne.NewSize(400, 300))
	d.Show()
}

func showGuidedGraphicsSetup(win fyne.Window) {
	// Get the path to eqgame.exe
	eqPath, err := filepath.Abs(config.GameExe)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Could not determine game path: %v", err), win)
		return
	}

	// Check if game exists
	if _, err := os.Stat(eqPath); os.IsNotExist(err) {
		dialog.ShowError(fmt.Errorf("Game executable not found: %s", eqPath), win)
		return
	}

	// Create guided setup dialog
	var selectedOption string

	options := []string{
		"Full Compatibility (Fullscreen - Disable DWM, DPI Override)",
		"DPI Awareness Only (Borderless Windowed)",
		"Remove All Compatibility Settings",
		"Show Current Settings",
	}

	option1 := widget.NewRadioGroup(options, func(value string) {
		selectedOption = value
	})

	applyButton := widget.NewButton("Apply", func() {
		if selectedOption == "" {
			dialog.ShowInformation("Select Option", "Please select an option first", win)
			return
		}

		switch {
		case strings.Contains(selectedOption, "Full Compatibility"):
			applyFullCompatibility(eqPath, win)
		case strings.Contains(selectedOption, "DPI Awareness Only"):
			applyDPIAwareness(eqPath, win)
		case strings.Contains(selectedOption, "Remove All"):
			removeCompatibility(eqPath, win)
		case strings.Contains(selectedOption, "Show Current"):
			showCurrentSettings(eqPath, win)
		}
	})
	applyButton.Importance = widget.HighImportance

	content := container.NewVBox(
		widget.NewLabel("Windows Compatibility Settings"),
		widget.NewLabel(""),
		widget.NewLabelWithStyle("Select compatibility mode:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel(""),
		option1,
		widget.NewLabel(""),
		widget.NewLabel("Note: Changes require restarting the game"),
		widget.NewLabel(""),
		applyButton,
	)

	d := dialog.NewCustom("Guided Graphics Setup", "Close", content, win)
	d.Resize(fyne.NewSize(500, 400))
	d.Show()
}

func applyFullCompatibility(eqPath string, win fyne.Window) {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows NT\CurrentVersion\AppCompatFlags\Layers`, registry.SET_VALUE)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to open registry: %v\nThis feature requires Windows", err), win)
		return
	}
	defer key.Close()

	value := "~ DISABLEDXMAXIMIZEDWINDOWEDMODE DPIUNAWARE HIGHDPIAWARE"
	err = key.SetStringValue(eqPath, value)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to set compatibility: %v", err), win)
		return
	}

	dialog.ShowInformation("Success", "Applied full compatibility settings:\n\n"+
		"- Disabled DX Maximized Windowed Mode\n"+
		"- DPI Unaware mode (prevents scaling)\n"+
		"- High DPI Aware flag\n\n"+
		"These settings work best for traditional fullscreen mode.\n"+
		"Restart EverQuest for changes to take effect.", win)
}

func applyDPIAwareness(eqPath string, win fyne.Window) {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows NT\CurrentVersion\AppCompatFlags\Layers`, registry.SET_VALUE)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to open registry: %v\nThis feature requires Windows", err), win)
		return
	}
	defer key.Close()

	value := "~ HIGHDPIAWARE"
	err = key.SetStringValue(eqPath, value)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to set compatibility: %v", err), win)
		return
	}

	dialog.ShowInformation("Success", "Applied DPI awareness setting.\n\n"+
		"This works best with borderless windowed mode.\n"+
		"Restart EverQuest for changes to take effect.", win)
}

func removeCompatibility(eqPath string, win fyne.Window) {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows NT\CurrentVersion\AppCompatFlags\Layers`, registry.SET_VALUE)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to open registry: %v\nThis feature requires Windows", err), win)
		return
	}
	defer key.Close()

	err = key.DeleteValue(eqPath)
	if err != nil && err != registry.ErrNotExist {
		dialog.ShowError(fmt.Errorf("Failed to remove settings: %v", err), win)
		return
	}

	dialog.ShowInformation("Success", "Compatibility settings removed.\n"+
		"EverQuest will use default Windows behavior.", win)
}

func showCurrentSettings(eqPath string, win fyne.Window) {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows NT\CurrentVersion\AppCompatFlags\Layers`, registry.QUERY_VALUE)
	if err != nil {
		dialog.ShowInformation("Current Settings", "No compatibility settings currently applied.", win)
		return
	}
	defer key.Close()

	value, _, err := key.GetStringValue(eqPath)
	if err == registry.ErrNotExist {
		dialog.ShowInformation("Current Settings", "No compatibility settings currently applied.", win)
		return
	}
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to read settings: %v", err), win)
		return
	}

	dialog.ShowInformation("Current Settings", fmt.Sprintf("Current compatibility flags:\n\n%s", value), win)
}

func showManualINIDialog(win fyne.Window) {
	// Find INI files
	iniFiles := []string{}

	// Common EQ INI files
	candidates := []string{"eqclient.ini", "eqgame.ini", "dbg.txt"}

	for _, name := range candidates {
		if _, err := os.Stat(name); err == nil {
			iniFiles = append(iniFiles, name)
		}
	}

	// Also check for character-specific INIs
	pattern := "*_*.ini"
	matches, _ := filepath.Glob(pattern)
	iniFiles = append(iniFiles, matches...)

	if len(iniFiles) == 0 {
		dialog.ShowInformation("No INI Files", "No EverQuest configuration files found in current directory.\n\n"+
			"Common files:\n"+
			"- eqclient.ini (main graphics settings)\n"+
			"- eqgame.ini (game settings)\n"+
			"- dbg.txt (debug/performance settings)", win)
		return
	}

	// Create file list
	var selectedFile string
	fileList := widget.NewList(
		func() int { return len(iniFiles) },
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(iniFiles[i])
		},
	)

	fileList.OnSelected = func(id widget.ListItemID) {
		selectedFile = iniFiles[id]
	}

	openButton := widget.NewButton("Open Selected File", func() {
		if selectedFile == "" {
			dialog.ShowInformation("No Selection", "Please select a file to open", win)
			return
		}

		openINIFile(selectedFile, win)
	})
	openButton.Importance = widget.HighImportance

	helpText := widget.NewLabel(
		"Select an INI file to view and edit.\n\n" +
		"Key files:\n" +
		"• eqclient.ini - Graphics and display settings\n" +
		"• eqgame.ini - Game behavior settings\n" +
		"• Character INIs - Per-character settings",
	)

	content := container.NewBorder(
		container.NewVBox(
			widget.NewLabel("EverQuest Configuration Files"),
			widget.NewSeparator(),
			helpText,
			widget.NewSeparator(),
		),
		container.NewVBox(
			widget.NewSeparator(),
			openButton,
		),
		nil,
		nil,
		fileList,
	)

	d := dialog.NewCustom("Manual INI Configuration", "Close", content, win)
	d.Resize(fyne.NewSize(500, 400))
	d.Show()
}

func openINIFile(filename string, win fyne.Window) {
	// Read file
	content, err := os.ReadFile(filename)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to read file: %v", err), win)
		return
	}

	// Create editor
	editor := widget.NewMultiLineEntry()
	editor.SetText(string(content))
	editor.Wrapping = fyne.TextWrapOff

	saveButton := widget.NewButton("Save Changes", func() {
		err := os.WriteFile(filename, []byte(editor.Text), 0644)
		if err != nil {
			dialog.ShowError(fmt.Errorf("Failed to save file: %v", err), win)
			return
		}
		dialog.ShowInformation("Success", fmt.Sprintf("Saved changes to %s", filename), win)
	})
	saveButton.Importance = widget.HighImportance

	editorContent := container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle(filename, fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewLabel("Edit the file below and click Save Changes"),
			widget.NewSeparator(),
		),
		container.NewVBox(
			widget.NewSeparator(),
			saveButton,
		),
		nil,
		nil,
		container.NewScroll(editor),
	)

	d := dialog.NewCustom("Edit INI File", "Close", editorContent, win)
	d.Resize(fyne.NewSize(700, 500))
	d.Show()
}
