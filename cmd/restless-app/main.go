package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"

	"github.com/shahadulhaider/restless/internal/gui"
	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:dist
var assets embed.FS

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
