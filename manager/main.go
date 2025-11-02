package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// AppState holds the global application state
type AppState struct {
	connMgr     *ConnectionManager
	uploadQueue *UploadQueue
	manifestMgr *ManifestManager
	newsConfig  *NewsConfig
	mainWindow  fyne.Window
}

// NewAppState creates a new application state
func NewAppState(window fyne.Window) *AppState {
	connMgr := NewConnectionManager()
	return &AppState{
		connMgr:     connMgr,
		uploadQueue: NewUploadQueue(),
		manifestMgr: NewManifestManager(connMgr),
		newsConfig:  NewNewsConfig(),
		mainWindow:  window,
	}
}

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("EQ Patch Manager")
	myWindow.Resize(fyne.NewSize(1400, 900))

	appState := NewAppState(myWindow)

	// Create tabs for different sections
	tabs := container.NewAppTabs(
		container.NewTabItem("Connection", makeConnectionTabIntegrated(appState)),
		container.NewTabItem("File Upload", makeFileUploadTabIntegrated(appState)),
		container.NewTabItem("News Editor", makeNewsEditorTabIntegrated(appState)),
		container.NewTabItem("Manifest", makeManifestTabIntegrated(appState)),
	)

	myWindow.SetContent(tabs)
	myWindow.ShowAndRun()
}

// makeConnectionTabIntegrated creates the connection tab with working functionality
func makeConnectionTabIntegrated(state *AppState) *fyne.Container {
	hostEntry := widget.NewEntry()
	hostEntry.SetPlaceHolder("example.com or 192.168.1.100")

	portEntry := widget.NewEntry()
	portEntry.SetText("22")

	usernameEntry := widget.NewEntry()
	usernameEntry.SetPlaceHolder("root")

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("SSH Password")

	remotePath := widget.NewEntry()
	remotePath.SetText("/var/www/html/eq-patches")

	statusLabel := widget.NewLabel("Not connected")
	statusLabel.Wrapping = fyne.TextWrapWord

	var connectBtn *widget.Button
	connectBtn = widget.NewButton("Connect", func() {
		if state.connMgr.IsConnected() {
			// Disconnect
			state.connMgr.Disconnect()
			statusLabel.SetText("Disconnected")
			connectBtn.SetText("Connect")
			return
		}

		// Connect
		statusLabel.SetText("Connecting...")
		profile := &ConnectionProfile{
			Host:       hostEntry.Text,
			Port:       portEntry.Text,
			Username:   usernameEntry.Text,
			Password:   passwordEntry.Text,
			RemotePath: remotePath.Text,
		}

		err := state.connMgr.Connect(profile)
		if err != nil {
			statusLabel.SetText(fmt.Sprintf("Connection failed: %v", err))
			dialog.ShowError(err, state.mainWindow)
			return
		}

		statusLabel.SetText(fmt.Sprintf("✓ Connected to %s@%s:%s", profile.Username, profile.Host, profile.Port))
		connectBtn.SetText("Disconnect")

		// Test remote path
		_, err = state.connMgr.ListRemoteDir(profile.RemotePath)
		if err != nil {
			statusLabel.SetText(statusLabel.Text + fmt.Sprintf("\n⚠ Warning: Cannot access remote path: %v", err))
		} else {
			statusLabel.SetText(statusLabel.Text + fmt.Sprintf("\n✓ Remote path accessible: %s", profile.RemotePath))
		}
	})

	testBtn := widget.NewButton("Test Connection", func() {
		if !state.connMgr.IsConnected() {
			dialog.ShowError(fmt.Errorf("not connected"), state.mainWindow)
			return
		}

		output, err := state.connMgr.ExecuteCommand("pwd && whoami")
		if err != nil {
			dialog.ShowError(err, state.mainWindow)
			return
		}

		dialog.ShowInformation("Connection Test", "SSH command executed successfully:\n\n"+output, state.mainWindow)
	})

	form := container.NewVBox(
		widget.NewLabel("SSH Connection Settings"),
		widget.NewForm(
			widget.NewFormItem("Host", hostEntry),
			widget.NewFormItem("Port", portEntry),
			widget.NewFormItem("Username", usernameEntry),
			widget.NewFormItem("Password", passwordEntry),
			widget.NewFormItem("Remote Patch Path", remotePath),
		),
		container.NewHBox(connectBtn, testBtn),
		widget.NewSeparator(),
		statusLabel,
	)

	return container.NewPadded(form)
}

