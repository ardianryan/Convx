import { writable } from 'svelte/store';

const STORAGE_KEY = 'convx_player_state';

// Load saved state from localStorage
function getSavedState() {
  if (typeof window === 'undefined') return null;
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    return raw ? JSON.parse(raw) : null;
  } catch (e) {
    return null;
  }
}

const savedState = getSavedState();

export const currentSong = writable(savedState?.song || null);
export const isPlaying = writable(false);
export const isLoading = writable(false);
export const currentTime = writable(savedState?.currentTime || 0);
export const duration = writable(savedState?.duration || 0);
export const volume = writable(savedState?.volume !== undefined ? savedState.volume : 0.85);
export const queue = writable(savedState?.queue || (savedState?.song ? [savedState.song] : []));
export const currentIndex = writable(savedState?.currentIndex !== undefined ? savedState.currentIndex : (savedState?.song ? 0 : -1));
export const error = writable(null);
export const playbackEngine = writable('audio'); // Default: Native HTML5 Audio via proxy

let audio = null;
let keepaliveAudio = null;
let ytPlayer = null;
let ytReady = false;
let pendingSong = null;
let pendingSeekTime = savedState?.currentTime || 0;
let updateInterval = null;
let currentVol = savedState?.volume !== undefined ? savedState.volume : 0.85;
let currentActiveSong = savedState?.song || null;
let currentEngine = 'audio';
let userExplicitlyPaused = true;
let isRetryingAlternate = false;

// Persist player state to localStorage
function saveState() {
  if (typeof window === 'undefined') return;
  try {
    let q = [];
    queue.subscribe((val) => (q = val))();
    let cIdx = 0;
    currentIndex.subscribe((val) => (cIdx = val))();
    let cTime = 0;
    currentTime.subscribe((val) => (cTime = val))();
    let cDur = 0;
    duration.subscribe((val) => (cDur = val))();

    if (currentActiveSong) {
      localStorage.setItem(
        STORAGE_KEY,
        JSON.stringify({
          song: currentActiveSong,
          currentTime: Math.floor(cTime),
          duration: Math.floor(cDur),
          volume: currentVol,
          queue: q.slice(0, 50),
          currentIndex: cIdx,
        })
      );
    }
  } catch (e) {}
}

playbackEngine.subscribe((val) => {
  currentEngine = val;
});

// Create a silent 1-second WAV blob to anchor mobile audio focus
function getSilentAudioUrl() {
  try {
    const sampleRate = 8000;
    const numSamples = 8000;
    const buffer = new ArrayBuffer(44 + numSamples);
    const view = new DataView(buffer);

    // RIFF header
    view.setUint32(0, 0x52494646, false); // "RIFF"
    view.setUint32(4, 36 + numSamples, true);
    view.setUint32(8, 0x57415645, false); // "WAVE"

    // fmt subchunk
    view.setUint32(12, 0x666d7420, false); // "fmt "
    view.setUint32(16, 16, true);
    view.setUint16(20, 1, true); // PCM
    view.setUint16(22, 1, true); // Mono
    view.setUint32(24, sampleRate, true);
    view.setUint32(28, sampleRate, true);
    view.setUint16(32, 1, true);
    view.setUint16(34, 8, true); // 8-bit

    // data subchunk
    view.setUint32(36, 0x64617461, false); // "data"
    view.setUint32(40, numSamples, true);

    const bytes = new Uint8Array(buffer, 44);
    bytes.fill(128); // 8-bit unsigned PCM midpoint (silence)

    const blob = new Blob([buffer], { type: 'audio/wav' });
    return URL.createObjectURL(blob);
  } catch (e) {
    return 'data:audio/wav;base64,UklGRjIAAABXQVZFZm10IBIAAAABAAEAQB8AAEAfAAABAAgAAABmYWN0BAAAAAAAAABkYXRhAAAAAA==';
  }
}

function startKeepaliveAudio() {
  if (typeof window === 'undefined') return;
  try {
    if (!keepaliveAudio) {
      keepaliveAudio = new Audio(getSilentAudioUrl());
      keepaliveAudio.loop = true;
      keepaliveAudio.volume = 0.001; // Tiny volume keeps mobile audio session active
    }
    const p = keepaliveAudio.play();
    if (p && p.catch) p.catch(() => {});
  } catch (e) {}
}

function pauseKeepaliveAudio() {
  if (keepaliveAudio) {
    try {
      keepaliveAudio.pause();
    } catch (e) {}
  }
}

