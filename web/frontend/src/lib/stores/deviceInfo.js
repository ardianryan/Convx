// Device Info Detection Store
// Detects device type, platform, browser using User-Agent Client Hints API with UA fallback

const DEVICE_ID_KEY = 'convx_device_id';

function generateId() {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) {
    return crypto.randomUUID();
  }
  return 'dev-' + Math.random().toString(36).substring(2, 15) + Date.now().toString(36);
}

function getOrCreateDeviceId() {
  if (typeof window === 'undefined') return generateId();
  let id = localStorage.getItem(DEVICE_ID_KEY);
  if (!id) {
    id = generateId();
    localStorage.setItem(DEVICE_ID_KEY, id);
  }
  return id;
}

function parseUserAgent() {
  const ua = navigator.userAgent || '';

  // Detect device type
  let deviceType = 'desktop';
  if (/iPad|tablet/i.test(ua) || (navigator.maxTouchPoints > 1 && /Macintosh/i.test(ua))) {
    deviceType = 'tablet';
  } else if (/Mobile|iPhone|iPod|Android.*Mobile|webOS|BlackBerry/i.test(ua)) {
    deviceType = 'mobile';
  }

  // Detect platform
  let platform = 'Unknown';
  if (/Mac OS X|Macintosh/i.test(ua)) platform = 'macOS';
  else if (/Windows/i.test(ua)) platform = 'Windows';
  else if (/CrOS/i.test(ua)) platform = 'Chrome OS';
  else if (/Linux/i.test(ua)) platform = 'Linux';
  else if (/iPhone|iPad|iPod/i.test(ua)) platform = 'iOS';
  else if (/Android/i.test(ua)) platform = 'Android';

  // Detect browser
  let browser = 'Browser';
  if (/Edg\//i.test(ua)) browser = 'Edge';
  else if (/OPR\//i.test(ua) || /Opera/i.test(ua)) browser = 'Opera';
  else if (/SamsungBrowser/i.test(ua)) browser = 'Samsung';
  else if (/Chrome/i.test(ua) && !/Chromium/i.test(ua)) browser = 'Chrome';
  else if (/Safari/i.test(ua) && !/Chrome/i.test(ua)) browser = 'Safari';
  else if (/Firefox/i.test(ua)) browser = 'Firefox';

  // Generate friendly device name
  let name = `${platform} ${browser}`;
  if (deviceType === 'mobile') {
    if (/iPhone/i.test(ua)) {
      name = 'iPhone';
    } else {
      const match = ua.match(/;\s*([^;)]+)\s*Build\//);
      name = match ? match[1].trim() : `${platform} Phone`;
    }
  } else if (deviceType === 'tablet') {
    if (/iPad/i.test(ua)) {
      name = 'iPad';
    } else {
      name = `${platform} Tablet`;
    }
  } else {
    // Desktop: use platform name
    if (platform === 'macOS') name = 'Mac';
    else if (platform === 'Windows') name = 'Windows PC';
    else if (platform === 'Linux') name = 'Linux PC';
    else if (platform === 'Chrome OS') name = 'Chromebook';
  }

  return { deviceType, platform, browser, name };
}

async function detectWithClientHints() {
  // Use high-entropy UA Client Hints for better accuracy
  if (navigator.userAgentData && navigator.userAgentData.getHighEntropyValues) {
    try {
      const hints = await navigator.userAgentData.getHighEntropyValues([
        'platform', 'platformVersion', 'model', 'mobile',
      ]);

      const deviceType = hints.mobile ? 'mobile' :
        (navigator.maxTouchPoints > 1 ? 'tablet' : 'desktop');

      let platform = hints.platform || 'Unknown';
      if (platform === 'macOS') platform = 'macOS';
      else if (platform === 'Windows') platform = 'Windows';
      else if (platform === 'Android') platform = 'Android';
      else if (platform === 'iOS') platform = 'iOS';
      else if (platform === 'Chrome OS') platform = 'Chrome OS';
      else if (platform === 'Linux') platform = 'Linux';

      // Get browser brand
      let browser = 'Browser';
      const brands = navigator.userAgentData.brands || [];
      for (const b of brands) {
        const bn = b.brand || '';
        if (/Edge/i.test(bn)) { browser = 'Edge'; break; }
        if (/Opera/i.test(bn)) { browser = 'Opera'; break; }
        if (/Chrome/i.test(bn) && !/Chromium/i.test(bn)) { browser = 'Chrome'; }
        if (/Safari/i.test(bn)) { browser = 'Safari'; }
        if (/Firefox/i.test(bn)) { browser = 'Firefox'; break; }
      }

      // Generate name
      let name = `${platform} ${browser}`;
      if (deviceType === 'mobile') {
        name = hints.model || `${platform} Phone`;
      } else if (deviceType === 'tablet') {
        name = hints.model || `${platform} Tablet`;
      } else {
        if (platform === 'macOS') name = 'Mac';
        else if (platform === 'Windows') name = 'Windows PC';
        else if (platform === 'Linux') name = 'Linux PC';
        else if (platform === 'Chrome OS') name = 'Chromebook';
      }

      return { deviceType, platform, browser, name };
    } catch (e) {
      // Fall back to UA parsing
    }
  }
  return null;
}

let _deviceInfo = null;

export async function getDeviceInfo() {
  if (_deviceInfo) return _deviceInfo;

  const deviceId = getOrCreateDeviceId();

  // Try Client Hints first, then fall back to UA parsing
  let detected = await detectWithClientHints();
  if (!detected) {
    detected = parseUserAgent();
  }

  _deviceInfo = {
    deviceId,
    ...detected,
  };

  return _deviceInfo;
}

// Sync version using cached or UA-only
export function getDeviceInfoSync() {
  if (_deviceInfo) return _deviceInfo;

  const deviceId = getOrCreateDeviceId();
  const detected = parseUserAgent();

  _deviceInfo = { deviceId, ...detected };
  return _deviceInfo;
}