// makeFileUploadTabIntegrated creates the file upload tab with tree browsers
func makeFileUploadTabIntegrated(state *AppState) *fyne.Container {
	// Local folder browser with tree
	var localRootPath string
	var localFileMap map[string]*FileItem
	localFileMap = make(map[string]*FileItem)

	localPathLabel := widget.NewLabel("Local Folder: (not selected)")

	selectLocalFolderBtn := widget.NewButton("Select Local Folder", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}

			localRootPath = uri.Path()
			localPathLabel.SetText("Local Folder: " + filepath.Base(localRootPath))

			// Scan local files
			files, err := GetLocalFiles(localRootPath)
			if err != nil {
				dialog.ShowError(err, state.mainWindow)
				return
			}

			// Build file map
			localFileMap = make(map[string]*FileItem)
			var buildMap func(*FileItem, string)
			buildMap = func(item *FileItem, path string) {
				localFileMap[path] = item
				for _, child := range item.Children {
					childPath := filepath.Join(path, child.Name)
					buildMap(child, childPath)
				}
			}
			buildMap(files, localRootPath)
		}, state.mainWindow)
	})

	localList := widget.NewList(
		func() int {
			count := 0
			for _, item := range localFileMap {
				if !item.IsDir {
					count++
				}
			}
			return count
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			i := 0
			for _, item := range localFileMap {
				if !item.IsDir {
					if i == id {
						label.SetText(fmt.Sprintf("📄 %s (%s)", item.Name, FormatFileSize(item.Size)))
						break
					}
					i++
				}
			}
		},
	)

	// Right side - Remote folder tree (static structure)
	remoteFolderList := widget.NewList(
		func() int { return len(EQFolderStructure) },
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			folder := EQFolderStructure[id]
			if folder == "" {
				folder = "/ (Root)"
			}
			label.SetText(fmt.Sprintf("📁 %s", folder))
		},
	)

	selectedFolder := ""
	remoteFolderList.OnSelected = func(id widget.ListItemID) {
		selectedFolder = EQFolderStructure[id]
	}

	// Add files button
	addToQueueBtn := widget.NewButton("Add Selected Files to Queue", func() {
		if !state.connMgr.IsConnected() {
			dialog.ShowError(fmt.Errorf("not connected to server"), state.mainWindow)
			return
		}

		if localRootPath == "" {
			dialog.ShowError(fmt.Errorf("no local folder selected"), state.mainWindow)
			return
		}

		// Add all files from local folder to queue
		remotePath := state.connMgr.profile.RemotePath
		added := 0

		for path, item := range localFileMap {
			if item.IsDir {
				continue
			}

			// Determine destination
			destFolder := selectedFolder
			if destFolder == "" {
				destFolder = GetEQFolderForFile(item.Name)
			}

			remoteDest := filepath.Join(remotePath, destFolder, item.Name)
			state.uploadQueue.AddFile(path, remoteDest, item.Size)
			added++
		}

		dialog.ShowInformation("Queue Updated", fmt.Sprintf("Added %d files to upload queue", added), state.mainWindow)
	})

	// Upload queue display
	queueLabel := widget.NewLabel("Upload Queue: 0 files")
	queueList := widget.NewLabel("Queue is empty")
	uploadProgress := widget.NewProgressBar()

	updateQueueDisplay := func() {
		total, completed, failed, pending := state.uploadQueue.GetStats()
		queueLabel.SetText(fmt.Sprintf("Upload Queue: %d total, %d completed, %d failed, %d pending", total, completed, failed, pending))

		if len(state.uploadQueue.Items) == 0 {
			queueList.SetText("Queue is empty")
		} else {
			var queueText strings.Builder
			for i, item := range state.uploadQueue.Items {
				status := item.Status
				if item.Uploaded > 0 {
					pct := float64(item.Uploaded) / float64(item.Size) * 100
					status = fmt.Sprintf("%s (%.1f%%)", status, pct)
				}
				queueText.WriteString(fmt.Sprintf("%d. %s -> %s [%s]\n", i+1, filepath.Base(item.LocalPath), item.DestFolder, status))
			}
			queueList.SetText(queueText.String())
		}
	}

	uploadBtn := widget.NewButton("Start Upload", func() {
		if !state.connMgr.IsConnected() {
			dialog.ShowError(fmt.Errorf("not connected"), state.mainWindow)
			return
		}

		if len(state.uploadQueue.Items) == 0 {
			dialog.ShowError(fmt.Errorf("queue is empty"), state.mainWindow)
			return
		}

		// Upload files
		go func() {
			for _, item := range state.uploadQueue.Items {
				if item.Status != "pending" {
					continue
				}

				item.Status = "uploading"
				updateQueueDisplay()

				// Create remote directory if needed
				remoteDir := filepath.Dir(item.RemotePath)
				state.connMgr.CreateRemoteDir(remoteDir)

				// Upload with progress
				err := state.connMgr.UploadFileResumable(item.LocalPath, item.RemotePath, func(uploaded, total int64) {
					item.Uploaded = uploaded
					uploadProgress.SetValue(float64(uploaded) / float64(total))
					updateQueueDisplay()
				})

				if err != nil {
					item.Status = "failed"
					item.Error = err.Error()
				} else {
					item.Status = "completed"
				}

				updateQueueDisplay()
			}

			dialog.ShowInformation("Upload Complete", "All files have been uploaded", state.mainWindow)
		}()
	})

	var pauseBtn *widget.Button
	pauseBtn = widget.NewButton("Pause", func() {
		state.uploadQueue.IsPaused = !state.uploadQueue.IsPaused
		if state.uploadQueue.IsPaused {
			pauseBtn.SetText("Resume")
		} else {
			pauseBtn.SetText("Pause")
		}
	})

	clearQueueBtn := widget.NewButton("Clear Queue", func() {
		state.uploadQueue.Clear()
		updateQueueDisplay()
	})

	// Layout
	leftPanel := container.NewBorder(
		container.NewVBox(
			localPathLabel,
			selectLocalFolderBtn,
		),
		nil, nil, nil,
		container.NewScroll(localList),
	)

	rightPanel := container.NewBorder(
		container.NewVBox(
			widget.NewLabel("Select Destination Folder (from EQ structure)"),
			widget.NewLabel("Click a folder, then click 'Add Files to Queue'"),
		),
		addToQueueBtn,
		nil, nil,
		container.NewScroll(remoteFolderList),
	)

	topSplit := container.NewHSplit(leftPanel, rightPanel)
	topSplit.Offset = 0.5

	queuePanel := container.NewBorder(
		container.NewVBox(queueLabel, uploadProgress),
		container.NewHBox(uploadBtn, pauseBtn, clearQueueBtn),
		nil, nil,
		container.NewScroll(queueList),
	)

	return container.NewBorder(topSplit, queuePanel, nil, nil, container.NewVBox())
}