let lastSaveTime = 0;

function startProgressTimer() {
  stopProgressTimer();
  updateInterval = setInterval(() => {
    if (currentEngine === 'video' && ytPlayer && ytReady) {
      try {
        const t = ytPlayer.getCurrentTime();
        const d = ytPlayer.getDuration();
        if (typeof t === 'number' && !isNaN(t)) {
          currentTime.set(t);
          // Throttle saving state to localStorage every 2 seconds
          const now = Date.now();
          if (now - lastSaveTime > 2000) {
            lastSaveTime = now;
            saveState();
          }
        }
        if (typeof d === 'number' && !isNaN(d) && d > 0) {
          duration.set(d);
          if ('mediaSession' in navigator && 'setPositionState' in navigator.mediaSession) {
            try {
              navigator.mediaSession.setPositionState({
                duration: Math.max(0, d),
                playbackRate: 1,
                position: Math.min(Math.max(0, t), d),
              });
            } catch (err) {}
          }
        }
      } catch (e) {
        // ignore
      }
    }
  }, 400);
}

function stopProgressTimer() {
  if (updateInterval) {
    clearInterval(updateInterval);
    updateInterval = null;
  }
  saveState();
}

// Page Visibility API Spoofing & event interceptor to prevent mobile pause on lock screen
if (typeof document !== 'undefined') {
  try {
    Object.defineProperty(document, 'hidden', {
      get: () => false,
      configurable: true,
    });
    Object.defineProperty(document, 'visibilityState', {
      get: () => 'visible',
      configurable: true,
    });
    Object.defineProperty(document, 'webkitVisibilityState', {
      get: () => 'visible',
      configurable: true,
    });
  } catch (e) {}

  window.addEventListener(
    'visibilitychange',
    (e) => {
      if (!userExplicitlyPaused) {
        e.stopImmediatePropagation();
      }
    },
    true
  );
}

// Check if YouTube Player iframe is healthy and attached to DOM
function isPlayerAttached() {
  if (!ytPlayer) return false;
  try {
    const iframe = typeof ytPlayer.getIframe === 'function' ? ytPlayer.getIframe() : null;
    return !!(iframe && document.body.contains(iframe));
  } catch (e) {
    return false;
  }
}

// Reset container and get a clean div for YouTube player
function resetHolder() {
  if (typeof document === 'undefined') return null;
  let container = document.getElementById('yt-audio-container');
  if (!container) {
    container = document.createElement('div');
    container.id = 'yt-audio-container';
    container.style.cssText =
      'position:fixed;bottom:0;right:0;width:200px;height:200px;opacity:0.001;pointer-events:none;z-index:-9999;overflow:hidden;';
    document.body.appendChild(container);
  }
  container.innerHTML = '<div id="yt-audio-holder"></div>';
  return document.getElementById('yt-audio-holder');
}

// Ensure container exists for YouTube player
function getOrCreateHolder() {
  if (typeof document === 'undefined') return null;
  let holder = document.getElementById('yt-audio-holder');
  if (holder && holder.tagName.toLowerCase() === 'div') return holder;
  return resetHolder();
}

