package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"convx-web/innertube"
	"convx-web/proxy"
)

func startLocalServer(port string, ytClient *innertube.Client, audioProxy *proxy.AudioProxy) {
	mux := http.NewServeMux()

	// API Health Check
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "ok",
			"service": "convx-desktop",
			"version": "1.0.0",
		})
	})

	// Search Songs & Artists
	mux.HandleFunc("GET /api/search", func(w http.ResponseWriter, r *http.Request) {
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		if query == "" {
			http.Error(w, `{"error":"query param 'q' is required"}`, http.StatusBadRequest)
			return
		}

		songs, err := ytClient.Search(query)
		if err != nil {
			log.Printf("[Desktop Server] Search error for '%s': %v", query, err)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"query":   query,
			"results": songs,
			"count":   len(songs),
		})
	})

	// Get Stream Info
	mux.HandleFunc("GET /api/stream/{videoId}", func(w http.ResponseWriter, r *http.Request) {
		videoID := r.PathValue("videoId")
		if videoID == "" {
			http.Error(w, `{"error":"missing videoId"}`, http.StatusBadRequest)
			return
		}

		streamInfo, err := ytClient.GetStream(videoID)
		if err != nil {
			log.Printf("[Desktop Server] GetStream error for %s: %v", videoID, err)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}

		proxyURL := fmt.Sprintf("/api/proxy/audio/%s", videoID)
		streamInfo.ProxyURL = proxyURL

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(streamInfo)
	})

	// Proxy Audio Streams
	mux.HandleFunc("GET /api/proxy/audio/{videoId}", func(w http.ResponseWriter, r *http.Request) {
		videoID := r.PathValue("videoId")
		audioProxy.ServeVideo(w, r, videoID, func(id string) (string, error) {
			info, err := ytClient.GetStream(id)
			if err != nil {
				return "", err
			}
			return info.StreamURL, nil
		})
	})

	mux.HandleFunc("GET /api/proxy/audio", func(w http.ResponseWriter, r *http.Request) {
		audioProxy.ServeHTTP(w, r)
	})

	// Fetch Lyrics from LRCLIB
	mux.HandleFunc("GET /api/lyrics", handleDesktopLyrics)

	// YouTube Account & Cookie Management
	mux.HandleFunc("GET /api/account/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		hasCookie := ytClient.HasCookie()
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"isLoggedIn": hasCookie,
			"hasCookie":  hasCookie,
		})
	})

	mux.HandleFunc("POST /api/account/cookie", func(w http.ResponseWriter, r *http.Request) {
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
		log.Printf("[Desktop Server] YouTube cookie updated successfully")

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":     "ok",
			"isLoggedIn": true,
		})
	})

	mux.HandleFunc("POST /api/account/logout", func(w http.ResponseWriter, r *http.Request) {
		ytClient.ClearCookie()
		log.Printf("[Desktop Server] YouTube cookie cleared")

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":     "ok",
			"isLoggedIn": false,
		})
	})

	// Relay Control Endpoints
	mux.HandleFunc("POST /api/internal/set-relay", func(w http.ResponseWriter, r *http.Request) {
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
	})

	mux.HandleFunc("GET /api/internal/relay-status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"relayUrl": ytClient.GetRelayURL(),
		})
	})

	// Standalone Desktop Placeholders for Web-Only Endpoints
	mux.HandleFunc("POST /api/setup/init", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "ok",
			"message": "Desktop mode initialized",
		})
	})

	mux.HandleFunc("POST /api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"user":   map[string]string{"username": "admin", "role": "admin"},
		})
	})

	mux.HandleFunc("GET /api/auth/me", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"user":   map[string]string{"username": "admin", "role": "admin"},
		})
	})

	mux.HandleFunc("POST /api/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
	})

	mux.HandleFunc("GET /api/settings", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"mode": "desktop",
		})
	})

	mux.HandleFunc("POST /api/devices/heartbeat", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
	})

	// Wrap with CORS & Logging Middleware
	handler := desktopCorsMiddleware(mux)

	addr := "127.0.0.1:" + port
	log.Printf("✨ Convx Desktop Backend Server listening at http://%s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Printf("[Desktop Server] http.ListenAndServe warning: %v", err)
	}
}