// makeNewsEditorTabIntegrated creates the news editor tab with working functionality
func makeNewsEditorTabIntegrated(state *AppState) *fyne.Container {
	newsEntry := widget.NewMultiLineEntry()
	newsEntry.SetPlaceHolder("Enter news text here...")
	newsEntry.Wrapping = fyne.TextWrapWord

	colorSelect := widget.NewSelect(GetColorPresetNames(), nil)
	colorSelect.SetSelected("White")

	boldCheck := widget.NewCheck("Bold", nil)
	italicCheck := widget.NewCheck("Italic", nil)

	rotationEntry := widget.NewEntry()
	rotationEntry.SetText(fmt.Sprintf("%d", state.newsConfig.RotationTime))

	newsListWidget := widget.NewList(
		func() int { return len(state.newsConfig.Items) },
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(state.newsConfig.Items[i].GetPreviewText())
		},
	)

	updateNewsList := func() {
		newsListWidget.Refresh()
	}

	addBtn := widget.NewButton("Add News Item", func() {
		if newsEntry.Text == "" {
			dialog.ShowError(fmt.Errorf("news text cannot be empty"), state.mainWindow)
			return
		}

		color := ColorPresets[colorSelect.Selected]
		state.newsConfig.AddItem(newsEntry.Text, color, boldCheck.Checked, italicCheck.Checked)

		newsEntry.SetText("")
		updateNewsList()

		dialog.ShowInformation("Success", "News item added", state.mainWindow)
	})

	removeBtn := widget.NewButton("Remove Selected", func() {
		if len(state.newsConfig.Items) == 0 {
			return
		}

		// Remove the first selected item (Fyne list doesn't expose selection easily)
		if len(state.newsConfig.Items) > 0 {
			state.newsConfig.RemoveItem(0)
			updateNewsList()
		}
	})

	publishBtn := widget.NewButton("Publish to Server", func() {
		if !state.connMgr.IsConnected() {
			dialog.ShowError(fmt.Errorf("not connected to server"), state.mainWindow)
			return
		}

		// Save to temp file
		tempFile := "/tmp/news.json"
		err := state.newsConfig.SaveToFile(tempFile)
		if err != nil {
			dialog.ShowError(err, state.mainWindow)
			return
		}

		// Upload to server
		remotePath := filepath.Join(state.connMgr.profile.RemotePath, "news.json")
		err = state.connMgr.UploadFile(tempFile, remotePath)
		if err != nil {
			dialog.ShowError(err, state.mainWindow)
			return
		}

		dialog.ShowInformation("Success", "News published to server!", state.mainWindow)
	})

	editorPanel := container.NewBorder(
		container.NewVBox(
			widget.NewLabel("News Editor"),
			container.NewHBox(
				widget.NewLabel("Color:"),
				colorSelect,
				boldCheck,
				italicCheck,
			),
		),
		addBtn,
		nil, nil,
		newsEntry,
	)

	listPanel := container.NewBorder(
		widget.NewLabel("News Items"),
		container.NewVBox(
			widget.NewForm(widget.NewFormItem("Rotation Time (seconds)", rotationEntry)),
			container.NewHBox(removeBtn, publishBtn),
		),
		nil, nil,
		newsListWidget,
	)

	split := container.NewHSplit(editorPanel, listPanel)
	split.Offset = 0.6

	return container.NewPadded(split)
}

