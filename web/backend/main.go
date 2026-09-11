package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"convx-web/innertube"
	"convx-web/proxy"
)

//go:embed all:dist
var distFS embed.FS

var (
	ytClient   *innertube.Client
	audioProxy *proxy.AudioProxy
)

func main() {
	ytClient = innertube.NewClient()
	audioProxy = proxy.NewAudioProxy()

	port := os.Getenv("PORT")
	if port == "" {
		port = "7555" // Default internal port when running behind Node gateway
	}

	mux := http.NewServeMux()

	// API Endpoints
	mux.HandleFunc("GET /api/health", handleHealth)
	mux.HandleFunc("POST /api/internal/set-relay", handleInternalSetRelay)
	mux.HandleFunc("GET /api/internal/relay-status", handleInternalRelayStatus)
	mux.HandleFunc("GET /api/account/status", handleAccountStatus)
	mux.HandleFunc("POST /api/account/cookie", handleSetCookie)
	mux.HandleFunc("POST /api/account/logout", handleLogout)
	mux.HandleFunc("GET /api/search", handleSearch)
	mux.HandleFunc("GET /api/stream/{videoId}", handleStream)
	mux.HandleFunc("GET /api/proxy/audio/{videoId}", handleProxyVideo)
	mux.HandleFunc("GET /api/proxy/audio", handleProxyAudio)
	mux.HandleFunc("GET /api/lyrics", handleLyrics)

	// Static Files (Svelte Frontend)
	staticFS, err := fs.Sub(distFS, "dist")
	if err != nil {
		log.Fatalf("Failed to initialize static file server: %v", err)
	}

	fileServer := http.FileServer(http.FS(staticFS))
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		// If path doesn't have an extension and isn't root, serve index.html for SPA routing
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path != "" && !strings.Contains(path, ".") {
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})

	// Wrap with CORS & Logging Middleware
	handler := corsMiddleware(mux)

	log.Printf("✨ Convx Web Server running at http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"service": "convx-web",
		"version": "1.0.0",
	})
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		http.Error(w, `{"error":"query param 'q' is required"}`, http.StatusBadRequest)
		return
	}

	songs, err := ytClient.Search(query)
	if err != nil {
		log.Printf("Search error for '%s': %v", query, err)
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"query":   query,
		"results": songs,
		"count":   len(songs),
	})
}

func handleStream(w http.ResponseWriter, r *http.Request) {
	videoID := r.PathValue("videoId")
	if videoID == "" {
		http.Error(w, `{"error":"missing videoId"}`, http.StatusBadRequest)
		return
	}

	streamInfo, err := ytClient.GetStream(videoID)
	if err != nil {
		log.Printf("GetStream error for %s: %v", videoID, err)
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	// Create local proxy URL
	proxyURL := fmt.Sprintf("/api/proxy/audio/%s", videoID)
	streamInfo.ProxyURL = proxyURL

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(streamInfo)
}

func handleProxyVideo(w http.ResponseWriter, r *http.Request) {
	videoID := r.PathValue("videoId")
	audioProxy.ServeVideo(w, r, videoID, func(id string) (string, error) {
		info, err := ytClient.GetStream(id)
		if err != nil {
			return "", err
		}
		return info.StreamURL, nil
	})
}

func handleProxyAudio(w http.ResponseWriter, r *http.Request) {
	audioProxy.ServeHTTP(w, r)
}

func handleLyrics(w http.ResponseWriter, r *http.Request) {
	title := strings.TrimSpace(r.URL.Query().Get("title"))
	artist := strings.TrimSpace(r.URL.Query().Get("artist"))

	if title == "" {
		http.Error(w, `{"error":"title parameter required"}`, http.StatusBadRequest)
		return
	}

	// Clean up YouTube title suffixes like (Official Music Video), [Lyric Video], etc.
	cleanTitle := title
	patterns := []string{
		`(?i)\s*[\(\[](official\s*)?(music\s*)?video[\)\]]`,
		`(?i)\s*[\(\[](official\s*)?(lyric\s*)?video[\)\]]`,
		`(?i)\s*[\(\[](official\s*)?audio[\)\]]`,
		`(?i)\s*[\(\[]lirik[\)\]]`,
		`(?i)\s*[\(\[]lyrics[\)\]]`,
		`(?i)\s*[\(\[]visualizer[\)\]]`,
	}
	for _, p := range patterns {
		cleanTitle = regexp.MustCompile(p).ReplaceAllString(cleanTitle, "")
	}
	cleanTitle = strings.TrimSpace(cleanTitle)

	cleanArtist := artist
	cleanArtist = strings.TrimSuffix(cleanArtist, " - Topic")

	// Try querying LRCLIB
	lrclibURL := fmt.Sprintf("https://lrclib.net/api/search?track_name=%s&artist_name=%s",
		url.QueryEscape(cleanTitle), url.QueryEscape(cleanArtist))

	req, _ := http.NewRequest("GET", lrclibURL, nil)
	req.Header.Set("User-Agent", "Convx-Apple-Player/1.0 (https://github.com/ardianryan/convx)")

	client := &http.Client{Timeout: 6 * time.Second}
	resp, err := client.Do(req)

	var items []map[string]interface{}
	if err == nil && resp.StatusCode == http.StatusOK {
		_ = json.NewDecoder(resp.Body).Decode(&items)
		resp.Body.Close()
	}

	// Fallback to broader search query if specific track/artist lookup returned empty
	if len(items) == 0 {
		fallbackURL := fmt.Sprintf("https://lrclib.net/api/search?q=%s",
			url.QueryEscape(cleanTitle+" "+cleanArtist))
		req2, _ := http.NewRequest("GET", fallbackURL, nil)
		req2.Header.Set("User-Agent", "Convx-Apple-Player/1.0 (https://github.com/ardianryan/convx)")
		if resp2, err2 := client.Do(req2); err2 == nil && resp2.StatusCode == http.StatusOK {
			_ = json.NewDecoder(resp2.Body).Decode(&items)
			resp2.Body.Close()
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if len(items) > 0 {
		_ = json.NewEncoder(w).Encode(items[0])
		return
	}

	w.WriteHeader(http.StatusNotFound)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "Lyrics not found"})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Range, Authorization, X-Requested-With, Accept, Origin")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Range, Content-Length, Accept-Ranges")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func handleAccountStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	hasCookie := ytClient.HasCookie()
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"isLoggedIn": hasCookie,
		"hasCookie":  hasCookie,
	})
}

func handleSetCookie(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Cookie string `json:"cookie"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	cookie := strings.TrimSpace(body.Cookie)
	if cookie == "" {
		http.Error(w, `{"error":"cookie cannot be empty"}`, http.StatusBadRequest)
		return
	}

	ytClient.SetCookie(cookie, true)
	log.Printf("[Account] YouTube cookie updated successfully")

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "ok",
		"isLoggedIn": true,
	})
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	ytClient.ClearCookie()
	log.Printf("[Account] YouTube account logged out (cookie cleared)")

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "ok",
		"isLoggedIn": false,
	})
}

func handleInternalSetRelay(w http.ResponseWriter, r *http.Request) {
	var body struct {
		URL string `json:"url"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	ytClient.SetRelayURL(body.URL)
	audioProxy.SetRelayURL(body.URL)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"relay":  body.URL,
	})
}

func handleInternalRelayStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"relayUrl": ytClient.GetRelayURL(),
	})
}
