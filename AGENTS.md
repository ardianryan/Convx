# AGENTS.md — Convx Web Architecture & Agent Memory

> **IMPORTANT FOR ALL AI ASSISTANTS & SUBAGENTS**:  
> Read this document to understand the codebase layout, rules, and constraints of the Convx Web ecosystem before modifying any files.

---

## 1. Core Architecture Overview

Convx Web (`Convx-Web`) is a high-performance web music streaming platform:
1. **Web Stack (`web/`)**:
   - `web/frontend/`: **Svelte 5 + Vite + Tailwind CSS**. This is the **SINGLE SOURCE OF TRUTH** for all UI.
   - `web/backend/`: Go HTTP/WebSocket REST API server for audio proxying, search, and system settings (`http://localhost:7554`).
2. **Security & Certs (`scripts/certs/`)**:
   - Identity: `O=PPTI MangoTek`.
   - Certificates: Root CA (`ppti-mangotek-rootca.crt`) & Code Signing cert (`ppti-mangotek-codesign.pfx`).

---

## 2. Mandatory Coding Rules & Constraints

### Rule 1: Single Source of Truth for UI
- All UI features, components, and pages **MUST** be written inside `web/frontend/`.
- Native HTML5 `<audio>` element with Go AudioProxy (`/api/proxy/audio/{videoId}`) is the primary and sole playback engine.

### Rule 2: Isolated GitHub Actions Workflows
- Web Docker releases are handled by `.github/workflows/docker-publish.yml`.

---

## 3. Key File Locations

- **PRD**: [docs/PRD.md](file:///Users/ardianryan/Documents/convx/docs/PRD.md)
- **Shared UI Root**: [web/frontend/](file:///Users/ardianryan/Documents/convx/web/frontend/)
- **Go Backend Server**: [web/backend/main.go](file:///Users/ardianryan/Documents/convx/web/backend/main.go)
- **Audio Proxy**: [web/backend/proxy/audio.go](file:///Users/ardianryan/Documents/convx/web/backend/proxy/audio.go)
- **InnerTube Client**: [web/backend/innertube/client.go](file:///Users/ardianryan/Documents/convx/web/backend/innertube/client.go)