// Create YouTube Player Instance
export function createPlayerInstance() {
  if (typeof window === 'undefined') return;
  if (ytPlayer && isPlayerAttached()) return;
  if (!window.YT || !window.YT.Player) return;

  const holder = resetHolder();
  if (!holder) return;

  try {
    ytPlayer = new window.YT.Player('yt-audio-holder', {
      height: '100%',
      width: '100%',
      host: 'https://www.youtube.com',
      playerVars: {
        autoplay: 1,
        controls: 0,
        disablekb: 1,
        enablejsapi: 1,
        fs: 0,
        rel: 0,
        playsinline: 1,
        iv_load_policy: 3,
        origin: window.location.origin,
      },
      events: {
        onReady: () => {
          console.log('[Convx Engine] YouTube Player is ready and attached!');
          ytReady = true;
          try {
            ytPlayer.setVolume(currentVol * 100);
          } catch (err) {}
          if (pendingSong) {
            const s = pendingSong;
            const seekTime = pendingSeekTime || 0;
            pendingSong = null;
            pendingSeekTime = 0;
            playSong(s, null, seekTime);
          }
        },
        onStateChange: (event) => {
          if (currentEngine !== 'video') return;
          console.log('[Convx YT State]', event.data);
          // 1: playing
          if (event.data === 1) {
            // Auto Ad-Skip check: If an ad is detected, jump to the end of ad
            try {
              const dur = ytPlayer.getDuration();
              if (
                dur &&
                dur < 35 &&
                currentActiveSong &&
                currentActiveSong.duration &&
                currentActiveSong.duration > 45
              ) {
                console.log('[Convx Ad-Guard] Ad detected, skipping...');
                ytPlayer.seekTo(dur, true);
              }
            } catch (e) {}

            isPlaying.set(true);
            isLoading.set(false);
            error.set(null);
            startProgressTimer();
            startKeepaliveAudio();
            if ('mediaSession' in navigator) {
              navigator.mediaSession.playbackState = 'playing';
            }
          } else if (event.data === 2) {
            // 2: paused
            if (!userExplicitlyPaused) {
              console.log('[Convx Mobile Guard] Auto-resuming...');
              setTimeout(() => {
                if (!userExplicitlyPaused && ytPlayer && ytReady && currentEngine === 'video') {
                  try {
                    ytPlayer.playVideo();
                  } catch (err) {}
                }
              }, 150);
            } else {
              isPlaying.set(false);
              stopProgressTimer();
              pauseKeepaliveAudio();
              if ('mediaSession' in navigator) {
                navigator.mediaSession.playbackState = 'paused';
              }
            }
          } else if (event.data === 3) {
            // 3: buffering
            isLoading.set(true);
          } else if (event.data === 5) {
            // 5: video cued
            try {
              ytPlayer.playVideo();
            } catch (e) {}
          } else if (event.data === 0) {
            // 0: ended
            isPlaying.set(false);
            stopProgressTimer();
            console.log('[Convx Engine] Track ended, playing next in queue...');
            playNext();
          }
        },
        onError: (e) => {
          console.warn('[Convx Engine] YT Player error code:', e.data);
          // On any YT error, fall back to native audio
          if (currentActiveSong && (e.data === 2 || e.data === 5 || e.data === 100 || e.data === 101 || e.data === 150)) {
            console.log('[Convx Engine] YT error, falling back to native audio engine...');
            playbackEngine.set('audio');
            currentEngine = 'audio';
            let curT = 0;
            try { curT = ytPlayer.getCurrentTime() || 0; } catch(_) {}
            playSong(currentActiveSong, null, curT);
          }
        },
      },
    });
  } catch (err) {
    console.warn('Error creating YT.Player:', err);
  }
}

// Top-level Hoisted initYT function
export function initYT() {
  if (typeof window === 'undefined') return;
  if (window.YT && window.YT.Player) {
    createPlayerInstance();
  } else {
    window.onYouTubeIframeAPIReady = createPlayerInstance;
    if (!document.querySelector('script[src*="youtube.com/iframe_api"]')) {
      const tag = document.createElement('script');
      tag.src = 'https://www.youtube.com/iframe_api';
      document.head.appendChild(tag);
    }
  }
}

// Alternate embed fallback for songs restricted by music labels (Error 150/101)
async function tryAlternateEmbedVersion(song) {
  if (isRetryingAlternate || !song) return;
  isRetryingAlternate = true;
  console.log('[Convx Engine] Finding alternate embeddable version for:', song.title);

  try {
    const query = encodeURIComponent(`${song.title} ${song.artist}`);
    const res = await fetch(`/api/search?q=${query}`);
    if (res.ok) {
      const data = await res.json();
      const results = data.results || [];
      // Pick the first result that has a different video ID
      const alt = results.find((r) => r.id && r.id !== song.id);
      if (alt && ytPlayer && typeof ytPlayer.loadVideoById === 'function') {
        console.log('[Convx Engine] Switched to alternate version:', alt.title, alt.id);
        currentActiveSong = alt;
        currentSong.set(alt);
        ytPlayer.loadVideoById({
          videoId: alt.id,
          startSeconds: 0,
        });
        ytPlayer.playVideo();
        isRetryingAlternate = false;
        return;
      }
    }
  } catch (err) {
    console.warn('Failed to find alternate version:', err);
  }

  isRetryingAlternate = false;
  error.set(`Lagu "${song.title}" dibatasi hak cipta oleh label musik. Coba lagu lain.`);
  isLoading.set(false);
  isPlaying.set(false);
}

