package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
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

//go:embed all:dist
var distFS embed.FS

type WebSettings struct {
	IsInitialized bool        `json:"isInitialized"`
	PlatformName  string      `json:"platformName"`
	UserName      string      `json:"name"`
	Username      string      `json:"username"`
	CFAccountID   string      `json:"cfAccountId"`
	CFApiToken    string      `json:"cfApiToken"`
	ActiveRelay   *RelayEntry `json:"activeRelay"`
	Relays        []RelayEntry`json:"relays"`
}

type RelayEntry struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	URL      string `json:"url"`
	IsActive bool   `json:"isActive"`
}

var (
	ytClient         *innertube.Client
	audioProxy       *proxy.AudioProxy
	settingsMu       sync.RWMutex
	currentSettings  WebSettings
	settingsFilePath = filepath.Join("data", "web_settings.json")
)

func loadWebSettings() {
	settingsMu.Lock()
	defer settingsMu.Unlock()

	currentSettings = WebSettings{
		IsInitialized: true,
		PlatformName:  "Convx Music",
		UserName:      "Ryan Ardian",
		Username:      "admin",
		Relays:        []RelayEntry{},
	}

	_ = os.MkdirAll("data", 0755)
	data, err := os.ReadFile(settingsFilePath)
	if err == nil {
		_ = json.Unmarshal(data, &currentSettings)
	}

	if currentSettings.ActiveRelay != nil && currentSettings.ActiveRelay.URL != "" {
		currentSettings.ActiveRelay.IsActive = true
	}
}

func saveWebSettings() {
	settingsMu.Lock()
	defer settingsMu.Unlock()

	_ = os.MkdirAll("data", 0755)
	data, err := json.MarshalIndent(currentSettings, "", "  ")
	if err == nil {
		_ = os.WriteFile(settingsFilePath, data, 0644)
	}
}

func main() {
	ytClient = innertube.NewClient()
	audioProxy = proxy.NewAudioProxy()

	loadWebSettings()

	// Restore active relay if previously saved
	settingsMu.RLock()
	if currentSettings.ActiveRelay != nil && currentSettings.ActiveRelay.URL != "" {
		currentSettings.ActiveRelay.IsActive = true
		ytClient.SetRelayURL(currentSettings.ActiveRelay.URL)
		audioProxy.SetRelayURL(currentSettings.ActiveRelay.URL)
		log.Printf("[Convx Server] Restored active relay: %s", currentSettings.ActiveRelay.URL)
	}
	settingsMu.RUnlock()

	port := os.Getenv("PORT")
	if port == "" {
		port = "7554"
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

	// Settings & Relay Deploy Endpoints
	mux.HandleFunc("GET /api/settings", handleGetSettings)
	mux.HandleFunc("POST /api/settings", handlePostSettings)
	mux.HandleFunc("POST /api/relays/deploy", handleDeployRelay)
	mux.HandleFunc("POST /api/relays/test", handleTestRelay)
	mux.HandleFunc("POST /api/relays/toggle", handleToggleRelay)
	mux.HandleFunc("DELETE /api/relays", handleDeleteRelay)

	// Auth & Onboarding Setup Endpoints
	mux.HandleFunc("POST /api/setup/init", handleSetupInit)
	mux.HandleFunc("GET /api/auth/me", handleAuthMe)
	mux.HandleFunc("POST /api/auth/login", handleAuthLogin)
	mux.HandleFunc("POST /api/auth/logout", handleAuthLogout)
	mux.HandleFunc("POST /api/devices/heartbeat", handleDeviceHeartbeat)

	// Static Files (Svelte Frontend SPA Fallback)
	staticFS, err := fs.Sub(distFS, "dist")
	if err != nil {
		log.Printf("Static FS fallback: %v", err)
	}

	var fileServer http.Handler
	if staticFS != nil {
		fileServer = http.FileServer(http.FS(staticFS))
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
			return
		}

		if fileServer != nil {
			path := strings.TrimPrefix(r.URL.Path, "/")
			if path != "" && !strings.Contains(path, ".") {
				r.URL.Path = "/"
			}
			fileServer.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
	})

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
	req.Header.Set("User-Agent", "Convx-Player/1.0 (https://github.com/ardianryan/convx)")

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
		req2.Header.Set("User-Agent", "Convx-Player/1.0 (https://github.com/ardianryan/convx)")
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

func handleGetSettings(w http.ResponseWriter, r *http.Request) {
	settingsMu.RLock()
	defer settingsMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"mode":         "web",
		"platformName": currentSettings.PlatformName,
		"name":         currentSettings.UserName,
		"username":     currentSettings.Username,
		"cfAccountId":  currentSettings.CFAccountID,
		"activeRelay":  currentSettings.ActiveRelay,
		"relays":       currentSettings.Relays,
	})
}

