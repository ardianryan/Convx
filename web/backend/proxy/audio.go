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

	hasRelay := p.GetRelayURL() != ""

	targetURL, err := p.getOrResolveURL(videoID, resolver, false)
	if err != nil {
		log.Printf("[PROXY] Failed to resolve stream for %s: %v", videoID, err)
		http.Error(w, fmt.Sprintf("stream resolution failed: %v", err), http.StatusBadGateway)
		return
	}

	statusCode, err := p.streamChunk(w, r, targetURL, hasRelay)
	if err != nil || statusCode >= 400 {
		log.Printf("[PROXY] Stream status %d (err %v) for %s. Refreshing stream URL...", statusCode, err, videoID)
		freshURL, resolveErr := p.getOrResolveURL(videoID, resolver, true)
		if resolveErr == nil && freshURL != "" {
			directStatus, directErr := p.streamChunk(w, r, freshURL, hasRelay)
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
		outReq.Header.Set("x-relay-target", fmt.Sprintf("https://%s", parsedURL.Host))
		relayPath := parsedURL.Path
		if parsedURL.RawQuery != "" {
			relayPath += "?" + parsedURL.RawQuery
		}
		outReq.Header.Set("x-relay-path", relayPath)
	}

	outReq.Header.Set("User-Agent", "com.google.ios.youtube/20.08.3 (iPhone15,2; U; CPU iOS 18_0 like Mac OS X)")
	if rangeHeader := r.Header.Get("Range"); rangeHeader != "" {
		outReq.Header.Set("Range", rangeHeader)
	}

	resp, err := p.httpClient.Do(outReq)
	if err != nil {
		log.Printf("[PROXY] Do error (isRelayed=%t): %v", isRelayed, err)
		if isRelayed {
			// Retry direct once
			directReq, dErr := http.NewRequestWithContext(r.Context(), method, streamURL, nil)
			if dErr == nil {
				directReq.Header.Set("User-Agent", "com.google.ios.youtube/20.08.3 (iPhone15,2; U; CPU iOS 18_0 like Mac OS X)")
				if rangeHeader := r.Header.Get("Range"); rangeHeader != "" {
					directReq.Header.Set("Range", rangeHeader)
				}
				dResp, dDoErr := p.httpClient.Do(directReq)
				if dDoErr == nil {
					resp = dResp
					err = nil
				}
			}
		}
		if err != nil {
			return http.StatusBadGateway, err
		}
	}

	if resp.StatusCode >= 400 {
		bodySnippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		resp.Body.Close()
		log.Printf("[PROXY] Upstream status %d (isRelayed=%t): %s", resp.StatusCode, isRelayed, string(bodySnippet))

		if isRelayed {
			// Retry direct once
			directReq, dErr := http.NewRequestWithContext(r.Context(), method, streamURL, nil)
			if dErr == nil {
				directReq.Header.Set("User-Agent", "com.google.ios.youtube/20.08.3 (iPhone15,2; U; CPU iOS 18_0 like Mac OS X)")
				if rangeHeader := r.Header.Get("Range"); rangeHeader != "" {
					directReq.Header.Set("Range", rangeHeader)
				}
				dResp, dDoErr := p.httpClient.Do(directReq)
				if dDoErr == nil && dResp.StatusCode < 400 {
					resp = dResp
					goto processResponse
				}
				if dResp != nil {
					dResp.Body.Close()
				}
			}
		}
		return resp.StatusCode, fmt.Errorf("upstream error %d", resp.StatusCode)
	}

processResponse:
	defer resp.Body.Close()

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Range")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Range, Content-Length, Accept-Ranges")
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Cache-Control", "public, max-age=3600")

	if ct := resp.Header.Get("Content-Type"); ct != "" {
		w.Header().Set("Content-Type", ct)
	} else {
		w.Header().Set("Content-Type", "audio/mp4")
	}

	if cr := resp.Header.Get("Content-Range"); cr != "" {
		w.Header().Set("Content-Range", cr)
	}
	if cl := resp.Header.Get("Content-Length"); cl != "" {
		w.Header().Set("Content-Length", cl)
	}

	w.WriteHeader(resp.StatusCode)

	if method != http.MethodHead {
		buf := make([]byte, 32*1024)
		_, _ = io.CopyBuffer(w, resp.Body, buf)
	}

	return http.StatusOK, nil
}