// Native HTML5 Audio and MediaSession initialization
if (typeof window !== 'undefined') {
  audio = new Audio();
  audio.preload = 'auto';

  let audioLastSave = 0;
  audio.addEventListener('timeupdate', () => {
    if (currentEngine === 'audio') {
      currentTime.set(audio.currentTime);
      const now = Date.now();
      if (now - audioLastSave > 2000) {
        audioLastSave = now;
        saveState();
      }
    }
  });

  audio.addEventListener('loadedmetadata', () => {
    if (currentEngine === 'audio' && audio.duration && !isNaN(audio.duration)) {
      duration.set(audio.duration);
      saveState();
    }
  });

  audio.addEventListener('playing', () => {
    if (currentEngine === 'audio') {
      isPlaying.set(true);
      isLoading.set(false);
      startKeepaliveAudio();
      if ('mediaSession' in navigator) {
        navigator.mediaSession.playbackState = 'playing';
      }
    }
  });

  audio.addEventListener('pause', () => {
    if (currentEngine === 'audio') {
      if (!userExplicitlyPaused) {
        setTimeout(() => {
          if (!userExplicitlyPaused && audio) audio.play().catch(() => {});
        }, 100);
      } else {
        isPlaying.set(false);
        pauseKeepaliveAudio();
        if ('mediaSession' in navigator) {
          navigator.mediaSession.playbackState = 'paused';
        }
      }
    }
  });

  audio.addEventListener('waiting', () => {
    if (currentEngine === 'audio') isLoading.set(true);
  });

  audio.addEventListener('ended', () => {
    if (currentEngine === 'audio') playNext();
  });

  audio.addEventListener('error', (e) => {
    if (currentEngine === 'audio') {
      console.warn('HTML5 Audio playback error, falling back to YouTube iframe engine...', e, audio.error);
      if (currentActiveSong) {
        playbackEngine.set('video');
        currentEngine = 'video';
        let curT = audio.currentTime || 0;
        playSong(currentActiveSong, null, curT);
      } else {
        isLoading.set(false);
        isPlaying.set(false);
        error.set('Tidak dapat memutar lagu ini. Coba lagu lain atau refresh.');
      }
    }
  });

  if (document.readyState === 'loading') {
    window.addEventListener('DOMContentLoaded', initYT);
  } else {
    initYT();
  }

  if ('mediaSession' in navigator) {
    navigator.mediaSession.setActionHandler('play', () => {
      userExplicitlyPaused = false;
      togglePlay();
    });
    navigator.mediaSession.setActionHandler('pause', () => {
      userExplicitlyPaused = true;
      togglePlay();
    });
    navigator.mediaSession.setActionHandler('previoustrack', () => playPrev());
    navigator.mediaSession.setActionHandler('nexttrack', () => playNext());
    navigator.mediaSession.setActionHandler('seekto', (details) => {
      if (details.seekTime !== undefined) {
        seek(details.seekTime);
      }
    });
  }
}

export async function playSong(song, newQueue = null, startSeconds = 0) {
  if (!song || !song.id) return;

  userExplicitlyPaused = false;
  startKeepaliveAudio();
  error.set(null);
  isLoading.set(true);
  currentTime.set(startSeconds);
  currentSong.set(song);
  currentActiveSong = song;

  if (newQueue) {
    queue.set(newQueue);
    const idx = newQueue.findIndex((s) => s.id === song.id);
    currentIndex.set(idx !== -1 ? idx : 0);
  }

  saveState();

  // Update MediaSession
  if (typeof navigator !== 'undefined' && 'mediaSession' in navigator) {
    navigator.mediaSession.metadata = new MediaMetadata({
      title: song.title,
      artist: song.artist,
      album: song.album || 'Convx Music',
      artwork: [{ src: song.thumbnail, sizes: '512x512', type: 'image/jpeg' }],
    });
    navigator.mediaSession.playbackState = 'playing';
  }

  // Pure Native Audio Mode
  if (currentEngine === 'audio') {
    if (ytPlayer && ytReady) {
      try {
        ytPlayer.pauseVideo();
      } catch (e) {}
    }
    stopProgressTimer();

    try {
      const proxyUrl = `/api/proxy/audio/${song.id}`;
      audio.pause();
      audio.src = proxyUrl;
      audio.volume = currentVol;
      audio.load();
      if (startSeconds > 0) {
        audio.currentTime = startSeconds;
      }
      await audio.play();
      isPlaying.set(true);
      isLoading.set(false);
      return;
    } catch (err) {
      console.warn('Native audio play error:', err);
      isLoading.set(false);
      isPlaying.set(false);
      error.set('Gagal memutar audio: ' + (err.message || 'Stream error'));
      return;
    }
  }

  // Hidden YouTube IFrame Engine
  if (audio) {
    try {
      audio.pause();
      audio.removeAttribute('src');
      audio.load();
    } catch (e) {}
  }

  if (ytReady && ytPlayer && isPlayerAttached() && typeof ytPlayer.loadVideoById === 'function') {
    try {
      ytPlayer.loadVideoById({
        videoId: song.id,
        startSeconds: startSeconds || 0,
      });
      ytPlayer.playVideo();

      // Autoplay watchdog: if stuck buffering / loading for more than 4 seconds
      setTimeout(() => {
        let loadingNow = false;
        let playingNow = false;
        isLoading.subscribe((l) => (loadingNow = l))();
        isPlaying.subscribe((p) => (playingNow = p))();
        if (loadingNow && !playingNow && currentEngine === 'video') {
          console.log('[Convx Engine] Autoplay watchdog: re-triggering playVideo');
          if (ytPlayer && ytReady && isPlayerAttached()) {
            try {
              ytPlayer.playVideo();
            } catch (_) {}
          }
          isLoading.set(false);
        }
      }, 4000);

      return;
    } catch (err) {
      console.warn('Failed to loadVideoById on ytPlayer:', err);
    }
  }

  // If not ready or player became detached, queue and recreate player
  pendingSong = song;
  pendingSeekTime = startSeconds || 0;
  if (typeof window !== 'undefined') {
    ytReady = false;
    ytPlayer = null;
    initYT();
  }
}

