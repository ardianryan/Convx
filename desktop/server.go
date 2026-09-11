package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"convx-web/innertube"
	"convx-web/proxy"
)

type DesktopSettings struct {
	PlatformName string      `json:"platformName"`
	UserName     string      `json:"name"`
	Username     string      `json:"username"`
	CFAccountID  string      `json:"cfAccountId"`
	CFApiToken   string      `json:"cfApiToken"`
	ActiveRelay  *RelayEntry `json:"activeRelay"`
	Relays       []RelayEntry`json:"relays"`
}

type RelayEntry struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

var (
	settingsMu       sync.RWMutex
	currentSettings  DesktopSettings
	settingsFilePath = filepath.Join("data", "desktop_settings.json")
)

func loadDesktopSettings() {
	settingsMu.Lock()
	defer settingsMu.Unlock()

	currentSettings = DesktopSettings{
		PlatformName: "Convx",
		UserName:     "Admin",
		Username:     "admin",
		Relays:       []RelayEntry{},
	}

	_ = os.MkdirAll("data", 0755)
	data, err := os.ReadFile(settingsFilePath)
	if err == nil {
		_ = json.Unmarshal(data, &currentSettings)
	}
}

func saveDesktopSettings() {
	settingsMu.Lock()
	defer settingsMu.Unlock()

	_ = os.MkdirAll("data", 0755)
	data, err := json.MarshalIndent(currentSettings, "", "  ")
	if err == nil {
		_ = os.WriteFile(settingsFilePath, data, 0644)
	}
}

