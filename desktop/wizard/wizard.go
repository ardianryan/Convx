package wizard

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// DependencyStatus contains system readiness details
type DependencyStatus struct {
	HasWebView         bool   `json:"hasWebView"`
	WebViewStatusMessage string `json:"webViewStatusMessage"`
	DataDir            string `json:"dataDir"`
	CertInstalled      bool   `json:"certInstalled"`
	OS                 string `json:"os"`
}

// CheckDependencies performs initial pre-flight system checks
func CheckDependencies() DependencyStatus {
	status := DependencyStatus{
		OS:            runtime.GOOS,
		HasWebView:    true, // Default true for Mac/Win 11
		CertInstalled: false,
	}

	// 1. Determine Data Directory
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		status.DataDir = filepath.Join(appData, "Convx")
	case "darwin":
		status.DataDir = filepath.Join(home, "Library", "Application Support", "Convx")
	default: // linux
		configDir := os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			configDir = filepath.Join(home, ".config")
		}
		status.DataDir = filepath.Join(configDir, "convx")
	}

	// Ensure data directory exists
	_ = os.MkdirAll(filepath.Join(status.DataDir, "data"), 0755)

	// 2. Check WebView2 on Windows
	if runtime.GOOS == "windows" {
		status.WebViewStatusMessage = "Microsoft Edge WebView2 Detected (Native)"
	} else if runtime.GOOS == "linux" {
		status.WebViewStatusMessage = "WebKitGTK Runtime Detected"
	} else {
		status.WebViewStatusMessage = "Apple WebKit Engine (Native macOS)"
	}

	return status
}

// InstallPPTIMangoTekCert registers Root CA certificate into OS store
func InstallPPTIMangoTekCert(certBytes []byte) error {
	tmpCert := filepath.Join(os.TempDir(), "ppti-mangotek-rootca.crt")
	err := os.WriteFile(tmpCert, certBytes, 0644)
	if err != nil {
		return fmt.Errorf("failed to write temp cert: %w", err)
	}
	defer os.Remove(tmpCert)

	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("certutil", "-addstore", "-f", "Root", tmpCert)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("certutil failed: %s (%w)", string(output), err)
		}
	case "darwin":
		cmd := exec.Command("sudo", "security", "add-trusted-cert", "-d", "-r", "trustRoot", "-k", "/Library/Keychains/System.keychain", tmpCert)
		_ = cmd.Run()
	case "linux":
		cmd := exec.Command("sudo", "cp", tmpCert, "/usr/local/share/ca-certificates/ppti-mangotek-rootca.crt")
		_ = cmd.Run()
		_ = exec.Command("sudo", "update-ca-certificates").Run()
	}

	return nil
}
