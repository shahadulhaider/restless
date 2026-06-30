package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/shahadulhaider/restless/internal/gui"
	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:dist
var assets embed.FS

//go:embed appicon.png
var appIcon []byte

func emitMenuEvent(app *application.App, name string) func(*application.Context) {
	return func(_ *application.Context) {
		app.Event.Emit(name)
	}
}

func buildMenu(app *application.App) *application.Menu {
	menu := application.NewMenu()

	if runtime.GOOS == "darwin" {
		menu.AddRole(application.AppMenu)
	}

	fileMenu := menu.AddSubmenu("File")
	fileMenu.Add("New Request").
		SetAccelerator("CmdOrCtrl+N").
		OnClick(emitMenuEvent(app, "menu:new-request"))
	fileMenu.Add("Open Collection").
		SetAccelerator("CmdOrCtrl+O").
		OnClick(emitMenuEvent(app, "menu:open-collection"))
	fileMenu.AddSeparator()
	fileMenu.Add("Import").
		SetAccelerator("CmdOrCtrl+I").
		OnClick(emitMenuEvent(app, "menu:import"))
	fileMenu.AddSeparator()
	if runtime.GOOS != "darwin" {
		fileMenu.Add("Quit").
			SetAccelerator("CmdOrCtrl+Q").
			OnClick(func(_ *application.Context) { app.Quit() })
	}

	editMenu := menu.AddSubmenu("Edit")
	editMenu.Add("Undo").SetAccelerator("CmdOrCtrl+Z").SetRole(application.Undo)
	editMenu.Add("Redo").SetAccelerator("CmdOrCtrl+Shift+Z").SetRole(application.Redo)
	editMenu.AddSeparator()
	editMenu.Add("Cut").SetAccelerator("CmdOrCtrl+X").SetRole(application.Cut)
	editMenu.Add("Copy").SetAccelerator("CmdOrCtrl+C").SetRole(application.Copy)
	editMenu.Add("Paste").SetAccelerator("CmdOrCtrl+V").SetRole(application.Paste)
	editMenu.Add("Select All").SetAccelerator("CmdOrCtrl+A").SetRole(application.SelectAll)

	requestMenu := menu.AddSubmenu("Request")
	requestMenu.Add("Send Request").
		SetAccelerator("CmdOrCtrl+Return").
		OnClick(emitMenuEvent(app, "menu:send-request"))
	requestMenu.Add("Generate Code").
		SetAccelerator("CmdOrCtrl+Shift+G").
		OnClick(emitMenuEvent(app, "menu:generate-code"))
	requestMenu.Add("Copy as cURL").
		SetAccelerator("CmdOrCtrl+Shift+C").
		OnClick(emitMenuEvent(app, "menu:copy-curl"))

	viewMenu := menu.AddSubmenu("View")
	viewMenu.Add("Toggle Sidebar").
		SetAccelerator("CmdOrCtrl+B").
		OnClick(emitMenuEvent(app, "menu:toggle-sidebar"))
	viewMenu.Add("Toggle Theme").
		SetAccelerator("CmdOrCtrl+Shift+T").
		OnClick(emitMenuEvent(app, "menu:toggle-theme"))
	viewMenu.AddSeparator()
	viewMenu.AddRole(application.Reload)
	viewMenu.AddRole(application.ForceReload)
	viewMenu.AddRole(application.ToggleFullscreen)

	helpMenu := menu.AddSubmenu("Help")
	helpMenu.Add("Keyboard Shortcuts").
		SetAccelerator("CmdOrCtrl+Shift+?").
		OnClick(emitMenuEvent(app, "menu:keyboard-shortcuts"))
	helpMenu.AddSeparator()
	helpMenu.AddRole(application.About)

	return menu
}

func setupSystemTray(app *application.App) {
	tray := app.SystemTray.New()
	tray.SetIcon(appIcon).SetTooltip("Restless")

	trayMenu := application.NewMenu()
	trayMenu.Add("Show / Hide").OnClick(func(_ *application.Context) {
		app.Event.Emit("tray:toggle-window")
	})
	trayMenu.AddSeparator()
	trayMenu.Add("Quit").OnClick(func(_ *application.Context) {
		app.Quit()
	})
	tray.SetMenu(trayMenu)
}

func main() {
	dir := "."
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		log.Fatal(err)
	}

	app := application.New(application.Options{
		Name:        "Restless",
		Description: "Your API workbench — desktop edition",
		Services: []application.Service{
			application.NewService(&GreetService{rootDir: abs}),
			application.NewService(&gui.CollectionService{}),
			application.NewService(&gui.EnvironmentService{}),
			application.NewService(&gui.ExporterService{}),
			application.NewService(&gui.ImporterService{}),
			application.NewService(&gui.HistoryService{}),
			application.NewService(gui.NewRequestService()),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	app.Menu.SetApplicationMenu(buildMenu(app))
	setupSystemTray(app)

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "Restless",
		Width:  1200,
		Height: 800,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(30, 30, 46),
		URL:              "/",
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
