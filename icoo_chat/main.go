package main

import (
	"embed"
	"icoo_chat/internal/services"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := services.NewApp()

	err := wails.Run(&options.App{
		Title:  "icoo_chat",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.Startup,
		Bind: []interface{}{
			app,
		},
		Frameless: true,
		Windows: &windows.Options{
			DisableFramelessWindowDecorations: true,
			WebviewIsTransparent:              true,
			WindowIsTranslucent:               true,
		},
	})

	if err != nil {
		println("错误:", err.Error())
	}
}
