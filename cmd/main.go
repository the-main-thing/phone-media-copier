package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"phone-copier/internal/copy"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type Config struct {
	TargetDir string   `json:"target_dir"`
	Subdirs   []string `json:"subdirs"`
}

type AppState struct {
	config     Config
	sourceDir  string
	mainWindow fyne.Window
	status     *widget.Label
}

func getConfigPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	return filepath.Join(configDir, "dircopier", "config.json")
}

func loadConfig() Config {
	defaultConfig := Config{
		TargetDir: "",
		Subdirs:   []string{"docs", "images", "data"},
	}

	configPath := getConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		return defaultConfig
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return defaultConfig
	}
	return config
}

func saveConfig(config Config) error {
	configPath := getConfigPath()
	configDir := filepath.Dir(configPath)

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

func (state *AppState) startCopying() {
	if state.sourceDir == "" {
		dialog.ShowError(fmt.Errorf("please select a source directory"), state.mainWindow)
		return
	}

	if state.config.TargetDir == "" {
		dialog.ShowError(fmt.Errorf("please set target directory in configuration"), state.mainWindow)
		return
	}

	if len(state.config.Subdirs) == 0 {
		dialog.ShowError(fmt.Errorf("please specify subdirectories to copy"), state.mainWindow)
		return
	}

	progressBar := widget.NewProgressBar()
	progressDialog := dialog.NewCustomWithoutButtons(
		"Copying",
		container.NewVBox(
			widget.NewLabel("Copying directories..."),
			progressBar,
		),
		state.mainWindow,
	)
	progressDialog.Show()

	copy.Copy(sourcePath, targetPath, func(copiedFiles, totalFiles int) {
		progressBar.SetValue((float64(copiedFiles) / float64(totalFiles)))
	})

}

func main() {
	a := app.New()
	win := a.NewWindow("Directory Copier")
	state := &AppState{
		config:     loadConfig(),
		mainWindow: win,
	}

	// Config section
	targetDirEntry := widget.NewEntry()
	targetDirEntry.SetText(state.config.TargetDir)

	targetDirButton := widget.NewButton("Browse", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			targetDirEntry.SetText(uri.Path())
			state.config.TargetDir = uri.Path()
		}, win)
	})

	subdirsEntry := widget.NewMultiLineEntry()
	subdirsEntry.SetText(strings.Join(state.config.Subdirs, "\n"))

	saveConfigBtn := widget.NewButton("Save Configuration", func() {
		state.config.TargetDir = targetDirEntry.Text
		state.config.Subdirs = strings.Split(subdirsEntry.Text, "\n")
		// Clean empty lines
		var cleanSubdirs []string
		for _, s := range state.config.Subdirs {
			if strings.TrimSpace(s) != "" {
				cleanSubdirs = append(cleanSubdirs, strings.TrimSpace(s))
			}
		}
		state.config.Subdirs = cleanSubdirs

		if err := saveConfig(state.config); err != nil {
			dialog.ShowError(err, win)
			return
		}
		dialog.ShowInformation("Success", "Configuration saved!", win)
	})

	// Operation section
	state.status = widget.NewLabel("Select source directory to begin")
	state.status.Wrapping = fyne.TextWrapWord

	selectSourceBtn := widget.NewButton("Select Source Directory", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			state.sourceDir = uri.Path()
			state.status.SetText(fmt.Sprintf("Selected source directory: %s\nClick 'Start Copying' to proceed", state.sourceDir))
		}, win)
	})

	startCopyBtn := widget.NewButton("Start Copying", state.startCopying)

	// Layout
	configBox := container.NewVBox(
		widget.NewLabel("Target Directory:"),
		container.NewBorder(nil, nil, nil, targetDirButton, targetDirEntry),
		widget.NewLabel("Subdirectories to Copy:"),
		subdirsEntry,
		saveConfigBtn,
	)

	configFrame := widget.NewCard("Configuration", "", configBox)

	operationBox := container.NewVBox(
		state.status,
		selectSourceBtn,
		startCopyBtn,
	)

	operationFrame := widget.NewCard("Operation", "", operationBox)

	content := container.NewVBox(
		configFrame,
		operationFrame,
	)

	win.SetContent(container.NewPadded(content))
	win.Resize(fyne.NewSize(600, 400))
	win.ShowAndRun()
}
