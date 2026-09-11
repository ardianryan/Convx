# Product Requirements Document (PRD): Convx Desktop & Monorepo Ecosystem

> **Version**: 1.0.0  
> **Status**: Approved & Deployed  
> **Publisher & Maintainer**: PPTI MangoTek  
> **Repository**: [https://github.com/ardianryan/Convx-Desktop](https://github.com/ardianryan/Convx-Desktop)

---

## 1. Executive Summary

Convx was originally created as an Android music player streaming from YouTube Music with a custom **Liquid Glass UI**. 

This PRD defines the expansion of Convx into a **Cross-Platform Monorepo Ecosystem**, featuring:
1. **Convx Desktop**: Native Desktop Application for **macOS (Universal)**, **Windows (x64)**, and **Linux (x64)** built with **Wails v2** and **Svelte 5**.
2. **Shared Web Frontend**: Unified Svelte 5 UI codebase serving both browser users (`web/`) and native desktop clients (`desktop/`).
3. **PPTI MangoTek Security & Trust Standard**: Pre-flight dependency checker, interactive terms agreement, self-signed Root CA (`O=PPTI MangoTek`), and automated trust registration scripts (`install-trust.bat` / `install-trust.sh`).

---

## 2. Goals & Key Objectives

- **Unified UI**: Single source of truth for Web and Desktop user interfaces to eliminate UI drift.
- **Native Experience**: Native OS frameless Liquid Glass titlebar (`TitleBarHiddenInset` on macOS, `Mica` backdrop on Windows).
- **Security Compliance**: Pre-flight setup wizard enforcing `[✓] I accept PPTI MangoTek Certificate & Terms` before runtime initialization.
- **Automated CI/CD**: Matrix build & release pipeline publishing clean `.zip` bundles to GitHub Releases upon push to `main`.

---

## 3. Architecture & Monorepo Structure

```
convx/
├── desktop/                  # Wails v2 Native Application Wrapper
│   ├── main.go               # Entrypoint (Mica backdrop, Liquid Glass titlebar)
│   ├── app.go                # Wails JS Bridge & Certificate Agreement Handler
│   ├── wails.json            # Config ("frontend:dir": "../web/frontend")
│   └── wizard/               # Pre-flight dependency checker & data setup
├── web/                      # Web Application & Cloud Infrastructure
│   ├── frontend/             # Svelte 5 + Tailwind CSS + Vite (SHARED UI)
│   ├── backend/              # Go HTTP/WebSocket REST API Server
│   └── Dockerfile            # Production Docker image build
├── scripts/certs/            # PPTI MangoTek Security & Trust Tools
│   ├── generate-certs.sh     # OpenSSL Root CA & Code Signing Cert generator
│   ├── install-trust.bat     # Windows Administrator Root CA trust installer
│   ├── install-trust.sh      # macOS/Linux System Keychain trust installer
│   └── sign-binary.sh        # Code signing helper (osslsigncode / codesign)
└── .github/workflows/
    ├── desktop-release.yml   # Multi-platform Wails matrix release pipeline
    └── docker-publish.yml    # Docker container build & publish workflow
```

---

## 4. Key Functional Requirements

### 4.1 Shared Frontend Architecture
- The Svelte 5 codebase located at `web/frontend` **MUST** serve as the sole UI codebase for both Web and Desktop applications.
- `desktop/wails.json` links directly to `../web/frontend`.
- Wails embeds the output of `web/frontend/dist` directly into the compiled native desktop executable.

### 4.2 Pre-Flight Setup & Security Agreement Wizard
- Upon first launch of Convx Desktop, the application checks OS dependencies:
  - **Windows**: WebView2 Runtime check.
  - **Linux**: WebKitGTK (`libwebkit2gtk-4.1-dev`) runtime check.
- Shows an explicit interactive agreement dialog:
  - `[✓] I accept PPTI MangoTek Certificate & Terms`
- Persists user acceptance in `%APPDATA%/Convx` (Windows), `~/Library/Application Support/Convx` (macOS), or `~/.config/convx` (Linux).

### 4.3 PPTI MangoTek Code Signing & Certificate Trust
- All desktop releases must be signed with identity `O=PPTI MangoTek`.
- Distribution `.zip` packages must contain:
  1. Signed Desktop Application executable/bundle.
  2. `ppti-mangotek-rootca.crt` (Root CA Certificate).
  3. `install-trust.bat` (Windows) or `install-trust.sh` (macOS/Linux) script for silent trusted publisher registration.

### 4.4 Automated CI/CD Release Pipeline
- Triggered automatically on push to `main` when `desktop/**` or `scripts/certs/**` are modified.
- Matrix runners: `windows-latest`, `macos-latest`, `ubuntu-latest`.
- Compression tools:
  - **Windows**: Uses `7z a` (7-Zip) because `zip` command is absent in Windows bash environment.
  - **macOS/Linux**: Uses `zip -r`.
- Publishes release archives directly to GitHub Release tag `v1.0.0-desktop`.

---

## 5. Non-Functional Requirements

- **Performance**: Startup time < 1.5s on desktop platforms.
- **Memory Footprint**: System RAM usage < 150MB when idle.
- **Compatibility**:
  - Windows 10/11 (x64)
  - macOS 12+ (Apple Silicon & Intel Universal)
  - Linux (Ubuntu 22.04+, Fedora, Arch)
  - Android 8.0+ (Jetpack Compose App)
