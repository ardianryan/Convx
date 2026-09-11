# Rule: Convx Desktop & Monorepo Architecture

When working on this repository, strictly adhere to the following rules:

1. **Shared UI**: All UI changes must be performed in `web/frontend/` (Svelte 5 + Vite). `desktop/` embeds `web/frontend` directly (`"frontend:dir": "../web/frontend"` in `desktop/wails.json`). Do not duplicate UI code in `desktop/`.
2. **CI/CD Zip Packaging**: In `.github/workflows/desktop-release.yml`, Windows runner (`windows-latest`) must use `7z a` instead of `zip -r` to archive artifacts.
3. **PPTI MangoTek Security Trust**: All desktop binaries are signed under `O=PPTI MangoTek`. Ensure Root CA certificate (`ppti-mangotek-rootca.crt`) and trust installer scripts (`install-trust.bat`, `install-trust.sh`) are included in all desktop release zip bundles.
4. **Pre-flight Agreement**: Maintain the PPTI MangoTek Certificate & Terms agreement handler in `desktop/app.go` & `desktop/wizard/wizard.go`.
