<div align="center">
  <h1>🌐 CONVX-WEB</h1>
  <h3>High-Performance Web Music Streaming Platform</h3>
  <p><b>Powered by Svelte 5 + Vite + Tailwind CSS & Go AudioProxy Backend</b></p>
  <p><b>Publisher: PPTI MangoTek</b></p>

  <p>
    <a href="https://github.com/ardianryan/Convx-Web/releases/tag/v1.0.0">
      <img src="https://img.shields.io/badge/%E2%AC%87%EF%B8%8F%20CONVX--WEB%20v1.0.0-007ACC?style=for-the-badge&logo=github&logoColor=white" alt="Convx-Web Release v1.0.0">
    </a>
    <a href="https://github.com/ardianryan/Convx-Web/actions/workflows/docker-publish.yml">
      <img src="https://img.shields.io/github/actions/workflow/status/ardianryan/Convx-Web/docker-publish.yml?style=for-the-badge&label=Docker%20Release%20CI" alt="Docker CI Status">
    </a>
  </p>

  <p>
    🎵 <b>Official Release:</b> <a href="https://github.com/ardianryan/Convx-Web/releases/tag/v1.0.0"><b>Convx-Web v1.0.0 Tag</b></a>
  </p>
</div>

---

## 🚀 Overview

**Convx-Web** is a modern, high-performance web music streaming application featuring a liquid glass UI design, instant YouTube Music searching, native HTML5 high-fidelity audio proxying, synced LRC lyrics, and Cloudflare Worker relay integration.

### ✨ Key Features
- 🧊 **Liquid Glass UI**: Built with Svelte 5 + Tailwind CSS for a smooth, frosted aesthetic.
- 🎧 **Native HTML5 Audio Engine**: High-fidelity, low-latency audio streaming powered by Go AudioProxy.
- 🔍 **YouTube Music Search**: Fast search queries for official tracks, albums, and artists via InnerTube API.
- 📜 **Synced Lyrics**: Integrated LRCLIB lyrics synchronization with auto-scroll highlighting.
- ⚡ **Cloudflare Worker Relay**: One-click Cloudflare Relay deployment to bypass regional audio restrictions.
- 🔑 **Session & Auth Management**: Built-in setup wizard, user login, and YouTube session cookie import.

---

## 🏗️ Architecture

- **Frontend (`web/frontend`)**: Svelte 5 + Vite + Tailwind CSS SPA.
- **Backend (`web/backend`)**: Go REST API server & Audio Proxy (`http://localhost:7554`).

```
Convx-Web/
├── web/
│   ├── frontend/         # Svelte 5 + Vite UI (Single Source of Truth)
│   ├── backend/          # Go HTTP Server & Audio Proxy
│   │   ├── main.go       # Server entrypoint
│   │   ├── innertube/    # YouTube InnerTube API Client
│   │   └── proxy/        # Audio proxy & chunk range handler
├── docs/                 # Product Specifications & PRD
└── AGENTS.md             # Architecture Rules & Memory
```

---

## 💻 Quick Start & Local Development

### Prerequisites
- **Node.js**: v18+
- **Go**: 1.22+

### 1. Run Web Backend (Go)
```bash
cd web/backend
go run main.go
```
*Backend runs on `http://localhost:7554`.*

### 2. Run Web Frontend (Svelte 5)
```bash
cd web/frontend
npm install
npm run dev
```
*Frontend dev server runs on `http://localhost:5147`.*

### 3. Build Production Dist
```bash
cd web/frontend
npm run build
```
*Built assets are automatically compiled into `web/backend/dist`.*

---

## 📄 Documentation Links

- 📋 **[PRD Document](docs/PRD.md)** — Comprehensive Product Requirements & Architecture Specifications
- 🤖 **[AGENTS.md](AGENTS.md)** — Architectural Memory & Coding Rules for AI Assistants

---

## 🛡️ License & Credits

- **Publisher**: PPTI MangoTek
- **License**: GPL-3.0

