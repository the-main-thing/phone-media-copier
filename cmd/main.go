package main

import (
	"fmt"
	"phone-media-copier/internal/android"
	"phone-media-copier/internal/update"

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
	m.setButtonsEnabled(false)
	m.progress.SetValue(0)

	go func() {
		err := android.Copy("target/dir", func(progress int) {
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
	m.setButtonsEnabled(false)
	m.progress.SetValue(0)

	go func() {
		err := update.Update()
		if err != nil {
			dialog.ShowError(err, m.window)
			m.setButtonsEnabled(true)
		} else {
			// Only enable close button after successful update
			m.startBtn.Disable()
			m.updateBtn.Disable()
			m.closeBtn.Enable()
		}
	}()
}

func (m *MainApp) setButtonsEnabled(enabled bool) {
	m.startBtn.Enable()
	m.updateBtn.Enable()
	m.closeBtn.Enable()
	if !enabled {
		m.startBtn.Disable()
		m.updateBtn.Disable()
		m.closeBtn.Disable()
	}
}

func main() {
	app := newMainApp()
	app.window.Resize(fyne.NewSize(300, 300))
	app.window.ShowAndRun()
}