func startLocalServer(port string, ytClient *innertube.Client, audioProxy *proxy.AudioProxy) {
	loadDesktopSettings()

	// Restore active relay if previously saved
	settingsMu.RLock()
	if currentSettings.ActiveRelay != nil && currentSettings.ActiveRelay.URL != "" {
		ytClient.SetRelayURL(currentSettings.ActiveRelay.URL)
		audioProxy.SetRelayURL(currentSettings.ActiveRelay.URL)
		log.Printf("[Desktop Server] Restored active relay: %s", currentSettings.ActiveRelay.URL)
	}
	settingsMu.RUnlock()

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

	// Cloudflare Relay Worker Deployment
	mux.HandleFunc("POST /api/relays/deploy", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			CFAccountID string `json:"cfAccountId"`
			CFApiToken  string `json:"cfApiToken"`
			ProjectName string `json:"projectName"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"invalid json body"}`, http.StatusBadRequest)
			return
		}

		deployURL, err := deployCloudflareRelayWorker(body.CFAccountID, body.CFApiToken, body.ProjectName)
		if err != nil {
			log.Printf("[Desktop Server] Relay deploy failed: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		newRelay := RelayEntry{
			ID:   "convx-relay",
			Name: "Cloudflare Worker Relay",
			URL:  deployURL,
		}

		ytClient.SetRelayURL(deployURL)
		audioProxy.SetRelayURL(deployURL)

		settingsMu.Lock()
		currentSettings.CFAccountID = body.CFAccountID
		currentSettings.CFApiToken = body.CFApiToken
		currentSettings.ActiveRelay = &newRelay
		currentSettings.Relays = []RelayEntry{newRelay}
		settingsMu.Unlock()
		saveDesktopSettings()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"relay":  newRelay,
		})
	})

	// Test Relay Health
	mux.HandleFunc("POST /api/relays/test", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			URL string `json:"url"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		targetURL := strings.TrimSpace(body.URL)
		if targetURL == "" {
			settingsMu.RLock()
			if currentSettings.ActiveRelay != nil {
				targetURL = currentSettings.ActiveRelay.URL
			}
			settingsMu.RUnlock()
		}

		if targetURL == "" {
			http.Error(w, `{"error":"URL relay wajib diisi"}`, http.StatusBadRequest)
			return
		}

		client := &http.Client{Timeout: 6 * time.Second}
		resp, err := client.Get(targetURL)
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok":    false,
				"error": err.Error(),
			})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "status": resp.StatusCode})
		} else {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": fmt.Sprintf("HTTP %d", resp.StatusCode)})
		}
	})

	// Toggle Active Relay
	mux.HandleFunc("POST /api/relays/toggle", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Active bool `json:"active"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)

		settingsMu.Lock()
		if !body.Active {
			currentSettings.ActiveRelay = nil
			ytClient.SetRelayURL("")
			audioProxy.SetRelayURL("")
		} else if len(currentSettings.Relays) > 0 {
			currentSettings.ActiveRelay = &currentSettings.Relays[0]
			ytClient.SetRelayURL(currentSettings.ActiveRelay.URL)
			audioProxy.SetRelayURL(currentSettings.ActiveRelay.URL)
		}
		activeRelay := currentSettings.ActiveRelay
		settingsMu.Unlock()
		saveDesktopSettings()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":      "ok",
			"activeRelay": activeRelay,
		})
	})

	// Delete Active Relay
	mux.HandleFunc("DELETE /api/relays", func(w http.ResponseWriter, r *http.Request) {
		settingsMu.Lock()
		currentSettings.ActiveRelay = nil
		currentSettings.Relays = []RelayEntry{}
		ytClient.SetRelayURL("")
		audioProxy.SetRelayURL("")
		settingsMu.Unlock()
		saveDesktopSettings()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
	})

	// System Settings
	mux.HandleFunc("GET /api/settings", func(w http.ResponseWriter, r *http.Request) {
		settingsMu.RLock()
		defer settingsMu.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"mode":         "desktop",
			"platformName": currentSettings.PlatformName,
			"name":         currentSettings.UserName,
			"username":     currentSettings.Username,
			"cfAccountId":  currentSettings.CFAccountID,
			"activeRelay":  currentSettings.ActiveRelay,
			"relays":       currentSettings.Relays,
		})
	})

	mux.HandleFunc("POST /api/settings", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			PlatformName string `json:"platformName"`
			Name         string `json:"name"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)

		settingsMu.Lock()
		if body.PlatformName != "" {
			currentSettings.PlatformName = body.PlatformName
		}
		if body.Name != "" {
			currentSettings.UserName = body.Name
		}
		settingsMu.Unlock()
		saveDesktopSettings()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
	})

	// Onboarding Wizard Initialization
	mux.HandleFunc("POST /api/setup/init", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			PlatformName  string `json:"platformName"`
			Name          string `json:"name"`
			Username      string `json:"username"`
			Password      string `json:"password"`
			CFAccountID   string `json:"cfAccountId"`
			CFApiToken    string `json:"cfApiToken"`
			CFProjectName string `json:"cfProjectName"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)

		settingsMu.Lock()
		if body.PlatformName != "" {
			currentSettings.PlatformName = body.PlatformName
		}
		if body.Name != "" {
			currentSettings.UserName = body.Name
		}
		if body.Username != "" {
			currentSettings.Username = body.Username
		}
		settingsMu.Unlock()

		if body.CFAccountID != "" && body.CFApiToken != "" {
			deployURL, err := deployCloudflareRelayWorker(body.CFAccountID, body.CFApiToken, body.CFProjectName)
			if err == nil && deployURL != "" {
				newRelay := RelayEntry{
					ID:   "convx-relay",
					Name: "Cloudflare Worker Relay",
					URL:  deployURL,
				}
				ytClient.SetRelayURL(deployURL)
				audioProxy.SetRelayURL(deployURL)

				settingsMu.Lock()
				currentSettings.CFAccountID = body.CFAccountID
				currentSettings.CFApiToken = body.CFApiToken
				currentSettings.ActiveRelay = &newRelay
				currentSettings.Relays = []RelayEntry{newRelay}
				settingsMu.Unlock()
			}
		}

		saveDesktopSettings()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":       "ok",
			"platformName": currentSettings.PlatformName,
			"user": map[string]string{
				"name":     currentSettings.UserName,
				"username": currentSettings.Username,
			},
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

