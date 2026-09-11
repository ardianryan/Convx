package proxy

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const chunkSize int64 = 10 * 1024 * 1024 // 10 MB chunk to cover full audio tracks without cutting off at 15s

type streamCacheEntry struct {
	streamURL string
	edgeURL   string
	expiresAt time.Time
}

type AudioProxy struct {
	httpClient *http.Client
	cache      sync.Map // videoId -> *streamCacheEntry
	relayURL   string
	mu         sync.RWMutex
}

func (p *AudioProxy) SetRelayURL(url string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.relayURL = strings.TrimSpace(url)
	if p.relayURL != "" {
		log.Printf("[AudioProxy] Cloudflare Relay enabled: %s", p.relayURL)
	} else {
		log.Printf("[AudioProxy] Cloudflare Relay disabled (direct mode)")
	}
}

func (p *AudioProxy) GetRelayURL() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.relayURL
}

func NewAudioProxy() *AudioProxy {
	proxy := &AudioProxy{
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("stopped after 10 redirects")
				}
				// Re-apply Range header across redirects
				if len(via) > 0 {
					if r := via[0].Header.Get("Range"); r != "" {
						req.Header.Set("Range", r)
					}
				}
				return nil
			},
		},
	}
	if envRelay := os.Getenv("CF_WORKER_URL"); envRelay != "" {
		proxy.SetRelayURL(envRelay)
	}
	return proxy
}

// ServeVideo handles streaming for a video ID, with automatic stream resolution,
// caching, Range header clamping (256 KB), and auto-retry on 403 (for IP rotation/mismatch)
func (p *AudioProxy) ServeVideo(w http.ResponseWriter, r *http.Request, videoID string, resolver func(string) (string, error)) {
	if videoID == "" {
		http.Error(w, "missing videoId", http.StatusBadRequest)
		return
	}

	targetURL, err := p.getOrResolveURL(videoID, resolver, false)
	if err != nil {
		log.Printf("[PROXY] Failed to resolve stream for %s: %v", videoID, err)
		http.Error(w, fmt.Sprintf("stream resolution failed: %v", err), http.StatusBadGateway)
		return
	}

	// First attempt: try with relay if configured, otherwise direct
	hasRelay := p.GetRelayURL() != ""
	statusCode, err := p.streamChunk(w, r, targetURL, hasRelay)
	if err != nil || statusCode == http.StatusForbidden || statusCode == http.StatusGone {
		// If relay failed with 403/410/error, IMMEDIATELY retry with DIRECT connection!
		if hasRelay {
			log.Printf("[PROXY] Relay failed (%d) for %s. Retrying directly...", statusCode, videoID)
			directStatus, directErr := p.streamChunk(w, r, targetURL, false)
			if directErr == nil && directStatus < 400 {
				return
			}
		}

		// Refresh stream URL and retry once transparently!
		log.Printf("[PROXY] Refreshing stream URL for %s and retrying...", videoID)
		freshURL, resolveErr := p.getOrResolveURL(videoID, resolver, true)
		if resolveErr == nil && freshURL != "" {
			// Try direct first on retry
			directStatus, directErr := p.streamChunk(w, r, freshURL, false)
			if directErr == nil && directStatus < 400 {
				return
			}
			if hasRelay {
				_, retryErr := p.streamChunk(w, r, freshURL, true)
				if retryErr == nil {
					return
				}
			}
			log.Printf("[PROXY] Retry failed for %s", videoID)
			http.Error(w, "stream proxy retry failed", http.StatusBadGateway)
			return
		}
		http.Error(w, fmt.Sprintf("stream proxy error: %v", err), http.StatusBadGateway)
	}
}

// ServeURL handles streaming for an explicit direct URL
func (p *AudioProxy) ServeURL(w http.ResponseWriter, r *http.Request, rawURL string) {
	if rawURL == "" {
		http.Error(w, "missing stream url", http.StatusBadRequest)
		return
	}
	hasRelay := p.GetRelayURL() != ""
	status, err := p.streamChunk(w, r, rawURL, hasRelay)
	if err != nil || status >= 400 {
		if hasRelay {
			_, err = p.streamChunk(w, r, rawURL, false)
		}
	}
	if err != nil {
		log.Printf("[PROXY] ServeURL error: %v", err)
	}
}

// ServeHTTP implements http.Handler for legacy /api/proxy/audio?url=... requests
func (p *AudioProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rawURL := r.URL.Query().Get("url")
	p.ServeURL(w, r, rawURL)
}

