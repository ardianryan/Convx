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

const chunkSize int64 = 10 * 1024 * 1024 // 10 MB chunk

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
	cleaned := strings.TrimSpace(url)
	if cleaned != "" && !strings.HasPrefix(cleaned, "http://") && !strings.HasPrefix(cleaned, "https://") {
		cleaned = "https://" + cleaned
	}
	p.relayURL = cleaned
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

	statusCode, err := p.streamChunk(w, r, targetURL, false)
	if err != nil || statusCode >= 400 {
		log.Printf("[PROXY] Stream status %d (err %v) for %s. Refreshing stream URL...", statusCode, err, videoID)
		freshURL, resolveErr := p.getOrResolveURL(videoID, resolver, true)
		if resolveErr == nil && freshURL != "" {
			directStatus, directErr := p.streamChunk(w, r, freshURL, false)
			if directErr == nil && directStatus < 400 {
				return
			}
			log.Printf("[PROXY] Retry failed for %s (status %d, err %v)", videoID, directStatus, directErr)
			http.Error(w, "stream proxy retry failed", http.StatusBadGateway)
			return
		}
		http.Error(w, fmt.Sprintf("stream proxy error: %v", err), http.StatusBadGateway)
	}
}

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

	rangeHeader := r.Header.Get("Range")
	hasRange := rangeHeader != ""

	targetParsed, err := url.Parse(streamURL)
	if err != nil {
		targetParsed = parsedURL
	}

	relay := p.GetRelayURL()
	var isRelayed bool
	finalReqURL := streamURL
	if useRelay && relay != "" {
		finalReqURL = relay
		isRelayed = true
	}

	outReq, err := http.NewRequestWithContext(r.Context(), method, finalReqURL, nil)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	if isRelayed {
		outReq.Header.Set("x-relay-target", fmt.Sprintf("https://%s", targetParsed.Host))
		relayPath := targetParsed.Path
		if targetParsed.RawQuery != "" {
			relayPath += "?" + targetParsed.RawQuery
		}
		outReq.Header.Set("x-relay-path", relayPath)
	}

	if ua := r.Header.Get("User-Agent"); ua != "" {
		outReq.Header.Set("User-Agent", ua)
	} else {
		outReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36")
	}

	if hasRange {
		outReq.Header.Set("Range", rangeHeader)
	}

	resp, err := p.httpClient.Do(outReq)
	if err != nil {
		log.Printf("[PROXY-DEBUG] Do err: %v", err)
		return http.StatusBadGateway, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusGone {
		bodySnippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		log.Printf("[PROXY-DEBUG] Upstream %d for Range=%s UA=%s URL=%s body=%s",
			resp.StatusCode, rangeHeader, outReq.Header.Get("User-Agent"), streamURL[:min(len(streamURL), 100)], string(bodySnippet))
		return resp.StatusCode, fmt.Errorf("upstream error %d", resp.StatusCode)
	}

	// Forward streaming headers
	for _, h := range []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges"} {
		if val := resp.Header.Get(h); val != "" {
			w.Header().Set(h, val)
		}
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Range")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Range, Content-Length, Accept-Ranges")
	w.Header().Set("Cache-Control", "public, max-age=3600")

	finalStatus := resp.StatusCode
	if !hasRange && resp.StatusCode == http.StatusPartialContent {
		finalStatus = http.StatusOK
		w.Header().Del("Content-Range")
		if cr := resp.Header.Get("Content-Range"); cr != "" {
			if parts := strings.Split(cr, "/"); len(parts) == 2 && parts[1] != "*" {
				w.Header().Set("Content-Length", parts[1])
			}
		}
	}

	w.WriteHeader(finalStatus)
	if method != http.MethodHead {
		_, _ = io.Copy(w, resp.Body)
	}

	return finalStatus, nil
}

