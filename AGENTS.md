# AGENTS.md — Convx Monorepo Architecture & Agent Memory

> **IMPORTANT FOR ALL AI ASSISTANTS & SUBAGENTS**:  
> Read this document to understand the codebase layout, rules, and constraints of the Convx ecosystem before modifying any files.

---

## 1. Core Architecture Overview

Convx is a multi-platform music streaming monorepo:
1. **Android App (`app/`)**: Built with Jetpack Compose & Media3 ExoPlayer.
2. **Web Stack (`web/`)**:
   - `web/frontend/`: **Svelte 5 + Vite + Tailwind CSS**. This is the **SINGLE SOURCE OF TRUTH** for all UI.
   - `web/backend/`: Go HTTP/WebSocket REST API server for web hosting.
3. **Desktop App (`desktop/`)**:
   - **Wails v2** wrapper targeting Windows, macOS, and Linux.
   - Uses `web/frontend` directly via `"frontend:dir": "../web/frontend"` in `desktop/wails.json`.
   - **DO NOT CREATE SEPARATE FRONTEND CODE IN `desktop/`**.
4. **Security & Certs (`scripts/certs/`)**:
   - Identity: `O=PPTI MangoTek`.
   - Certificates: Root CA (`ppti-mangotek-rootca.crt`) & Code Signing cert (`ppti-mangotek-codesign.pfx`).
   - Trust Installers: `install-trust.bat` (Windows) and `install-trust.sh` (macOS/Linux).

---

## 2. Mandatory Coding Rules & Constraints

### Rule 1: Single Source of Truth for UI & Embed Asset Sync
- **NEVER** build or duplicate UI code inside `desktop/`. 
- All UI features, components, and pages **MUST** be written inside `web/frontend/`.
- `desktop/` only contains Wails native options (`main.go`), Wails JS bindings (`app.go`), pre-flight checks (`wizard/`), and OS build configs (`wails.json`, `build/`).
- `desktop/main.go` embeds `frontend/dist`. During CI or manual builds, `web/backend/dist` (Vite's `outDir`) **MUST** be copied into `desktop/frontend/dist` before invoking `wails build` so `go:embed` embeds the actual `index.html` and assets.

### Rule 2: CI/CD Packaging Tooling & Shell Compatibility
- All workflow steps in `.github/workflows/desktop-release.yml` MUST specify `shell: bash` so commands like `rm -rf` and `cp -r` execute consistently across Windows, macOS, and Linux runners (avoiding PowerShell parameter syntax errors).
- **Windows Runner (`windows-latest`)**: MUST use `7z a` for creating zip archives because the `zip` command is NOT available in the default Windows Git Bash runner environment.
- **macOS / Linux Runners**: Use `zip -r` for Linux, and `hdiutil` for macOS DMG creation.

### Rule 3: PPTI MangoTek Certificate Compliance
- Convx Desktop requires user acceptance of the PPTI MangoTek certificate agreement.
- The state handler is implemented in `desktop/app.go` (`IsCertificateAccepted`, `AcceptCertificate`).
- Do not bypass or remove this check during desktop app initialization.

### Rule 4: Isolated GitHub Actions Workflows
- Android workflows (`build.yml`, `nightly.yml`, `build_pr.yml`) are restricted to `workflow_dispatch` or Android-specific paths to avoid unnecessary Android builds on desktop pushes.
- Desktop releases are handled by `.github/workflows/desktop-release.yml`.
- Web Docker releases are handled by `.github/workflows/docker-publish.yml`.

---

## 3. Key File Locations

- **PRD**: [docs/PRD.md](file:///Users/ardianryan/Documents/convx/docs/PRD.md)
- **Wails Config**: [desktop/wails.json](file:///Users/ardianryan/Documents/convx/desktop/wails.json)
- **Desktop Entrypoint**: [desktop/main.go](file:///Users/ardianryan/Documents/convx/desktop/main.go)
- **Shared UI Root**: [web/frontend/](file:///Users/ardianryan/Documents/convx/web/frontend/)
- **Release CI Workflow**: [.github/workflows/desktop-release.yml](file:///Users/ardianryan/Documents/convx/.github/workflows/desktop-release.yml)
- **Trust Script (Windows)**: [scripts/certs/install-trust.bat](file:///Users/ardianryan/Documents/convx/scripts/certs/install-trust.bat)
- **Trust Script (macOS/Linux)**: [scripts/certs/install-trust.sh](file:///Users/ardianryan/Documents/convx/scripts/certs/install-trust.sh)
