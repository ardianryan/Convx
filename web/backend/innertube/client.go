package innertube

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	youtubeMusicBase = "https://music.youtube.com/youtubei/v1"
	youtubeBase      = "https://www.youtube.com/youtubei/v1"
	userAgentWeb     = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36"
	cookieFilePath   = "data/cookie.txt"
)

type Client struct {
	httpClient *http.Client
	mu         sync.RWMutex
	cookie     string
	cookieMap  map[string]string
	relayURL   string
}

func NewClient() *Client {
	c := &Client{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		cookieMap: make(map[string]string),
	}

	if envRelay := os.Getenv("CF_WORKER_URL"); envRelay != "" {
		c.SetRelayURL(envRelay)
	}

	// Load saved cookie if exists
	if data, err := os.ReadFile(cookieFilePath); err == nil {
		saved := strings.TrimSpace(string(data))
		if saved != "" {
			c.SetCookie(saved, false)
			log.Printf("[InnerTube] Loaded saved YouTube cookie from %s", cookieFilePath)
		}
	}

	return c
}

func (c *Client) SetCookie(cookie string, persist bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cookie = cookie
	c.cookieMap = parseCookieString(cookie)

	if persist {
		if cookie == "" {
			_ = os.Remove(cookieFilePath)
		} else {
			_ = os.MkdirAll(filepath.Dir(cookieFilePath), 0755)
			if err := os.WriteFile(cookieFilePath, []byte(cookie), 0600); err != nil {
				log.Printf("[InnerTube] Failed to save cookie to %s: %v", cookieFilePath, err)
			}
		}
	}
}

func (c *Client) GetCookie() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cookie
}

func (c *Client) HasCookie() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cookie != ""
}

func (c *Client) ClearCookie() {
	c.SetCookie("", true)
}

func (c *Client) SetRelayURL(url string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.relayURL = strings.TrimSpace(url)
	if c.relayURL != "" {
		log.Printf("[InnerTube] Cloudflare Relay enabled: %s", c.relayURL)
	} else {
		log.Printf("[InnerTube] Cloudflare Relay disabled (direct mode)")
	}
}

func (c *Client) GetRelayURL() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.relayURL
}

func (c *Client) doRequestWithFallback(method, targetBase, targetPath string, bodyBytes []byte, userAgent string, referer string) (*http.Response, error) {
	c.mu.RLock()
	relay := c.relayURL
	c.mu.RUnlock()

	if relay != "" {
		req, err := http.NewRequest(method, relay, bytes.NewReader(bodyBytes))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
			if userAgent != "" {
				req.Header.Set("User-Agent", userAgent)
			}
			if referer != "" {
				req.Header.Set("Referer", referer)
			}
			req.Header.Set("x-relay-target", targetBase)
			req.Header.Set("x-relay-path", targetPath)
			c.applyAuthHeaders(req)

			resp, err := c.httpClient.Do(req)
			if err == nil && resp.StatusCode == http.StatusOK {
				return resp, nil
			}
			if resp != nil {
				resp.Body.Close()
			}
			statusStr := "err"
			if resp != nil {
				statusStr = strconv.Itoa(resp.StatusCode)
			}
			log.Printf("[InnerTube] Relay request to %s%s failed (status %s, err %v). Retrying directly...", targetBase, targetPath, statusStr, err)
		}
	}

	// Direct request fallback (bypass relay)
	req, err := http.NewRequest(method, targetBase+targetPath, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	if referer != "" {
		req.Header.Set("Referer", referer)
	}
	c.applyAuthHeaders(req)

	return c.httpClient.Do(req)
}

