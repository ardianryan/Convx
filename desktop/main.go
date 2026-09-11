package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:             "Convx Desktop",
		Width:             1280,
		Height:            820,
		MinWidth:          900,
		MinHeight:         600,
		DisableResize:     false,
		Fullscreen:        false,
		Frameless:         false,
		StartHidden:       false,
		HideWindowOnClose: false,
		BackgroundColour:  &options.RGBA{R: 16, G: 18, B: 26, A: 255},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: app.startup,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  true,
			BackdropType:         windows.Mica,
			Theme:                windows.SystemDefault,
		},
		Mac: &mac.Options{
			TitleBar:             mac.TitlebarTitleInTitlebar(),
			Appearance:           mac.NSAppearanceNameVibrantDark,
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			About: &mac.AboutInfo{
				Title:   "Convx Desktop Music Player",
				Message: "Official Desktop Port by PPTI MangoTek\nLiquid Glass Music Experience",
			},
		},
	})

	if err != nil {
		log.Fatal("Error starting Convx Desktop app:", err.Error())
	}
}