func handleDesktopLyrics(w http.ResponseWriter, r *http.Request) {
	title := strings.TrimSpace(r.URL.Query().Get("title"))
	artist := strings.TrimSpace(r.URL.Query().Get("artist"))

	if title == "" {
		http.Error(w, `{"error":"title parameter required"}`, http.StatusBadRequest)
		return
	}

	cleanTitle := title
	patterns := []string{
		`(?i)\s*[\(\[](official\s*)?(music\s*)?video[\)\]]`,
		`(?i)\s*[\(\[](official\s*)?(lyric\s*)?video[\)\]]`,
		`(?i)\s*[\(\[](official\s*)?audio[\)\]]`,
		`(?i)\s*[\(\[]lirik[\)\]]`,
		`(?i)\s*[\(\[]lyrics[\)\]]`,
		`(?i)\s*[\(\[]visualizer[\)\]]`,
		`(?i)\s*[\(\[]remastered[\)\]]`,
		`(?i)\s*[\(\[]hd[\)\]]`,
		`(?i)\s*[\(\[]4k[\)\]]`,
	}
	for _, p := range patterns {
		cleanTitle = regexp.MustCompile(p).ReplaceAllString(cleanTitle, "")
	}
	cleanTitle = strings.TrimSpace(cleanTitle)

	cleanArtist := strings.TrimSuffix(artist, " - Topic")

	if strings.Contains(cleanTitle, "-") {
		parts := strings.SplitN(cleanTitle, "-", 2)
		if len(parts) == 2 {
			p0 := strings.TrimSpace(parts[0])
			p1 := strings.TrimSpace(parts[1])
			if strings.EqualFold(p0, cleanArtist) || strings.EqualFold(p0, artist) {
				cleanTitle = p1
			} else if cleanArtist == "" || strings.EqualFold(cleanArtist, "Various Artists") {
				cleanArtist = p0
				cleanTitle = p1
			}
		}
	}

	lrclibURL := fmt.Sprintf("https://lrclib.net/api/search?track_name=%s&artist_name=%s",
		url.QueryEscape(cleanTitle), url.QueryEscape(cleanArtist))

	req, _ := http.NewRequest("GET", lrclibURL, nil)
	req.Header.Set("User-Agent", "Convx-Desktop-Player/1.0 (https://github.com/ardianryan/convx)")

	client := &http.Client{Timeout: 6 * time.Second}
	resp, err := client.Do(req)

	var items []map[string]interface{}
	if err == nil && resp.StatusCode == http.StatusOK {
		_ = json.NewDecoder(resp.Body).Decode(&items)
		resp.Body.Close()
	}

	if len(items) == 0 {
		fallbackURL := fmt.Sprintf("https://lrclib.net/api/search?q=%s",
			url.QueryEscape(cleanTitle+" "+cleanArtist))
		req2, _ := http.NewRequest("GET", fallbackURL, nil)
		req2.Header.Set("User-Agent", "Convx-Desktop-Player/1.0 (https://github.com/ardianryan/convx)")
		if resp2, err2 := client.Do(req2); err2 == nil && resp2.StatusCode == http.StatusOK {
			_ = json.NewDecoder(resp2.Body).Decode(&items)
			resp2.Body.Close()
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if len(items) > 0 {
		bestItem := items[0]
		for _, item := range items {
			if sl, ok := item["syncedLyrics"].(string); ok && sl != "" {
				bestItem = item
				break
			}
		}
		_ = json.NewEncoder(w).Encode(bestItem)
		return
	}

	w.WriteHeader(http.StatusNotFound)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "Lyrics not found"})
}

func desktopCorsMiddleware(next http.Handler) http.Handler {
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
