/**
 * Helper to construct API URLs compatible with both Web Browser and Wails Desktop (WebKit)
 * @param {string} path API Endpoint path (e.g. '/api/setup/init')
 * @returns {string} Fully resolved API URL
 */
export function getApiUrl(path) {
  if (typeof window !== 'undefined') {
    const host = window.location.hostname || '';
    const proto = window.location.protocol || '';
    if (host === 'wails.localhost' || proto === 'wails:' || proto === 'file:') {
      const base = 'http://127.0.0.1:7554';
      return path.startsWith('/') ? `${base}${path}` : `${base}/${path}`;
    }
  }
  return path;
}