func deployCloudflareRelayWorker(accountId, apiToken, projectName string) (string, error) {
	accountId = strings.TrimSpace(accountId)
	apiToken = strings.TrimSpace(apiToken)
	if accountId == "" || apiToken == "" {
		return "", fmt.Errorf("Cloudflare Account ID dan API Token wajib diisi")
	}

	if projectName == "" {
		projectName = "convx-relay"
	}
	cleanProject := strings.ToLower(regexp.MustCompile(`[^a-zA-Z0-9_-]`).ReplaceAllString(projectName, "-"))

	workerCode := `export default {
  async fetch(request, env, ctx) {
    const target = request.headers.get("x-relay-target");
    const relayPath = request.headers.get("x-relay-path") || "/";

    if (!target) {
      return new Response(JSON.stringify({
        status: "ok",
        message: "Convx Cloudflare Relay Active",
        timestamp: Date.now()
      }), {
        status: 200,
        headers: { "content-type": "application/json" }
      });
    }

    const targetUrl = target.replace(/\/$/, "") + relayPath;
    const newHeaders = new Headers(request.headers);
    newHeaders.delete("x-relay-target");
    newHeaders.delete("x-relay-path");
    newHeaders.delete("host");

    const newRequestInit = {
      method: request.method,
      headers: newHeaders,
    };

    if (request.method !== "GET" && request.method !== "HEAD") {
      newRequestInit.body = request.body;
      newRequestInit.duplex = "half";
    }

    try {
      const response = await fetch(targetUrl, newRequestInit);
      return new Response(response.body, {
        status: response.status,
        headers: response.headers,
      });
    } catch (error) {
      return new Response(JSON.stringify({
        error: error.message,
        relay: "convx-worker"
      }), {
        status: 502,
        headers: { "content-type": "application/json" }
      });
    }
  }
};`

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	h1 := make(textproto.MIMEHeader)
	h1.Set("Content-Disposition", `form-data; name="index.js"; filename="index.js"`)
	h1.Set("Content-Type", "application/javascript+module")
	part1, err := writer.CreatePart(h1)
	if err != nil {
		return "", err
	}
	part1.Write([]byte(workerCode))

	h2 := make(textproto.MIMEHeader)
	h2.Set("Content-Disposition", `form-data; name="metadata"; filename="metadata.json"`)
	h2.Set("Content-Type", "application/json")
	part2, err := writer.CreatePart(h2)
	if err != nil {
		return "", err
	}
	part2.Write([]byte(`{"main_module":"index.js","compatibility_date":"2024-03-20","observability":{"enabled":true}}`))

	writer.Close()

	client := &http.Client{Timeout: 30 * time.Second}
	uploadURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/workers/scripts/%s", accountId, cleanProject)

	req, err := http.NewRequest("PUT", uploadURL, body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Cloudflare upload error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		var cfErr struct {
			Errors []struct {
				Message string `json:"message"`
			} `json:"errors"`
		}
		_ = json.Unmarshal(respBody, &cfErr)
		msg := fmt.Sprintf("HTTP %d", resp.StatusCode)
		if len(cfErr.Errors) > 0 {
			msg = cfErr.Errors[0].Message
		}
		return "", fmt.Errorf("Cloudflare error: %s", msg)
	}

	subdomainReq, _ := http.NewRequest("POST", uploadURL+"/subdomain", strings.NewReader(`{"enabled":true}`))
	subdomainReq.Header.Set("Authorization", "Bearer "+apiToken)
	subdomainReq.Header.Set("Content-Type", "application/json")
	subResp, err := client.Do(subdomainReq)
	if err == nil {
		subResp.Body.Close()
	}

	getSubReq, _ := http.NewRequest("GET", fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/workers/subdomain", accountId), nil)
	getSubReq.Header.Set("Authorization", "Bearer "+apiToken)
	subRes, err := client.Do(getSubReq)
	var deployURL string
	if err == nil && subRes.StatusCode == http.StatusOK {
		var subData struct {
			Result struct {
				Subdomain string `json:"subdomain"`
			} `json:"result"`
		}
		_ = json.NewDecoder(subRes.Body).Decode(&subData)
		subRes.Body.Close()
		if subData.Result.Subdomain != "" {
			deployURL = fmt.Sprintf("https://%s.%s.workers.dev", cleanProject, subData.Result.Subdomain)
		}
	}

	if deployURL == "" {
		return "", fmt.Errorf("Worker deployed tapi gagal mendapatkan subdomain workers.dev")
	}

	return deployURL, nil
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