func handlePostSettings(w http.ResponseWriter, r *http.Request) {
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
	saveWebSettings()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
}

func handleDeployRelay(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CFAccountID string `json:"cfAccountId"`
		CFApiToken  string `json:"cfApiToken"`
		ProjectName string `json:"projectName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid json body"}`, http.StatusBadRequest)
		return
	}

	deployURL, err := deployCloudflareRelayWorkerWeb(body.CFAccountID, body.CFApiToken, body.ProjectName)
	if err != nil {
		log.Printf("[Web Server] Relay deploy failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	newRelay := RelayEntry{
		ID:       "convx-relay",
		Name:     "Cloudflare Worker Relay",
		URL:      deployURL,
		IsActive: true,
	}

	ytClient.SetRelayURL(deployURL)
	audioProxy.SetRelayURL(deployURL)

	settingsMu.Lock()
	currentSettings.CFAccountID = body.CFAccountID
	currentSettings.CFApiToken = body.CFApiToken
	currentSettings.ActiveRelay = &newRelay
	currentSettings.Relays = []RelayEntry{newRelay}
	settingsMu.Unlock()
	saveWebSettings()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"relay":  newRelay,
	})
}

func handleTestRelay(w http.ResponseWriter, r *http.Request) {
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
}

func handleToggleRelay(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID       string `json:"id"`
		IsActive bool   `json:"isActive"`
		Active   bool   `json:"active"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	shouldBeActive := body.IsActive || body.Active

	settingsMu.Lock()
	if len(currentSettings.Relays) > 0 {
		currentSettings.Relays[0].IsActive = shouldBeActive
		if shouldBeActive {
			currentSettings.ActiveRelay = &currentSettings.Relays[0]
			currentSettings.ActiveRelay.IsActive = true
			ytClient.SetRelayURL(currentSettings.ActiveRelay.URL)
			audioProxy.SetRelayURL(currentSettings.ActiveRelay.URL)
		} else {
			currentSettings.ActiveRelay = &currentSettings.Relays[0]
			currentSettings.ActiveRelay.IsActive = false
			ytClient.SetRelayURL("")
			audioProxy.SetRelayURL("")
		}
	} else if currentSettings.ActiveRelay != nil {
		currentSettings.ActiveRelay.IsActive = shouldBeActive
		if shouldBeActive {
			ytClient.SetRelayURL(currentSettings.ActiveRelay.URL)
			audioProxy.SetRelayURL(currentSettings.ActiveRelay.URL)
		} else {
			ytClient.SetRelayURL("")
			audioProxy.SetRelayURL("")
		}
	}
	activeRelay := currentSettings.ActiveRelay
	settingsMu.Unlock()
	saveWebSettings()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "ok",
		"activeRelay": activeRelay,
	})
}

func handleDeleteRelay(w http.ResponseWriter, r *http.Request) {
	settingsMu.Lock()
	currentSettings.ActiveRelay = nil
	currentSettings.Relays = []RelayEntry{}
	ytClient.SetRelayURL("")
	audioProxy.SetRelayURL("")
	settingsMu.Unlock()
	saveWebSettings()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
}

func handleSetupInit(w http.ResponseWriter, r *http.Request) {
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
	currentSettings.IsInitialized = true
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
		deployURL, err := deployCloudflareRelayWorkerWeb(body.CFAccountID, body.CFApiToken, body.CFProjectName)
		if err == nil && deployURL != "" {
			newRelay := RelayEntry{
				ID:       "convx-relay",
				Name:     "Cloudflare Worker Relay",
				URL:      deployURL,
				IsActive: true,
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

	saveWebSettings()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "ok",
		"isInitialized": true,
		"isLoggedIn":    true,
		"platformName":  currentSettings.PlatformName,
		"user": map[string]string{
			"name":     currentSettings.UserName,
			"username": currentSettings.Username,
		},
	})
}

func handleAuthMe(w http.ResponseWriter, r *http.Request) {
	settingsMu.RLock()
	defer settingsMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "ok",
		"isInitialized": true,
		"isLoggedIn":    true,
		"platformName":  currentSettings.PlatformName,
		"user": map[string]string{
			"name":     currentSettings.UserName,
			"username": currentSettings.Username,
		},
		"activeRelay": currentSettings.ActiveRelay,
	})
}

func handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"user":   map[string]string{"username": "admin", "role": "admin"},
	})
}

func handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
}

func handleDeviceHeartbeat(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
}

func deployCloudflareRelayWorkerWeb(accountId, apiToken, projectName string) (string, error) {
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