export function togglePlay() {
  if (currentEngine === 'video') {
    if (ytPlayer && ytReady && isPlayerAttached()) {
      try {
        const state = ytPlayer.getPlayerState();
        // -1: unstarted, 0: ended, 1: playing, 2: paused, 3: buffering, 5: cued
        if (state === 1) {
          userExplicitlyPaused = true;
          pauseKeepaliveAudio();
          ytPlayer.pauseVideo();
          saveState();
        } else {
          userExplicitlyPaused = false;
          startKeepaliveAudio();
          ytPlayer.playVideo();
        }
        return;
      } catch (e) {
        console.warn('togglePlay yt error:', e);
      }
    }
    if (currentActiveSong) {
      let curT = 0;
      currentTime.subscribe((t) => (curT = t))();
      playSong(currentActiveSong, null, curT);
      return;
    }
  }

  if (!audio) return;
  if (audio.paused) {
    userExplicitlyPaused = false;
    startKeepaliveAudio();
    audio.play().catch(console.error);
  } else {
    userExplicitlyPaused = true;
    pauseKeepaliveAudio();
    audio.pause();
    saveState();
  }
}

export function seek(seconds) {
  currentTime.set(seconds);
  saveState();
  if (currentEngine === 'video' && ytPlayer && ytReady) {
    try {
      ytPlayer.seekTo(seconds, true);
      return;
    } catch (e) {}
  }

  if (!audio) return;
  audio.currentTime = seconds;
}

export function setVolume(vol) {
  const clamped = Math.max(0, Math.min(1, vol));
  currentVol = clamped;
  volume.set(clamped);
  saveState();

  if (currentEngine === 'video' && ytPlayer && ytReady) {
    try {
      ytPlayer.setVolume(clamped * 100);
    } catch (e) {}
  }

  if (audio) {
    audio.volume = clamped;
  }
}

export function playNext() {
  queue.update((q) => {
    currentIndex.update((idx) => {
      if (idx + 1 < q.length) {
        const nextSong = q[idx + 1];
        playSong(nextSong);
        return idx + 1;
      } else if (q.length > 0) {
        playSong(q[0]);
        return 0;
      }
      return idx;
    });
    return q;
  });
}

export function playPrev() {
  if (currentEngine === 'video' && ytPlayer && ytReady) {
    try {
      if (ytPlayer.getCurrentTime() > 3) {
        ytPlayer.seekTo(0, true);
        currentTime.set(0);
        return;
      }
    } catch (e) {}
  } else if (audio && audio.currentTime > 3) {
    audio.currentTime = 0;
    return;
  }

  queue.update((q) => {
    currentIndex.update((idx) => {
      if (idx > 0) {
        const prevSong = q[idx - 1];
        playSong(prevSong);
        return idx - 1;
      }
      return idx;
    });
    return q;
  });
}

export function addToQueue(song) {
  queue.update((q) => {
    if (!q.find((s) => s.id === song.id)) {
      const updated = [...q, song];
      setTimeout(saveState, 50);
      return updated;
    }
    return q;
  });
}