func (p *AudioProxy) getOrResolveURL(videoID string, resolver func(string) (string, error), forceRefresh bool) (string, error) {
	if !forceRefresh {
		if val, ok := p.cache.Load(videoID); ok {
			entry := val.(*streamCacheEntry)
			if time.Now().Before(entry.expiresAt) {
				if entry.edgeURL != "" {
					return entry.edgeURL, nil
				}
				return entry.streamURL, nil
			}
		}
	}

	rawURL, err := resolver(videoID)
	if err != nil {
		return "", err
	}

	p.cache.Store(videoID, &streamCacheEntry{
		streamURL: rawURL,
		expiresAt: time.Now().Add(4 * time.Hour),
	})
	return rawURL, nil
}

func (p *AudioProxy) streamChunk(w http.ResponseWriter, r *http.Request, streamURL string, useRelay bool) (int, error) {
	parsedURL, err := url.Parse(streamURL)
	if err != nil || (!strings.HasSuffix(parsedURL.Host, "googlevideo.com") && !strings.HasSuffix(parsedURL.Host, "youtube.com")) {
		return http.StatusForbidden, fmt.Errorf("invalid host: %s", parsedURL.Host)
	}

	method := r.Method
	if method != http.MethodGet && method != http.MethodHead {
		method = http.MethodGet
	}

	targetURL := streamURL
	relay := p.GetRelayURL()
	var isRelayed bool
	if useRelay && relay != "" {
		targetURL = relay
		isRelayed = true
	}

	outReq, err := http.NewRequestWithContext(r.Context(), method, targetURL, nil)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	if isRelayed {
		outReq.Header.Set("x-relay-target", fmt.Sprintf("https://%s", parsedURL.Host))
		relayPath := parsedURL.Path
		if parsedURL.RawQuery != "" {
			relayPath += "?" + parsedURL.RawQuery
		}
		outReq.Header.Set("x-relay-path", relayPath)
	}

	// Forward User-Agent or default to standard Chrome UA
	if ua := r.Header.Get("User-Agent"); ua != "" {
		outReq.Header.Set("User-Agent", ua)
	} else {
		outReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36")
	}

	// Clamping the Range header for Google Video CDN:
	// Google Video returns 403 Forbidden if:
	// 1) Range header is missing completely
	// 2) Range is open-ended without upper bound (e.g. bytes=0-)
	// 3) Range chunk size exceeds the allowed buffer
	rangeHeader := r.Header.Get("Range")
	var upstreamRange string
	if rangeHeader == "" || rangeHeader == "bytes=0-" {
		upstreamRange = fmt.Sprintf("bytes=0-%d", chunkSize-1)
	} else if strings.HasPrefix(rangeHeader, "bytes=") {
		rangeVal := strings.TrimPrefix(rangeHeader, "bytes=")
		parts := strings.Split(rangeVal, "-")
		if len(parts) >= 1 && parts[0] != "" {
			var start int64
			fmt.Sscanf(parts[0], "%d", &start)
			end := start + chunkSize - 1
			if len(parts) >= 2 && parts[1] != "" {
				var requestedEnd int64
				if _, err := fmt.Sscanf(parts[1], "%d", &requestedEnd); err == nil && requestedEnd >= start {
					if requestedEnd < end {
						end = requestedEnd
					}
				}
			}
			upstreamRange = fmt.Sprintf("bytes=%d-%d", start, end)
		} else {
			upstreamRange = fmt.Sprintf("bytes=0-%d", chunkSize-1)
		}
	} else {
		upstreamRange = fmt.Sprintf("bytes=0-%d", chunkSize-1)
	}
	outReq.Header.Set("Range", upstreamRange)

	resp, err := p.httpClient.Do(outReq)
	if err != nil {
		log.Printf("[PROXY-DEBUG] Do err: %v", err)
		return http.StatusBadGateway, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusGone {
		bodySnippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		log.Printf("[PROXY-DEBUG] Upstream %d for range=%s UA=%s URL=%s body=%s", 
			resp.StatusCode, upstreamRange, outReq.Header.Get("User-Agent"), streamURL[:min(len(streamURL), 100)], string(bodySnippet))
		return resp.StatusCode, fmt.Errorf("upstream error %d", resp.StatusCode)
	}

	// Forward necessary streaming headers
	for _, h := range []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges"} {
		if val := resp.Header.Get(h); val != "" {
			w.Header().Set(h, val)
		}
	}

	// Enable CORS on audio stream
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Range")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Range, Content-Length, Accept-Ranges")
	w.Header().Set("Cache-Control", "public, max-age=3600")

	w.WriteHeader(resp.StatusCode)
	if method != http.MethodHead {
		_, _ = io.Copy(w, resp.Body)
	}

	return resp.StatusCode, nil
}
