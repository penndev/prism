package internal

import (
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

var MainWindow application.Window
var MainWindowX, MainWindowY int

func MainWindowSetPosition() {
	if MainWindow.IsVisible() {
		MainWindowX, MainWindowY = MainWindow.Position()
		MainWindow.Hide()
	} else {
		if MainWindowX != 0 && MainWindowY != 0 {
			MainWindow.SetPosition(MainWindowX, MainWindowY)
		}
		MainWindow.Show()
		MainWindow.Focus()
	}
}

func SetUPMainWindow() application.Window {
	MainWindow = App.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Prism",
		Width:            1000,
		Height:           800,
		MinWidth:         400,
		MinHeight:        800,
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
		Windows: application.WindowsWindow{
			HiddenOnTaskbar: false,
		},
	})

	MainWindow.SetBackgroundColour(application.NewRGBA(30, 30, 30, 255))

	MainWindow.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		MainWindowX, MainWindowY = MainWindow.Position()
		MainWindow.Hide()
		e.Cancel()
	})
	return MainWindow
}
