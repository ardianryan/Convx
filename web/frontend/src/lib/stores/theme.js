import { writable, derived } from 'svelte/store';

// Available modes: 'auto' | 'light' | 'dark'
const STORAGE_KEY = 'convx_theme_mode';

function getInitialPreference() {
  if (typeof window !== 'undefined') {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved === 'light' || saved === 'dark' || saved === 'auto') {
      return saved;
    }
  }
  return 'auto';
}

export const themePreference = writable(getInitialPreference());

// Boolean store indicating if dark mode is currently active
export const isDarkActive = writable(true);

function getSystemPrefersDark() {
  if (typeof window !== 'undefined' && window.matchMedia) {
    return window.matchMedia('(prefers-color-scheme: dark)').matches;
  }
  return true;
}

export function applyTheme(preference) {
  if (typeof window === 'undefined') return;

  let shouldBeDark = true;
  if (preference === 'dark') {
    shouldBeDark = true;
  } else if (preference === 'light') {
    shouldBeDark = false;
  } else {
    // 'auto' - system
    shouldBeDark = getSystemPrefersDark();
  }

  isDarkActive.set(shouldBeDark);
  themePreference.set(preference);
  localStorage.setItem(STORAGE_KEY, preference);

  if (shouldBeDark) {
    document.documentElement.classList.add('dark');
    document.documentElement.style.colorScheme = 'dark';
  } else {
    document.documentElement.classList.remove('dark');
    document.documentElement.style.colorScheme = 'light';
  }
}

export function cycleTheme() {
  let current;
  themePreference.subscribe((val) => (current = val))();

  let next;
  if (current === 'auto') {
    next = 'light';
  } else if (current === 'light') {
    next = 'dark';
  } else {
    next = 'auto';
  }

  applyTheme(next);
  return next;
}

export function initTheme() {
  if (typeof window === 'undefined') return;

  const initial = getInitialPreference();
  applyTheme(initial);

  // Listen for system changes when in 'auto' mode
  if (window.matchMedia) {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    const handler = () => {
      let current;
      themePreference.subscribe((val) => (current = val))();
      if (current === 'auto') {
        applyTheme('auto');
      }
    };

    if (mediaQuery.addEventListener) {
      mediaQuery.addEventListener('change', handler);
    } else if (mediaQuery.addListener) {
      mediaQuery.addListener(handler);
    }
  }
}