func parseCookieString(raw string) map[string]string {
	cookies := make(map[string]string)
	for _, part := range strings.Split(raw, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if k, v, ok := strings.Cut(part, "="); ok {
			cookies[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	return cookies
}

func (c *Client) applyAuthHeaders(req *http.Request) {
	c.mu.RLock()
	cookie := c.cookie
	cookieMap := c.cookieMap
	c.mu.RUnlock()

	if cookie == "" {
		return
	}

	origin := "https://www.youtube.com"
	if req.URL != nil && req.URL.Host == "music.youtube.com" {
		origin = "https://music.youtube.com"
	}

	req.Header.Set("Cookie", cookie)
	req.Header.Set("X-Origin", origin)
	req.Header.Set("Referer", origin+"/")

	sapisid := cookieMap["SAPISID"]
	if sapisid == "" {
		sapisid = cookieMap["__Secure-3PAPISID"]
	}
	if sapisid != "" {
		now := time.Now().Unix()
		hashInput := fmt.Sprintf("%d %s %s", now, sapisid, origin)
		h := sha1.Sum([]byte(hashInput))
		sapisidHash := hex.EncodeToString(h[:])
		req.Header.Set("Authorization", fmt.Sprintf("SAPISIDHASH %d_%s", now, sapisidHash))
	}
}

type ClientConfig struct {
	Name       string
	Version    string
	UserAgent  string
	DeviceMake string
	DeviceModel string
	OsName     string
	OsVersion  string
	SdkVer     int
}

var clientHierarchy = []ClientConfig{
	{
		Name:        "IOS",
		Version:     "20.08.3",
		UserAgent:   "com.google.ios.youtube/20.08.3 (iPhone15,2; U; CPU iOS 18_0 like Mac OS X)",
		DeviceMake:  "Apple",
		DeviceModel: "iPhone15,2",
		OsName:      "iOS",
		OsVersion:   "18.0",
	},
	{
		Name:        "IOS",
		Version:     "20.05.1",
		UserAgent:   "com.google.ios.youtube/20.05.1 (iPhone15,2; U; CPU iOS 18_0 like Mac OS X)",
		DeviceMake:  "Apple",
		DeviceModel: "iPhone15,2",
		OsName:      "iOS",
		OsVersion:   "18.0",
	},
	{
		Name:        "IOS",
		Version:     "20.01.2",
		UserAgent:   "com.google.ios.youtube/20.01.2 (iPhone14,5; U; CPU iOS 17_5 like Mac OS X)",
		DeviceMake:  "Apple",
		DeviceModel: "iPhone14,5",
		OsName:      "iOS",
		OsVersion:   "17.5",
	},
}

// GetStream resolves the audio stream for a given YouTube video ID
// If user has a cookie set, it uses IOS with authentication first.
func (c *Client) GetStream(videoID string) (*StreamInfo, error) {
	var lastErr error

	var hierarchy []ClientConfig
	if c.HasCookie() {
		hierarchy = append([]ClientConfig{
			{
				Name:        "IOS",
				Version:     "20.08.3",
				UserAgent:   "com.google.ios.youtube/20.08.3 (iPhone15,2; U; CPU iOS 18_0 like Mac OS X)",
				DeviceMake:  "Apple",
				DeviceModel: "iPhone15,2",
				OsName:      "iOS",
				OsVersion:   "18.0",
			},
		}, clientHierarchy...)
	} else {
		hierarchy = clientHierarchy
	}

	for _, cfg := range hierarchy {
		stream, err := c.requestStreamWithClient(videoID, cfg)
		if err == nil && stream != nil && stream.StreamURL != "" {
			return stream, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("all streaming clients failed for %s: %w", videoID, lastErr)
}

func (c *Client) requestStreamWithClient(videoID string, cfg ClientConfig) (*StreamInfo, error) {
	clientMap := map[string]interface{}{
		"clientName":    cfg.Name,
		"clientVersion": cfg.Version,
		"hl":            "en",
		"gl":            "US",
	}
	if cfg.DeviceModel != "" {
		clientMap["deviceModel"] = cfg.DeviceModel
	}
	if cfg.DeviceMake != "" {
		clientMap["deviceMake"] = cfg.DeviceMake
	}
	if cfg.OsName != "" {
		clientMap["osName"] = cfg.OsName
	}
	if cfg.OsVersion != "" {
		clientMap["osVersion"] = cfg.OsVersion
	}
	if cfg.SdkVer > 0 {
		clientMap["androidSdkVersion"] = cfg.SdkVer
	}

	reqBody := map[string]interface{}{
		"context": map[string]interface{}{
			"client": clientMap,
		},
		"videoId": videoID,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	targetBase := youtubeBase
	if cfg.Name == "WEB_REMIX" {
		targetBase = youtubeMusicBase
	}

	resp, err := c.doRequestWithFallback("POST", targetBase, "/player", bodyBytes, cfg.UserAgent, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	var playerResp PlayerResponse
	if err := json.NewDecoder(resp.Body).Decode(&playerResp); err != nil {
		return nil, err
	}

	if playerResp.PlayabilityStatus.Status != "OK" {
		return nil, fmt.Errorf("playability: %s (%s)", playerResp.PlayabilityStatus.Status, playerResp.PlayabilityStatus.Reason)
	}

	var audioFormats []Format
	for _, f := range playerResp.StreamingData.AdaptiveFormats {
		if strings.HasPrefix(f.MimeType, "audio/") && f.URL != "" {
			audioFormats = append(audioFormats, f)
		}
	}
	if len(audioFormats) == 0 {
		for _, f := range playerResp.StreamingData.Formats {
			if strings.HasPrefix(f.MimeType, "audio/") && f.URL != "" {
				audioFormats = append(audioFormats, f)
			}
		}
	}
	if len(audioFormats) == 0 {
		return nil, fmt.Errorf("no direct audio streams")
	}

	var bestFormat Format
	bestScore := -1

	for _, f := range audioFormats {
		score := f.Bitrate
		if strings.HasPrefix(f.MimeType, "audio/webm") {
			score += 15000 // preference bonus for Opus
		}
		if score > bestScore {
			bestScore = score
			bestFormat = f
		}
	}

	expiresIn := 21600
	if s := playerResp.StreamingData.ExpiresInSeconds; s != "" {
		if val, err := strconv.Atoi(s); err == nil && val > 0 {
			expiresIn = val
		}
	}

	return &StreamInfo{
		VideoID:   videoID,
		Title:     playerResp.VideoDetails.Title,
		Artist:    playerResp.VideoDetails.Author,
		StreamURL: bestFormat.URL,
		MimeType:  bestFormat.MimeType,
		Bitrate:   bestFormat.Bitrate,
		ExpiresIn: expiresIn,
	}, nil
}

// Search queries YouTube for embeddable music videos and tracks
func (c *Client) Search(query string) ([]Song, error) {
	// Search both YouTube Music (official topic releases) and YouTube Web
	ytmSongs, _ := c.searchYouTubeMusic(query)
	webSongs, _ := c.searchYouTubeWeb(query)

	var topics []Song
	var others []Song
	var longVideos []Song
	seenIDs := make(map[string]bool)

	filterSong := func(s Song) {
		if seenIDs[s.ID] {
			return
		}
		seenIDs[s.ID] = true
		// Filter out 1h+ bootleg compilations that often have copyright strikes
		if s.Duration > 900 {
			longVideos = append(longVideos, s)
		} else if s.IsTopic {
			topics = append(topics, s)
		} else {
			others = append(others, s)
		}
	}

	for _, s := range ytmSongs {
		filterSong(s)
	}

	for _, s := range webSongs {
		filterSong(s)
	}

	all := append(topics, others...)
	if len(all) < 10 {
		all = append(all, longVideos...)
	}
	if len(all) > 0 {
		return all, nil
	}

	return nil, fmt.Errorf("no results found")
}

func (c *Client) searchYouTubeWeb(query string) ([]Song, error) {
	reqBody := map[string]interface{}{
		"context": map[string]interface{}{
			"client": map[string]interface{}{
				"clientName":    "WEB",
				"clientVersion": "2.20240820.01.00",
				"hl":            "id",
				"gl":            "ID",
			},
		},
		"query": query,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequestWithFallback("POST", youtubeBase, "/search", bodyBytes, userAgentWeb, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var rawMap map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawMap); err != nil {
		return nil, err
	}

	var results []Song
	seenIDs := make(map[string]bool)

	var walk func(val interface{})
	walk = func(val interface{}) {
		switch v := val.(type) {
		case map[string]interface{}:
			if vr, ok := v["videoRenderer"].(map[string]interface{}); ok {
				if s := parseVideoRenderer(vr); s != nil && !seenIDs[s.ID] {
					seenIDs[s.ID] = true
					results = append(results, *s)
				}
			}
			for _, child := range v {
				walk(child)
			}
		case []interface{}:
			for _, elem := range v {
				walk(elem)
			}
		}
	}
	walk(rawMap)

	return results, nil
}

func parseVideoRenderer(vr map[string]interface{}) *Song {
	videoID, _ := vr["videoId"].(string)
	if videoID == "" {
		return nil
	}

	title := ""
	if titleObj, ok := vr["title"].(map[string]interface{}); ok {
		if runs, ok := titleObj["runs"].([]interface{}); ok && len(runs) > 0 {
			if r0, ok := runs[0].(map[string]interface{}); ok {
				title, _ = r0["text"].(string)
			}
		}
	}
	if title == "" {
		return nil
	}

	artist := ""
	if ownerObj, ok := vr["ownerText"].(map[string]interface{}); ok {
		if runs, ok := ownerObj["runs"].([]interface{}); ok && len(runs) > 0 {
			if r0, ok := runs[0].(map[string]interface{}); ok {
				artist, _ = r0["text"].(string)
			}
		}
	}
	if artist == "" {
		if shortObj, ok := vr["shortBylineText"].(map[string]interface{}); ok {
			if runs, ok := shortObj["runs"].([]interface{}); ok && len(runs) > 0 {
				if r0, ok := runs[0].(map[string]interface{}); ok {
					artist, _ = r0["text"].(string)
				}
			}
		}
	}

	durationText := ""
	durationSec := 0
	if lenObj, ok := vr["lengthText"].(map[string]interface{}); ok {
		durationText, _ = lenObj["simpleText"].(string)
		durationSec = parseDuration(durationText)
	}
	if durationSec == 0 {
		if overlays, ok := vr["thumbnailOverlays"].([]interface{}); ok {
			for _, o := range overlays {
				if oMap, ok := o.(map[string]interface{}); ok {
					if tsr, ok := oMap["thumbnailOverlayTimeStatusRenderer"].(map[string]interface{}); ok {
						if txtObj, ok := tsr["text"].(map[string]interface{}); ok {
							if st, ok := txtObj["simpleText"].(string); ok {
								durationText = st
								durationSec = parseDuration(st)
								break
							}
						}
					}
				}
			}
		}
	}

	thumbnail := ""
	if thumbObj, ok := vr["thumbnail"].(map[string]interface{}); ok {
		if thumbs, ok := thumbObj["thumbnails"].([]interface{}); ok && len(thumbs) > 0 {
			last := thumbs[len(thumbs)-1].(map[string]interface{})
			thumbnail, _ = last["url"].(string)
		}
	}
	if thumbnail == "" {
		thumbnail = fmt.Sprintf("https://i.ytimg.com/vi/%s/hqdefault.jpg", videoID)
	}

	isTopic := false
	album := "YouTube Video"
	cleanArtist := artist
	if strings.HasSuffix(artist, " - Topic") {
		isTopic = true
		cleanArtist = strings.TrimSuffix(artist, " - Topic")
		album = "Official Audio • Topic"
	} else if strings.Contains(strings.ToLower(artist), "topic") {
		isTopic = true
		album = "Official Audio • Topic"
	} else if strings.Contains(strings.ToLower(title), "official audio") {
		isTopic = true
		album = "Official Audio"
	}

	return &Song{
		ID:           videoID,
		Title:        title,
		Artist:       cleanArtist,
		Album:        album,
		Duration:     durationSec,
		DurationText: durationText,
		Thumbnail:    thumbnail,
		IsTopic:      isTopic,
	}
}

func (c *Client) searchYouTubeMusic(query string) ([]Song, error) {
	reqBody := map[string]interface{}{
		"context": map[string]interface{}{
			"client": map[string]interface{}{
				"clientName":    "WEB_REMIX",
				"clientVersion": "1.20240820.01.00",
				"hl":            "en",
				"gl":            "US",
			},
		},
		"query": query,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	resp, err := c.doRequestWithFallback("POST", youtubeMusicBase, "/search", bodyBytes, userAgentWeb, "https://music.youtube.com/")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var rawMap map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawMap); err != nil {
		return nil, err
	}

	return extractSongsFromSearchResponse(rawMap, query), nil
}

func extractSongsFromSearchResponse(data map[string]interface{}, query string) []Song {
	var results []Song
	seenIDs := make(map[string]bool)

	var walk func(val interface{})
	walk = func(val interface{}) {
		switch v := val.(type) {
		case map[string]interface{}:
			if itemRenderer, ok := v["musicResponsiveListItemRenderer"].(map[string]interface{}); ok {
				if song := parseMusicResponsiveListItem(itemRenderer, query); song != nil && !seenIDs[song.ID] {
					seenIDs[song.ID] = true
					results = append(results, *song)
				}
			}
			for _, child := range v {
				walk(child)
			}
		case []interface{}:
			for _, elem := range v {
				walk(elem)
			}
		}
	}

	walk(data)
	return results
}

func parseMusicResponsiveListItem(item map[string]interface{}, query string) *Song {
	var videoID string

	if navEndpoint, ok := item["navigationEndpoint"].(map[string]interface{}); ok {
		if watchEndpoint, ok := navEndpoint["watchEndpoint"].(map[string]interface{}); ok {
			if id, ok := watchEndpoint["videoId"].(string); ok {
				videoID = id
			}
		}
	}
	if videoID == "" {
		if plData, ok := item["playlistItemData"].(map[string]interface{}); ok {
			if id, ok := plData["videoId"].(string); ok {
				videoID = id
			}
		}
	}
	if videoID == "" {
		return nil
	}

	flexCols, ok := item["flexColumns"].([]interface{})
	if !ok || len(flexCols) == 0 {
		return nil
	}

	title := extractTextFromFlexColumn(flexCols[0])
	if title == "" {
		return nil
	}

	artist := ""
	album := ""
	durationText := ""
	durationSec := 0

	for _, col := range flexCols[1:] {
		colMap, ok := col.(map[string]interface{})
		if !ok {
			continue
		}
		flexCol, ok := colMap["musicResponsiveListItemFlexColumnRenderer"].(map[string]interface{})
		if !ok {
			continue
		}
		textObj, ok := flexCol["text"].(map[string]interface{})
		if !ok {
			continue
		}
		runs, ok := textObj["runs"].([]interface{})
		if !ok {
			continue
		}

		for _, r := range runs {
			rMap, ok := r.(map[string]interface{})
			if !ok {
				continue
			}
			txt := strings.TrimSpace(rMap["text"].(string))
			if txt == "" || txt == "•" || strings.EqualFold(txt, "Song") || strings.EqualFold(txt, "Video") || strings.HasSuffix(strings.ToLower(txt), "plays") {
				continue
			}

			// Check if this run is artist
			isArtist := false
			if nav, ok := rMap["navigationEndpoint"].(map[string]interface{}); ok {
				if browse, ok := nav["browseEndpoint"].(map[string]interface{}); ok {
					if configs, ok := browse["browseEndpointContextSupportedConfigs"].(map[string]interface{}); ok {
						if musicConfig, ok := configs["browseEndpointContextMusicConfig"].(map[string]interface{}); ok {
							if pageType, ok := musicConfig["pageType"].(string); ok && pageType == "MUSIC_PAGE_TYPE_ARTIST" {
								isArtist = true
							}
						}
					}
				}
			}

			if isArtist && artist == "" {
				artist = txt
			} else if durationSec == 0 && (strings.Contains(txt, ":") || strings.Contains(txt, ".")) && len(txt) <= 12 {
				d := parseDuration(txt)
				if d > 0 {
					durationText = txt
					durationSec = d
					continue
				}
			} else if artist == "" && !strings.Contains(txt, ":") {
				artist = txt
			} else if album == "" && !strings.Contains(txt, ":") && txt != artist {
				album = txt
			}
		}
	}

	if artist == "" {
		artist = query
	}

	thumbnail := ""
	if thumbRenderer, ok := item["thumbnail"].(map[string]interface{}); ok {
		if musicThumb, ok := thumbRenderer["musicThumbnailRenderer"].(map[string]interface{}); ok {
			if thumbObj, ok := musicThumb["thumbnail"].(map[string]interface{}); ok {
				if thumbs, ok := thumbObj["thumbnails"].([]interface{}); ok && len(thumbs) > 0 {
					last := thumbs[len(thumbs)-1].(map[string]interface{})
					if u, ok := last["url"].(string); ok {
						thumbnail = u
					}
				}
			}
		}
	}
	if thumbnail == "" {
		thumbnail = fmt.Sprintf("https://i.ytimg.com/vi/%s/hqdefault.jpg", videoID)
	}

	isTopic := false
	cleanArtist := artist
	if strings.HasSuffix(artist, " - Topic") {
		isTopic = true
		cleanArtist = strings.TrimSuffix(artist, " - Topic")
	} else if strings.Contains(strings.ToLower(artist), "topic") {
		isTopic = true
	} else if strings.Contains(strings.ToLower(album), "topic") {
		isTopic = true
	}

	return &Song{
		ID:           videoID,
		Title:        title,
		Artist:       cleanArtist,
		Album:        album,
		Duration:     durationSec,
		DurationText: durationText,
		Thumbnail:    thumbnail,
		IsTopic:      isTopic,
	}
}

func extractTextFromFlexColumn(col interface{}) string {
	colMap, ok := col.(map[string]interface{})
	if !ok {
		return ""
	}
	flexCol, ok := colMap["musicResponsiveListItemFlexColumnRenderer"].(map[string]interface{})
	if !ok {
		return ""
	}
	textObj, ok := flexCol["text"].(map[string]interface{})
	if !ok {
		return ""
	}
	runs, ok := textObj["runs"].([]interface{})
	if !ok || len(runs) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, r := range runs {
		if rMap, ok := r.(map[string]interface{}); ok {
			if t, ok := rMap["text"].(string); ok {
				sb.WriteString(t)
			}
		}
	}
	return sb.String()
}

func extractRunsFromFlexColumn(col interface{}) []string {
	var result []string
	colMap, ok := col.(map[string]interface{})
	if !ok {
		return result
	}
	flexCol, ok := colMap["musicResponsiveListItemFlexColumnRenderer"].(map[string]interface{})
	if !ok {
		return result
	}
	textObj, ok := flexCol["text"].(map[string]interface{})
	if !ok {
		return result
	}
	runs, ok := textObj["runs"].([]interface{})
	if !ok {
		return result
	}
	for _, r := range runs {
		if rMap, ok := r.(map[string]interface{}); ok {
			if t, ok := rMap["text"].(string); ok {
				result = append(result, t)
			}
		}
	}
	return result
}

func parseDuration(timeStr string) int {
	clean := strings.TrimSpace(timeStr)
	for _, c := range clean {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			return 0
		}
	}
	norm := strings.ReplaceAll(clean, ".", ":")
	parts := strings.Split(norm, ":")
	if len(parts) == 2 {
		m, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
		s, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err1 == nil && err2 == nil {
			return m*60 + s
		}
	} else if len(parts) == 3 {
		h, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
		m, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
		s, err3 := strconv.Atoi(strings.TrimSpace(parts[2]))
		if err1 == nil && err2 == nil && err3 == nil {
			return h*3600 + m*60 + s
		}
	}
	return 0
}
