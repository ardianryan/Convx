# 🎵 Convx Web (Liquid Glass Player)

Convx Web adalah versi web mandiri dari **Convx** yang dirancang **super ringan, kencang, dan hemat memori** (idle RAM < 30 MB).
Dibangun dengan backend **Go (Golang)** sebagai audio proxy & Innertube engine, serta frontend **Svelte 5 + Vite + Tailwind CSS** dengan tema khas **Liquid Glass**.

---

## ⚡ Fitur Utama

- **Ultra Lightweight:** Single binary Go berukuran hanya ~10 MB, ukuran container Docker ~25 MB.
- **Liquid Glass UI:** Antarmuka modern dengan efek frosted glass (`backdrop-blur`), animasi transisi mulus, dan tema gelap adaptif.
- **Bypass CORS & Anti-Bot YouTube:** Backend Go menggunakan emulasi client YouTube `ANDROID_VR` dan `IOS` fallback dari Convx Android serta reverse proxy audio streaming dengan dukungan HTTP Range Request (`206 Partial Content`).
- **MediaSession API:** Dukungan tombol keyboard multimedia dan kontrol notifikasi/lockscreen.
- **Antrean Lagu (Queue):** Manajemen playlist dan antrean lagu yang reaktif.

---

## 🔌 Konfigurasi Port

- **Port `7554` (Default Production / Server):** Port server Go lengkap (API + Frontend Web yang di-embed).
- **Port `5147` (Default Frontend Dev):** Port dev server Vite saat pengembangan frontend.

---

## 🐳 Menjalankan dengan Docker

### 1. Jalankan langsung dari GitHub Container Registry (GHCR):
```bash
docker run -d \
  --name convx-web \
  --restart unless-stopped \
  -p 7554:7554 \
  ghcr.io/ardianryan/convx-web:latest
```
Buka di browser: `http://localhost:7554`

### 2. Jalankan via Docker Compose:
```bash
cd web
docker compose up -d
```

---

## 🛠️ Pengembangan Lokal (Local Development)

### Prasyarat
- Go 1.22+
- Node.js 18+

### Menjalankan Backend & Frontend Bersama:
1. **Terminal 1: Jalankan Backend Go (Port 7554)**
   ```bash
   cd web/backend
   go run .
   ```

2. **Terminal 2: Jalankan Frontend Svelte Dev (Port 5147)**
   ```bash
   cd web/frontend
   npm install
   npm run dev
   ```
   Buka di browser: `http://localhost:5147` (otomatis mem-proxy request API ke `http://localhost:7554`).

### Build Single Binary Mandiri:
```bash
# 1. Build frontend ke backend/dist
cd web/frontend && npm run build

# 2. Compile binary Go
cd ../backend && CGO_ENABLED=0 go build -ldflags="-s -w" -o convx-web .

# 3. Jalankan binary tunggal!
./convx-web
```
Binary `./convx-web` sudah memuat seluruh frontend dan backend di dalam satu file!
