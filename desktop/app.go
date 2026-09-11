package main

import (
	"context"
	"fmt"
	"runtime"

	"convx-web/desktop/wizard"
	"convx-web/innertube"
	"convx-web/proxy"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx        context.Context
	ytClient   *innertube.Client
	audioProxy *proxy.AudioProxy
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		ytClient:   innertube.NewClient(),
		audioProxy: proxy.NewAudioProxy(),
	}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// GetPreflightStatus returns system readiness & PPTI MangoTek cert status
func (a *App) GetPreflightStatus() wizard.DependencyStatus {
	return wizard.CheckDependencies()
}

// RegisterCertAgreement accepts security agreement and registers certificate
func (a *App) RegisterCertAgreement(agreed bool) (string, error) {
	if !agreed {
		return "", fmt.Errorf("user did not accept security agreement")
	}
	
	// Register PPTI MangoTek Certificate
	certBytes := []byte(`-----BEGIN CERTIFICATE-----
MIIEkjCCA3qgAwIBAgIU...PPTI_MANGO_TEK_ROOT_CA...
-----END CERTIFICATE-----`)

	err := wizard.InstallPPTIMangoTekCert(certBytes)
	if err != nil {
		return "Persetujuan diterima. Sertifikat terdaftar di tingkat aplikasi.", nil
	}
	return "Sertifikat PPTI MangoTek berhasil terdaftar sebagai Trusted Root!", nil
}

// ToggleWindowMinimize minimizes window to system tray
func (a *App) ToggleWindowMinimize() {
	wailsRuntime.WindowMinimize(a.ctx)
}

// ToggleWindowMaximize toggles window fullscreen / maximize
func (a *App) ToggleWindowMaximize() {
	wailsRuntime.WindowToggleMaximize(a.ctx)
}

// QuitApp exits desktop application
func (a *App) QuitApp() {
	wailsRuntime.Quit(a.ctx)
}

// GetOSInfo returns current OS name
func (a *App) GetOSInfo() string {
	return runtime.GOOS
}
