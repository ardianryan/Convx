// Devices Store — Heartbeat + SSE for multi-device tracking
import { writable, get } from 'svelte/store';
import { getDeviceInfo, getDeviceInfoSync } from './deviceInfo.js';
import { currentSong, isPlaying } from './player.js';

export const activeDevices = writable([]);
export const thisDeviceId = writable(null);

let heartbeatInterval = null;
let eventSource = null;
let started = false;

function getCookie(name) {
  if (typeof document === 'undefined') return null;
  const match = document.cookie.match(new RegExp('(^| )' + name + '=([^;]+)'));
  return match ? match[2] : null;
}

async function sendHeartbeat() {
  try {
    const info = await getDeviceInfo();
    const song = get(currentSong);
    const playing = get(isPlaying);

    const body = {
      deviceId: info.deviceId,
      name: info.name,
      deviceType: info.deviceType,
      platform: info.platform,
      browser: info.browser,
      isPlaying: playing,
      currentSong: playing && song ? { id: song.id, title: song.title, artist: song.artist } : null,
    };

    await fetch('/api/devices/heartbeat', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify(body),
    });
  } catch (err) {
    // Silent fail — will retry next interval
  }
}

function connectSSE() {
  if (typeof EventSource === 'undefined') return;
  if (eventSource) {
    try { eventSource.close(); } catch (_) {}
  }

  const token = getCookie('convx_session');
  const url = token ? `/api/devices/stream?token=${encodeURIComponent(token)}` : '/api/devices/stream';

  eventSource = new EventSource(url);

  eventSource.onmessage = (event) => {
    try {
      const devices = JSON.parse(event.data);
      // Parse currentSong JSON string if present
      const parsed = devices.map(d => ({
        ...d,
        currentSong: d.current_song ? (typeof d.current_song === 'string' ? JSON.parse(d.current_song) : d.current_song) : null,
        isPlaying: !!d.is_playing,
        deviceType: d.device_type,
        lastHeartbeat: d.last_heartbeat,
      }));
      activeDevices.set(parsed);
    } catch (e) {
      console.warn('[Devices SSE] Parse error:', e);
    }
  };

  eventSource.onerror = () => {
    // Reconnect after 5 seconds
    try { eventSource.close(); } catch (_) {}
    eventSource = null;
    setTimeout(() => {
      if (started) connectSSE();
    }, 5000);
  };
}

export function startDeviceTracking() {
  if (started || typeof window === 'undefined') return;
  started = true;

  const info = getDeviceInfoSync();
  thisDeviceId.set(info.deviceId);

  // Send initial heartbeat
  sendHeartbeat();

  // Heartbeat every 30 seconds
  heartbeatInterval = setInterval(sendHeartbeat, 30_000);

  // Connect SSE for real-time updates
  connectSSE();

  // Also send heartbeat on play/pause changes
  const unsubPlaying = isPlaying.subscribe(() => {
    if (started) sendHeartbeat();
  });

  const unsubSong = currentSong.subscribe(() => {
    if (started) sendHeartbeat();
  });

  // Send heartbeat on visibility change (tab focus)
  const onVisibility = () => {
    if (!document.hidden && started) sendHeartbeat();
  };
  document.addEventListener('visibilitychange', onVisibility);

  // Send offline signal on page unload
  const onUnload = () => {
    const info = getDeviceInfoSync();
    const body = JSON.stringify({
      deviceId: info.deviceId, name: info.name,
      deviceType: info.deviceType, platform: info.platform,
      browser: info.browser, isPlaying: false, currentSong: null,
    });
    // Use sendBeacon for reliable delivery on page close
    navigator.sendBeacon('/api/devices/heartbeat', new Blob([body], { type: 'application/json' }));
  };
  window.addEventListener('beforeunload', onUnload);

  return () => {
    started = false;
    clearInterval(heartbeatInterval);
    if (eventSource) { try { eventSource.close(); } catch (_) {} }
    unsubPlaying();
    unsubSong();
    document.removeEventListener('visibilitychange', onVisibility);
    window.removeEventListener('beforeunload', onUnload);
  };
}

export function stopDeviceTracking() {
  started = false;
  clearInterval(heartbeatInterval);
  if (eventSource) {
    try { eventSource.close(); } catch (_) {}
    eventSource = null;
  }
}