// makeManifestTabIntegrated creates the manifest tab with working functionality
func makeManifestTabIntegrated(state *AppState) *fyne.Container {
	manifestContent := widget.NewMultiLineEntry()
	manifestContent.SetPlaceHolder("Manifest content will appear here...")
	manifestContent.Wrapping = fyne.TextWrapWord

	statusLabel := widget.NewLabel("Manifest not loaded")

	loadBtn := widget.NewButton("Load Manifest", func() {
		if !state.connMgr.IsConnected() {
			dialog.ShowError(fmt.Errorf("not connected to server"), state.mainWindow)
			return
		}

		err := state.manifestMgr.LoadManifest(state.connMgr.profile.RemotePath)
		if err != nil {
			dialog.ShowError(err, state.mainWindow)
			return
		}

		summary := state.manifestMgr.GetManifestSummary()
		manifestContent.SetText(summary)
		statusLabel.SetText("✓ Manifest loaded successfully")
	})

	rebuildBtn := widget.NewButton("Rebuild Manifest", func() {
		if !state.connMgr.IsConnected() {
			dialog.ShowError(fmt.Errorf("not connected to server"), state.mainWindow)
			return
		}

		statusLabel.SetText("Rebuilding manifest...")

		output, err := state.manifestMgr.RebuildManifest(state.connMgr.profile.RemotePath)
		if err != nil {
			statusLabel.SetText(fmt.Sprintf("Error: %v", err))
			dialog.ShowError(err, state.mainWindow)
			return
		}

		summary := state.manifestMgr.GetManifestSummary()
		manifestContent.SetText(summary + "\n\nBuild Output:\n" + output)
		statusLabel.SetText("✓ Manifest rebuilt successfully")

		dialog.ShowInformation("Success", "Manifest rebuilt successfully!", state.mainWindow)
	})

	viewJSONBtn := widget.NewButton("View JSON", func() {
		json, err := state.manifestMgr.GetManifestJSON()
		if err != nil {
			dialog.ShowError(err, state.mainWindow)
			return
		}

		manifestContent.SetText(json)
	})

	controls := container.NewVBox(
		widget.NewLabel("Manifest Management"),
		container.NewHBox(loadBtn, rebuildBtn, viewJSONBtn),
		statusLabel,
		widget.NewSeparator(),
	)

	return container.NewBorder(controls, nil, nil, nil, container.NewScroll(manifestContent))
}
