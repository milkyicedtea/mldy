package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"

	"mldy/internal/app"
)

// Wails embeds the built frontend (frontend/dist) into the binary.
//
//go:embed all:frontend/dist
var assets embed.FS

func main() {
	queueSvc := app.NewService()
	depsSvc := app.NewDepsService()

	// Register events so the binding generator produces typed frontend APIs.
	application.RegisterEvent[app.State]("state")
	application.RegisterEvent[string]("deps:log")

	a := application.New(application.Options{
		Name:        "mldy",
		Description: "Download videos with yt-dlp",
		Services: []application.Service{
			application.NewService(queueSvc),
			application.NewService(depsSvc),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})


	a.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "mldy",
		Width:            900,
		Height:           640,
		DevToolsEnabled:  true,
		URL:              "/",
		BackgroundColour: application.NewRGB(17, 17, 24),
	})

	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}
