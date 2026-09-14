package proxy

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
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

var clusterID = []byte{0x1f, 0x43, 0xb6, 0x75}  // WebM Cluster ID
var timecodeID = []byte{0xe7}                   // WebM Timecode ID

func adjustWebMClusterTimecodesFixed(data []byte, offsetMs uint64) []byte {
	var out bytes.Buffer

	idx := 0
	for {
		cPos := bytes.Index(data[idx:], clusterID)
		if cPos == -1 {
			out.Write(data[idx:])
			break
		}
		absPos := idx + cPos
		out.Write(data[idx:absPos])

		searchLimit := absPos + 32
		if searchLimit > len(data) {
			searchLimit = len(data)
		}

		tcPos := bytes.Index(data[absPos:searchLimit], timecodeID)
		if tcPos != -1 {
			actualTcPos := absPos + tcPos
			out.Write(data[absPos:actualTcPos])

			lenByte := data[actualTcPos+1]
			valLen := int(lenByte & 0x0f)

			if valLen >= 1 && valLen <= 4 && actualTcPos+2+valLen <= len(data) {
				tcBytes := data[actualTcPos+2 : actualTcPos+2+valLen]
				var currentTc uint64
				if valLen == 1 {
					currentTc = uint64(tcBytes[0])
				} else if valLen == 2 {
					currentTc = uint64(binary.BigEndian.Uint16(tcBytes))
				} else if valLen == 3 {
					currentTc = uint64(tcBytes[0])<<16 | uint64(tcBytes[1])<<8 | uint64(tcBytes[2])
				} else if valLen == 4 {
					currentTc = uint64(binary.BigEndian.Uint32(tcBytes))
				}

				newTc := uint32(currentTc + offsetMs)

				out.WriteByte(0xe7)
				out.WriteByte(0x84)
				var b4 [4]byte
				binary.BigEndian.PutUint32(b4[:], newTc)
				out.Write(b4[:])

				idx = actualTcPos + 2 + valLen
			} else {
				out.Write(data[absPos : absPos+4])
				idx = absPos + 4
			}
		} else {
			out.Write(data[absPos : absPos+4])
			idx = absPos + 4
		}
	}

	return out.Bytes()
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

	const maxChunkSize int64 = 512 * 1024 // 512 KB bounded chunk size for Google Video CDN

	rangeHeader := r.Header.Get("Range")
	hasRange := rangeHeader != ""

	var startByte int64 = 0
	if hasRange && strings.HasPrefix(rangeHeader, "bytes=") {
		parts := strings.Split(strings.TrimPrefix(rangeHeader, "bytes="), "-")
		if len(parts) >= 1 && parts[0] != "" {
			if val, err := strconv.ParseInt(parts[0], 10, 64); err == nil && val >= 0 {
				startByte = val
			}
		}
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Range")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Range, Content-Length, Accept-Ranges")
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Type", "audio/webm; codecs=opus")
	w.Header().Set("Cache-Control", "public, max-age=3600")

	initialStatus := http.StatusOK
	if hasRange && startByte > 0 {
		initialStatus = http.StatusPartialContent
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-/*", startByte))
	}

	headerWritten := false
	flusher, _ := w.(http.Flusher)

	currentByte := startByte
	chunkIndex := 0

	for {
		select {
		case <-r.Context().Done():
			return http.StatusOK, nil
		default:
		}

		var targetURL string
		var upstreamRange string
		var approxMs uint64 = 0

		if currentByte == 0 {
			targetURL = streamURL
			upstreamRange = "bytes=0-524287"
		} else {
			approxMs = uint64((currentByte * 1000) / 19200)
			if strings.Contains(streamURL, "?") {
				targetURL = fmt.Sprintf("%s&begin=%d", streamURL, approxMs)
			} else {
				targetURL = fmt.Sprintf("%s?begin=%d", streamURL, approxMs)
			}
			upstreamRange = "bytes=0-524287"
		}

		targetParsed, err := url.Parse(targetURL)
		if err != nil {
			targetParsed = parsedURL
		}

		relay := p.GetRelayURL()
		var isRelayed bool
		finalReqURL := targetURL
		if useRelay && relay != "" {
			finalReqURL = relay
			isRelayed = true
		}

		outReq, err := http.NewRequestWithContext(r.Context(), method, finalReqURL, nil)
		if err != nil {
			if !headerWritten {
				return http.StatusInternalServerError, err
			}
			return http.StatusOK, nil
		}

		if isRelayed {
			outReq.Header.Set("x-relay-target", fmt.Sprintf("https://%s", targetParsed.Host))
			relayPath := targetParsed.Path
			if targetParsed.RawQuery != "" {
				relayPath += "?" + targetParsed.RawQuery
			}
			outReq.Header.Set("x-relay-path", relayPath)
		}

		outReq.Header.Set("User-Agent", "com.google.ios.youtube/20.08.3 (iPhone15,2; U; CPU iOS 18_0 like Mac OS X)")
		outReq.Header.Set("Range", upstreamRange)

		resp, err := p.httpClient.Do(outReq)
		if err != nil {
			log.Printf("[PROXY-DEBUG] Do err for chunk %d: %v", chunkIndex, err)
			if !headerWritten {
				return http.StatusBadGateway, err
			}
			return http.StatusOK, nil
		}

		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusGone {
			bodySnippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
			resp.Body.Close()
			log.Printf("[PROXY-DEBUG] Upstream %d for chunk %d (useRelay=%t) range %s: %s. Retrying direct...", resp.StatusCode, chunkIndex, useRelay, upstreamRange, string(bodySnippet))

			if isRelayed {
				// Retry current chunk directly without Cloudflare Relay
				directReq, dErr := http.NewRequestWithContext(r.Context(), method, targetURL, nil)
				if dErr == nil {
					directReq.Header.Set("User-Agent", "com.google.ios.youtube/20.08.3 (iPhone15,2; U; CPU iOS 18_0 like Mac OS X)")
					directReq.Header.Set("Range", upstreamRange)
					dResp, dDoErr := p.httpClient.Do(directReq)
					if dDoErr == nil && dResp.StatusCode < 400 {
						resp = dResp
						goto processChunkBody
					}
					if dResp != nil {
						dResp.Body.Close()
					}
				}
			}

			if !headerWritten {
				return resp.StatusCode, fmt.Errorf("upstream error %d", resp.StatusCode)
			}
			return http.StatusOK, nil
		}

processChunkBody:

		chunkData, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil && len(chunkData) == 0 {
			if !headerWritten {
				return http.StatusBadGateway, err
			}
			return http.StatusOK, nil
		}

		if len(chunkData) == 0 {
			break
		}

		writeBytes := chunkData
		if chunkIndex > 0 {
			if pos := bytes.Index(chunkData, clusterID); pos != -1 {
				rawClusters := chunkData[pos:]
				writeBytes = adjustWebMClusterTimecodesFixed(rawClusters, approxMs)
			}
		}

		if !headerWritten {
			w.WriteHeader(initialStatus)
			headerWritten = true
		}

		if method != http.MethodHead {
			if _, err := w.Write(writeBytes); err != nil {
				return http.StatusOK, nil
			}
			if flusher != nil {
				flusher.Flush()
			}
		} else {
			break
		}

		currentByte += maxChunkSize
		chunkIndex++

		if int64(len(chunkData)) < maxChunkSize {
			break
		}
	}

	return http.StatusOK, nil
}



