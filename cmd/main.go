package main

import (
	"fmt"
	"os"
	"path/filepath"
	"phone-media-copier/internal/android"
	"phone-media-copier/internal/update"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type MainApp struct {
	window    fyne.Window
	progress  *widget.ProgressBar
	startBtn  *widget.Button
	updateBtn *widget.Button
	closeBtn  *widget.Button
}

func newMainApp() *MainApp {
	a := app.New()
	w := a.NewWindow("File Copy & Update")

	m := &MainApp{
		window:   w,
		progress: widget.NewProgressBar(),
	}

	m.createUI()
	return m
}

func (m *MainApp) createUI() {
	m.startBtn = widget.NewButton("Start", m.handleCopy)
	m.updateBtn = widget.NewButton("Update", m.handleUpdate)
	m.closeBtn = widget.NewButton("Close", func() {
		m.window.Close()
	})

	buttons := container.NewHBox(m.startBtn, m.updateBtn, m.closeBtn)
	content := container.NewVBox(
		buttons,
		m.progress,
	)

	m.window.SetContent(content)
}

func (m *MainApp) handleCopy() {
	m.progress.SetValue(0)
	go func() {
		m.setButtonsEnabled(false)
		targetDir, err := GetUserHomeDir()
		if err != nil {
			dialog.ShowError(err, m.window)
			m.setButtonsEnabled(true)
			return
		}

		targetDir = filepath.Join(targetDir, "Telephone")

		err = android.Copy(targetDir, func(progress int) {
			m.progress.SetValue(float64(progress) / 100)
		})

		if err != nil {
			if err.Error() == "no device connected" {
				dialog.ShowError(fmt.Errorf("Make sure you've connected the device"), m.window)
			} else {
				dialog.ShowError(err, m.window)
			}
			m.setButtonsEnabled(true)
		} else {
			m.setButtonsEnabled(true)
		}
	}()
}

func (m *MainApp) handleUpdate() {
	m.progress.SetValue(0)

	go func() {
		m.setButtonsEnabled(false)
		updated, err := update.Update()
		if err != nil {
			dialog.ShowError(err, m.window)
			m.setButtonsEnabled(true)
		} else {
			if updated {
				dialog.ShowInformation("Update successful", "The application has been updated", m.window)
			} else {
				dialog.ShowInformation("No updates available", "The application is up to date", m.window)
			}
			m.setButtonsEnabled(true)
		}
	}()
}

func (m *MainApp) setButtonsEnabled(enabled bool) {
	fyne.DoAndWait(func() {
		if !enabled {
			m.startBtn.Disable()
			m.updateBtn.Disable()
			m.closeBtn.Disable()
		} else {
			m.startBtn.Enable()
			m.updateBtn.Enable()
			m.closeBtn.Enable()
		}
	})
}

func main() {
	app := newMainApp()
	app.window.Resize(fyne.NewSize(300, 300))
	app.window.ShowAndRun()
}

func GetUserHomeDir() (string, error) {
	if runtime.GOOS == "windows" {
		homeDrive := os.Getenv("HOMEDRIVE")
		homePath := os.Getenv("HOMEPATH")
		if homeDrive != "" && homePath != "" {
			return homeDrive + homePath, nil
		}
		return os.Getenv("USERPROFILE"), nil
	}
	return os.Getenv("HOME"), nil
}
