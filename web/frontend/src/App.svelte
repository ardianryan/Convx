<script>
  import { onMount } from 'svelte';
  import {
    Disc3,
    Radio,
    Sparkles,
    AlertCircle,
    Key,
    Play,
    Pause,
    Search,
    Compass,
    ListMusic,
    Flame,
    Music2,
    Clock,
    User,
    Disc,
    Music,
    Layers,
    ChevronRight,
    Smartphone,
    Heart,
    Smile,
    Zap,
    Coffee,
    MoreHorizontal,
    X,
    FolderPlus,
    Mic2,
    Mic,
    Headphones,
    Globe,
    TrendingUp,
    Tv,
    Library,
    Settings,
    Cloud,
    Loader2,
    Monitor,
    Tablet,
    Info
  } from 'lucide-svelte';
  import SearchBar from './lib/components/SearchBar.svelte';
  import TrackList from './lib/components/TrackList.svelte';
  import PlayerBar from './lib/components/PlayerBar.svelte';
  import AccountModal from './lib/components/AccountModal.svelte';
  import AboutModal from './lib/components/AboutModal.svelte';
  import OnboardingWizard from './lib/components/OnboardingWizard.svelte';
  import LoginView from './lib/components/LoginView.svelte';
  import SettingsModal from './lib/components/SettingsModal.svelte';
  import ThemeToggle from './lib/components/ThemeToggle.svelte';
  import { getApiUrl } from './lib/api.js';
  import { initTheme } from './lib/stores/theme.js';
  import { currentSong, isPlaying, error, queue, playSong, togglePlay } from './lib/stores/player.js';
  import { activeDevices, thisDeviceId, startDeviceTracking } from './lib/stores/devices.js';

  // System Setup & Auth State
  let isSystemLoading = true;
  let isInitialized = false;
  let isAuthenticated = false;
  let currentUser = null;
  let platformName = 'Convx Music';
  let activeRelay = null;
  let showSettingsModal = false;
  let showAboutModal = false;

  function getInitials(name) {
    if (!name) return 'U';
    const parts = name.trim().split(/\s+/);
    if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
    return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
  }

  $: userDisplayName = currentUser?.name || currentUser?.username || 'Ryan Ardian';
  $: userInitials = getInitials(userDisplayName);

  function handleUserNameChange(newName) {
    if (currentUser) {
      currentUser = { ...currentUser, name: newName };
    }
  }

  // Dynamic Browser Document Title based on current track and playback state
  $: if (typeof document !== 'undefined') {
    if ($currentSong && $isPlaying) {
      document.title = `▶ ${$currentSong.title} • ${$currentSong.artist} — ${platformName}`;
    } else if ($currentSong) {
      document.title = `⏸ ${$currentSong.title} • ${$currentSong.artist} — ${platformName}`;
    } else {
      document.title = `${platformName} — Apple Style Music Player`;
    }
  }

  // Navigation tab: 'home' (Beranda) | 'new' (Baru) | 'radio' (Radio) | 'library' (Perpustakaan) | 'search' (Pencarian)
  let activeTab = 'home';

  let query = '';
  let loading = false;
  let tracks = [];
  let searchTitle = 'Diputar Baru-Baru Ini';
  let showAccountModal = false;
  let isLoggedIn = false;
  let showSyncBanner = true;

  // Browse Categories matching Apple Music screenshot with rich gradients & icons
  const browseCategories = [
    { title: 'Apple Music Radio', gradient: 'from-rose-500 to-red-600', icon: Radio, query: 'Apple Music Radio Hits' },
    { title: 'Alternative', gradient: 'from-amber-500 to-orange-600', icon: Disc, query: 'Alternative Rock Best' },
    { title: 'Indonesian Music', gradient: 'from-blue-600 to-indigo-700', icon: Globe, query: 'Indonesian Pop Music' },
    { title: 'Pop', gradient: 'from-pink-500 to-rose-600', icon: Sparkles, query: 'Top Pop Hits Global' },
    { title: 'K-Pop', gradient: 'from-fuchsia-600 to-pink-600', icon: Flame, query: 'K-Pop Top Hits' },
    { title: 'Rock', gradient: 'from-orange-600 to-red-700', icon: Zap, query: 'Classic & Modern Rock' },
    { title: 'R&B', gradient: 'from-purple-600 to-violet-800', icon: Music2, query: 'Indonesian R&B Soul' },
    { title: 'Hip-Hop/Rap', gradient: 'from-indigo-600 to-blue-800', icon: Mic2, query: 'Hip Hop Rap Hits' },
    { title: 'Dance', gradient: 'from-emerald-500 to-teal-700', icon: Disc3, query: 'Electronic Dance EDM' },
    { title: 'Charts', gradient: 'from-slate-700 to-stone-900', icon: TrendingUp, query: 'Top Charts Billboard' },
    { title: 'Sing', gradient: 'from-red-500 to-rose-600', icon: Mic, query: 'Karaoke Sing Along Hits' },
    { title: 'Spatial Audio', gradient: 'from-rose-600 to-pink-700', icon: Headphones, query: 'Spatial Audio Dolby Atmos' },
    { title: 'Apple Music Live', gradient: 'from-slate-800 to-neutral-950', icon: Tv, query: 'Live Concert Acoustic' },
    { title: 'Love', gradient: 'from-amber-700 to-rose-800', icon: Heart, query: 'Romantic Love Songs' },
    { title: 'Party', gradient: 'from-violet-900 to-purple-950', icon: Smile, query: 'Weekend Party Playlist' }
  ];

  // Recently Searched Pills matching screenshot 1
  const recentlySearched = [
    { title: 'Top 100: Indonesia', subtitle: 'Playlist', query: 'Top 100 Indonesia' },
    { title: 'Roar', subtitle: 'Song • Katy Perry', query: 'Roar Katy Perry' },
    { title: 'Randy Pangalila', subtitle: 'Artist', query: 'Randy Pangalila' },
    { title: 'Denny Caknan Essentials', subtitle: 'Playlist', query: 'Denny Caknan' },
    { title: 'Sal Priadi', subtitle: 'Artist', query: 'Sal Priadi' }
  ];

  // Top Picks for You (Hero Cards from screenshot 3 - Beranda)
  const topPicks = [
    {
      badge: 'Dibuat untuk Anda',
      title: 'New Music',
      desc: 'Alex Crichton, Ndarboy Genk, Happy Asmara, Whisnu Santika, Rony Parulian, dan lainnya',
      gradient: 'from-pink-500 via-rose-500 to-red-600',
      icon: Heart,
      query: 'Top Hits Indonesia Terbaru'
    },
    {
      badge: 'Suasana hati untuk Anda',
      title: 'Sukaria',
      desc: 'Deretan trek paling menyenangkan untuk cerahkan hari Anda.',
      gradient: 'from-amber-500 via-yellow-500 to-orange-500',
      icon: Zap,
      query: 'Sukaria Semangat Pop Indonesia'
    },
    {
      badge: 'Rilis Baru',
      title: 'Setelah Obrolan Jam 3 Pagi',
      desc: 'Lomba Sihir',
      gradient: 'from-slate-800 to-indigo-950',
      icon: Coffee,
      query: 'Lomba Sihir Setelah Obrolan Jam 3 Pagi'
    }
  ];

  // Radio Station Grids matching screenshot 2
  const radioStations = [
    { title: 'Apple Music 1', subtitle: 'Music Radio', color: 'text-red-500', num: '1', query: 'Apple Music 1 Live' },
    { title: 'HITS', subtitle: 'Music Radio', color: 'text-sky-400', num: 'HITS', query: 'Hits Radio Hits' },
    { title: 'COUNTRY', subtitle: 'Music Radio', color: 'text-amber-400', num: '★', query: 'Country Radio' },
    { title: 'MÚSICA UNO', subtitle: 'Music Radio', color: 'text-pink-500', num: 'UNO', query: 'Musica Latina' },
    { title: 'club', subtitle: 'Music Radio', color: 'text-white', num: 'club', query: 'Club Dance Radio' },
    { title: 'Chill', subtitle: 'Music Radio', color: 'text-blue-400', num: 'Chill', query: 'Chill Lo-Fi Radio' }
  ];

  let hasSearched = false;

  async function checkAccountStatus() {
    try {
      const res = await fetch(getApiUrl('/api/account/status'));
      if (res.ok) {
        const data = await res.json();
        isLoggedIn = !!data.isLoggedIn;
      }
    } catch (e) {
      console.warn('Account status check error:', e);
    }
  }

  async function handleSearch(e, forceSearchTab = false) {
    const q = typeof e?.detail === 'string' ? e.detail : query;
    if (!q) return;

    loading = true;
    error.set(null);
    try {
      const res = await fetch(getApiUrl(`/api/search?q=${encodeURIComponent(q)}`));
      if (!res.ok) throw new Error(`Search failed: HTTP ${res.status}`);
      const data = await res.json();
      tracks = data.results || [];
      searchTitle = `Hasil: "${q}"`;
      hasSearched = true;
      if (forceSearchTab) {
        activeTab = 'search';
      }
    } catch (err) {
      console.error('Search error:', err);
      error.set(err.message || 'Gagal mencari lagu');
    } finally {
      loading = false;
    }
  }

  function selectCategory(catQuery) {
    query = catQuery;
    handleSearch({ detail: catQuery }, true);
  }

  function handleClearSearch() {
    query = '';
    hasSearched = false;
    searchTitle = 'Pencarian';
  }

  async function checkSystemAuth() {
    isSystemLoading = true;
    try {
      const res = await fetch(getApiUrl('/api/auth/me'));
      if (res.ok) {
        const data = await res.json();
        isInitialized = !!data.isInitialized;
        isAuthenticated = !!data.isLoggedIn;
        currentUser = data.user || null;
        platformName = data.platformName || 'Convx Music';
        activeRelay = data.activeRelay || null;
      }
    } catch (e) {
      console.warn('Auth status check error:', e);
    } finally {
      isSystemLoading = false;
    }
  }

  function handleOnboardingComplete(data) {
    isInitialized = true;
    isAuthenticated = true;
    currentUser = data.user;
    platformName = data.platformName || platformName;
    checkAccountStatus();
    handleSearch({ detail: 'Top 100 Indonesia' });
    startDeviceTracking();
  }

  function handleLoginSuccess(data) {
    isAuthenticated = true;
    currentUser = data.user;
    platformName = data.platformName || platformName;
    checkAccountStatus();
    handleSearch({ detail: 'Top 100 Indonesia' });
    startDeviceTracking();
  }

  async function handleLogout() {
    try {
      await fetch(getApiUrl('/api/auth/logout'), { method: 'POST' });
    } catch (_) {}
    isAuthenticated = false;
    currentUser = null;
    showSettingsModal = false;
  }

  onMount(async () => {
    initTheme();
    await checkSystemAuth();
    if (isAuthenticated) {
      checkAccountStatus();
      handleSearch({ detail: 'Top 100 Indonesia' });
      startDeviceTracking();
    }
  });
</script>

{#if isSystemLoading}
  <div class="min-h-screen w-screen bg-[#0a0a0c] flex flex-col items-center justify-center text-white select-none">
    <div class="w-14 h-14 rounded-2xl bg-gradient-to-tr from-red-500 to-pink-500 flex items-center justify-center shadow-xl shadow-red-500/30 mb-4 animate-pulse">
      <Disc3 class="w-7 h-7 text-white animate-spin-slow" />
    </div>
    <div class="text-sm font-semibold tracking-tight">{platformName}</div>
    <div class="text-xs text-neutral-500 mt-1 flex items-center gap-1.5">
      <Loader2 class="w-3.5 h-3.5 animate-spin" />
      <span>Menghubungkan ke database SQLite...</span>
    </div>
  </div>
{:else if !isInitialized}
  <OnboardingWizard onComplete={handleOnboardingComplete} />
{:else if !isAuthenticated}
  <LoginView {platformName} onSuccess={handleLoginSuccess} />
{:else}
  <!-- Ambient Dynamic iOS & macOS Mesh Glow -->
  <div class="fixed inset-0 pointer-events-none overflow-hidden -z-10 bg-[#f5f5f7] dark:bg-[#0f1015] transition-colors duration-300">
    {#if $currentSong?.thumbnail}
      <div
        class="absolute -top-[15%] left-1/2 -translate-x-1/2 w-[120vw] max-w-[1000px] h-[65vh] rounded-full opacity-15 dark:opacity-30 blur-[130px] transition-all duration-1000 ease-out"
        style="background: radial-gradient(circle, #fa2d48 0%, #6366f1 50%, #000 100%); background-image: url('{$currentSong.thumbnail}'); background-size: cover; background-position: center; filter: blur(95px) saturate(200%);"
      ></div>
    {:else}
      <div class="absolute -top-40 -left-40 w-96 h-96 bg-rose-600/10 dark:bg-rose-600/15 rounded-full blur-[120px]"></div>
      <div class="absolute top-1/3 -right-40 w-96 h-96 bg-indigo-600/10 dark:bg-indigo-600/15 rounded-full blur-[120px]"></div>
    {/if}
  </div>

  <!-- Main App Frame (macOS & Responsive Mobile Layout) -->
  <div class="flex h-screen w-screen overflow-hidden text-neutral-900 dark:text-white font-[-apple-system,BlinkMacSystemFont,'SF_Pro_Display','SF_Pro_Text','Inter',sans-serif]">

  <!-- ========================================================================= -->
  <!-- 1. LEFT SIDEBAR (Apple Music macOS Exact Sidebar, hidden on mobile)       -->
  <!-- ========================================================================= -->
  <aside class="hidden md:flex flex-col w-60 lg:w-64 bg-[#f2f2f7]/90 dark:bg-[#14161f]/85 backdrop-blur-3xl border-r border-black/[0.08] dark:border-white/[0.08] select-none shrink-0 z-20 transition-colors">
    <!-- Window Traffic Lights (Red, Yellow, Green macOS buttons) -->
    <div class="flex items-center gap-2 px-5 pt-4 pb-2">
      <div class="w-3 h-3 rounded-full bg-[#ff5f56] border border-[#e0443e]/40 shadow-inner"></div>
      <div class="w-3 h-3 rounded-full bg-[#ffbd2e] border border-[#dea123]/40 shadow-inner"></div>
      <div class="w-3 h-3 rounded-full bg-[#27c93f] border border-[#1aab29]/40 shadow-inner"></div>
    </div>

    <!-- Sidebar Navigation Menu -->
    <div class="flex-1 overflow-y-auto px-3 py-2 space-y-5 text-[13px] no-scrollbar">
      <!-- Main Section -->
      <div class="space-y-0.5">
        <button
          on:click={() => (activeTab = 'home')}
          class="w-full flex items-center gap-2.5 px-3 py-1.5 rounded-lg font-medium transition-all {activeTab === 'home' ? 'bg-[#fa2d48]/15 text-[#fa2d48]' : 'text-neutral-700 dark:text-white/80 hover:bg-black/[0.05] dark:hover:bg-white/[0.06] hover:text-black dark:hover:text-white'}"
        >
          <Compass class="w-4 h-4 {activeTab === 'home' ? 'text-[#fa2d48]' : 'text-neutral-500 dark:text-white/60'}" />
          <span>Beranda</span>
        </button>

        <button
          on:click={() => {
            activeTab = 'new';
            handleSearch({ detail: 'Indonesian Music Today' });
          }}
          class="w-full flex items-center gap-2.5 px-3 py-1.5 rounded-lg font-medium transition-all {activeTab === 'new' ? 'bg-[#fa2d48]/15 text-[#fa2d48]' : 'text-neutral-700 dark:text-white/80 hover:bg-black/[0.05] dark:hover:bg-white/[0.06] hover:text-black dark:hover:text-white'}"
        >
          <Layers class="w-4 h-4 {activeTab === 'new' ? 'text-[#fa2d48]' : 'text-neutral-500 dark:text-white/60'}" />
          <span>Baru</span>
        </button>

        <button
          on:click={() => {
            activeTab = 'radio';
            handleSearch({ detail: 'Apple Music Radio' });
          }}
          class="w-full flex items-center gap-2.5 px-3 py-1.5 rounded-lg font-medium transition-all {activeTab === 'radio' ? 'bg-[#fa2d48]/15 text-[#fa2d48]' : 'text-neutral-700 dark:text-white/80 hover:bg-black/[0.05] dark:hover:bg-white/[0.06] hover:text-black dark:hover:text-white'}"
        >
          <Radio class="w-4 h-4 {activeTab === 'radio' ? 'text-[#fa2d48]' : 'text-neutral-500 dark:text-white/60'}" />
          <span>Radio</span>
        </button>

        <button
          on:click={() => (activeTab = 'library')}
          class="w-full flex items-center gap-2.5 px-3 py-1.5 rounded-lg font-medium transition-all {activeTab === 'library' ? 'bg-[#fa2d48]/15 text-[#fa2d48]' : 'text-neutral-700 dark:text-white/80 hover:bg-black/[0.05] dark:hover:bg-white/[0.06] hover:text-black dark:hover:text-white'}"
        >
          <Library class="w-4 h-4 {activeTab === 'library' ? 'text-[#fa2d48]' : 'text-neutral-500 dark:text-white/60'}" />
          <span>Perpustakaan</span>
        </button>

        <button
          on:click={() => (activeTab = 'search')}
          class="w-full flex items-center gap-2.5 px-3 py-1.5 rounded-lg font-medium transition-all {activeTab === 'search' ? 'bg-[#fa2d48]/15 text-[#fa2d48]' : 'text-neutral-700 dark:text-white/80 hover:bg-black/[0.05] dark:hover:bg-white/[0.06] hover:text-black dark:hover:text-white'}"
        >
          <Search class="w-4 h-4 {activeTab === 'search' ? 'text-[#fa2d48]' : 'text-neutral-500 dark:text-white/60'}" />
          <span>Cari</span>
        </button>
      </div>

      <!-- Library Section -->
      <div class="space-y-1">
        <h3 class="px-3 text-[11px] font-semibold text-neutral-400 dark:text-white/40 uppercase tracking-wider">Perpustakaan</h3>
        <div class="space-y-0.5">
          <button
            on:click={() => (activeTab = 'library')}
            class="w-full flex items-center gap-2.5 px-3 py-1.5 rounded-lg text-neutral-700 dark:text-white/70 hover:bg-black/[0.05] dark:hover:bg-white/[0.06] hover:text-black dark:hover:text-white transition-all"
          >
            <ListMusic class="w-4 h-4 text-[#fa2d48]" />
            <span>Daftar Putar</span>
          </button>
          <button
            on:click={() => selectCategory('Top Indonesian Artists')}
            class="w-full flex items-center gap-2.5 px-3 py-1.5 rounded-lg text-neutral-700 dark:text-white/70 hover:bg-black/[0.05] dark:hover:bg-white/[0.06] hover:text-black dark:hover:text-white transition-all"
          >
            <Mic2 class="w-4 h-4 text-[#fa2d48]" />
            <span>Artis</span>
          </button>
          <button
            on:click={() => selectCategory('Album Indonesia Terbaik')}
            class="w-full flex items-center gap-2.5 px-3 py-1.5 rounded-lg text-neutral-700 dark:text-white/70 hover:bg-black/[0.05] dark:hover:bg-white/[0.06] hover:text-black dark:hover:text-white transition-all"
          >
            <Disc class="w-4 h-4 text-[#fa2d48]" />
            <span>Album</span>
          </button>
          <button
            on:click={() => selectCategory('Lagu Populer Indonesia')}
            class="w-full flex items-center gap-2.5 px-3 py-1.5 rounded-lg text-neutral-700 dark:text-white/70 hover:bg-black/[0.05] dark:hover:bg-white/[0.06] hover:text-black dark:hover:text-white transition-all"
          >
            <Music class="w-4 h-4 text-[#fa2d48]" />
            <span>Lagu</span>
          </button>
        </div>
      </div>

      <!-- Devices Section -->
      <div class="space-y-1">
        <h3 class="px-3 text-[11px] font-semibold text-neutral-400 dark:text-white/40 uppercase tracking-wider">Perangkat</h3>
        {#if $activeDevices.length > 0}
          {#each $activeDevices as device (device.id)}
            <div class="flex items-center justify-between px-3 py-1.5 text-neutral-700 dark:text-white/70 {device.id === $thisDeviceId ? 'bg-black/[0.04] dark:bg-white/[0.06] rounded-lg' : ''}">
              <div class="flex items-center gap-2.5 min-w-0">
                {#if device.deviceType === 'desktop'}
                  <Monitor class="w-4 h-4 shrink-0 {device.isPlaying ? 'text-emerald-500' : 'text-neutral-400 dark:text-white/50'}" />
                {:else if device.deviceType === 'tablet'}
                  <Tablet class="w-4 h-4 shrink-0 {device.isPlaying ? 'text-emerald-500' : 'text-neutral-400 dark:text-white/50'}" />
                {:else}
                  <Smartphone class="w-4 h-4 shrink-0 {device.isPlaying ? 'text-emerald-500' : 'text-neutral-400 dark:text-white/50'}" />
                {/if}
                <div class="flex flex-col min-w-0">
                  <span class="text-xs font-medium truncate">{device.name}</span>
                  <span class="text-[10px] text-neutral-400 dark:text-white/35 truncate">
                    {#if device.id === $thisDeviceId}
                      Perangkat ini · {device.platform}
                    {:else if device.isPlaying && device.currentSong}
                      {device.currentSong.title}
                    {:else}
                      {device.platform} · {device.browser}
                    {/if}
                  </span>
                </div>
              </div>
              <span class="w-2 h-2 rounded-full shrink-0 {device.isPlaying ? 'bg-emerald-400 shadow-sm shadow-emerald-400/50' : 'bg-amber-400/60'}"></span>
            </div>
          {/each}
        {:else}
          <div class="px-3 py-1.5 text-neutral-700 dark:text-white/70 flex items-center gap-2.5">
            <Monitor class="w-4 h-4 text-neutral-400 dark:text-white/50" />
            <span class="text-xs text-neutral-400 dark:text-white/40">Menghubungkan...</span>
          </div>
        {/if}
      </div>
    </div>

    <!-- User Profile Footer -->
    <div class="p-3 border-t border-black/[0.08] dark:border-white/[0.06] flex items-center justify-between">
      <div class="flex items-center gap-2.5 min-w-0">
        <div class="w-8 h-8 rounded-full bg-gradient-to-tr from-rose-500 to-amber-500 flex items-center justify-center font-bold text-xs shadow-md text-white shrink-0">
          {userInitials}
        </div>
        <div class="flex flex-col min-w-0">
          <span class="text-xs font-semibold leading-tight text-neutral-900 dark:text-white truncate">{userDisplayName}</span>
          <span class="text-[10px] text-neutral-500 dark:text-white/40">{isLoggedIn ? 'YouTube Linked' : 'Admin'}</span>
        </div>
      </div>
      <div class="flex items-center gap-1.5">
        <button
          type="button"
          on:click={() => (showAboutModal = true)}
          class="w-7 h-7 rounded-full bg-black/[0.05] dark:bg-white/[0.06] hover:bg-black/[0.1] dark:hover:bg-white/[0.12] flex items-center justify-center text-neutral-600 dark:text-white/60 hover:text-black dark:hover:text-white transition-colors cursor-pointer"
          title="Tentang Aplikasi Convx"
        >
          <Info class="w-3.5 h-3.5" />
        </button>
        <button
          on:click={() => (showAccountModal = true)}
          class="w-7 h-7 rounded-full bg-black/[0.05] dark:bg-white/[0.06] hover:bg-black/[0.1] dark:hover:bg-white/[0.12] flex items-center justify-center text-neutral-600 dark:text-white/60 hover:text-black dark:hover:text-white transition-colors cursor-pointer"
          title="Pengaturan Akun & Cookie"
        >
          <Key class="w-3.5 h-3.5" />
        </button>
      </div>
    </div>
  </aside>

  <!-- ========================================================================= -->
  <!-- 2. MAIN CONTENT VIEWPORT (Scrollable)                                     -->
  <!-- ========================================================================= -->
  <div class="flex-1 flex flex-col h-full overflow-hidden relative">

    <!-- Top Navigation Header (Desktop macOS bar & Mobile iOS top bar) -->
    <header class="h-14 px-4 sm:px-8 border-b border-black/[0.06] dark:border-white/[0.06] bg-white/80 dark:bg-[#10121a]/70 backdrop-blur-2xl flex items-center justify-between shrink-0 z-10 transition-colors">
      <!-- Left: Mobile Brand / Desktop Sync Status -->
      <div class="flex items-center gap-3">
        <div class="flex md:hidden items-center gap-2">
          <div class="w-7 h-7 rounded-xl bg-gradient-to-tr from-[#fa2d48] to-[#ff6480] flex items-center justify-center shadow-md text-white">
            <Disc3 class="w-4 h-4 animate-spin-slow" />
          </div>
          <span class="font-bold text-sm truncate max-w-[120px] text-neutral-900 dark:text-white">{platformName}</span>
        </div>

        <div class="hidden sm:flex items-center gap-2.5">
          <!-- Cloudflare Relay Pill -->
          {#if activeRelay && activeRelay.isActive}
            <button
              type="button"
              on:click={() => (showSettingsModal = true)}
              class="flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-emerald-500/15 border border-emerald-500/30 text-emerald-700 dark:text-emerald-300 text-[11px] font-medium hover:bg-emerald-500/25 transition-all cursor-pointer"
              title="Cloudflare Relay Aktif: {activeRelay.url}"
            >
              <span class="w-1.5 h-1.5 rounded-full bg-emerald-500 dark:bg-emerald-400 animate-pulse"></span>
              <span>Relay Aktif</span>
            </button>
          {:else}
            <button
              type="button"
              on:click={() => (showSettingsModal = true)}
              class="flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-black/5 dark:bg-white/5 border border-black/10 dark:border-white/10 text-neutral-600 dark:text-neutral-400 text-[11px] font-medium hover:bg-black/10 dark:hover:bg-white/10 transition-all cursor-pointer"
              title="Direct Mode (Klik untuk aktifkan relay)"
            >
              <Cloud class="w-3 h-3 text-neutral-500 dark:text-neutral-400" />
              <span>Direct Mode</span>
            </button>
          {/if}

          <span class="text-xs text-neutral-300 dark:text-white/30">•</span>

          <p class="text-xs text-neutral-500 dark:text-white/40">
            {#if isLoggedIn}
              <span class="text-emerald-600 dark:text-emerald-400 font-semibold">● Sync YouTube</span>
            {:else}
              <button on:click={() => (showAccountModal = true)} class="text-neutral-600 dark:text-white/60 hover:text-black dark:hover:text-white underline">Connect YouTube</button>
            {/if}
          </p>
        </div>
      </div>

      <!-- Center Search Bar (Hidden on mobile home, shown on search tab or desktop) -->
      <div class="w-56 sm:w-80 md:w-96 {activeTab !== 'search' ? 'hidden sm:block' : 'block'}">
        <SearchBar bind:query {loading} on:search={(e) => handleSearch(e, true)} on:clear={handleClearSearch} />
      </div>

      <!-- Right Apple Profile, Theme Toggle & Settings Shortcut -->
      <div class="flex items-center gap-2">
        <!-- 1-Icon Theme Cycle Toggle (Auto -> Light -> Dark) -->
        <ThemeToggle />

        <button
          type="button"
          on:click={() => (showAboutModal = true)}
          class="w-8 h-8 rounded-full bg-black/5 dark:bg-white/5 hover:bg-black/10 dark:hover:bg-white/10 flex items-center justify-center text-neutral-600 dark:text-neutral-300 hover:text-neutral-900 dark:hover:text-white transition-colors cursor-pointer border border-black/5 dark:border-white/5"
          title="Tentang Aplikasi Convx Web"
        >
          <Info class="w-4 h-4" />
        </button>

        <button
          type="button"
          on:click={() => (showSettingsModal = true)}
          class="w-8 h-8 rounded-full bg-black/5 dark:bg-white/5 hover:bg-black/10 dark:hover:bg-white/10 flex items-center justify-center text-neutral-600 dark:text-neutral-300 hover:text-neutral-900 dark:hover:text-white transition-colors cursor-pointer border border-black/5 dark:border-white/5"
          title="Pengaturan Sistem & Cloudflare Relay"
        >
          <Settings class="w-4 h-4" />
        </button>

        <button
          type="button"
          on:click={() => (showSettingsModal = true)}
          class="w-8 h-8 rounded-full bg-gradient-to-tr from-rose-500 to-amber-500 flex items-center justify-center font-bold text-xs shadow-md border border-white/20 active:scale-95 transition-all text-white cursor-pointer"
          title="Profil: {userDisplayName}"
        >
          {userInitials}
        </button>
      </div>
    </header>

    <!-- Error Alert Banner -->
    {#if $error}
      <div class="px-6 py-2 bg-[#fa2d48]/15 border-b border-[#fa2d48]/30 flex items-center justify-between text-xs text-[#ff8090]">
        <div class="flex items-center gap-2">
          <AlertCircle class="w-4 h-4 text-[#fa2d48]" />
          <span>{$error}</span>
        </div>
        <button on:click={() => error.set(null)} class="px-2 py-0.5 rounded bg-white/10 text-white/70 hover:text-white">
          Tutup
        </button>
      </div>
    {/if}

    <!-- Main Scroll Container -->
    <main class="flex-1 overflow-y-auto px-4 sm:px-8 py-5 pb-48 lg:pb-36 space-y-6">

      <!-- =================================================================== -->
      <!-- TAB 1: BERANDA (Matching Screenshot 3)                             -->
      <!-- =================================================================== -->
      {#if activeTab === 'home'}
        <div class="space-y-6 anim-tab-view max-w-5xl mx-auto">
          <!-- iOS Large Title Header -->
          <div class="flex items-center justify-between pt-1">
            <h1 class="text-3xl sm:text-4xl font-extrabold text-neutral-900 dark:text-white tracking-tight">Beranda</h1>
            <div class="flex items-center gap-2">
              <button
                on:click={() => (showAccountModal = true)}
                class="w-9 h-9 rounded-full bg-black/5 dark:bg-white/10 hover:bg-black/10 dark:hover:bg-white/20 flex items-center justify-center text-neutral-700 dark:text-white/80 btn-pressable cursor-pointer"
                title="Pilihan Akun"
              >
                <MoreHorizontal class="w-5 h-5" />
              </button>
            </div>
          </div>

          <!-- Pilihan Teratas untuk Anda (Hero Cards from Screenshot 3) -->
          <section class="space-y-3">
            <h2 class="text-lg font-bold text-neutral-900 dark:text-white tracking-tight">Pilihan Teratas untuk Anda</h2>
            <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
              {#each topPicks as pick}
                <div
                  role="button"
                  tabindex="0"
                  on:click={() => selectCategory(pick.query)}
                  on:keydown={(e) => e.key === 'Enter' && selectCategory(pick.query)}
                  class="relative h-72 sm:h-80 rounded-3xl overflow-hidden p-6 flex flex-col justify-between bg-gradient-to-br {pick.gradient} shadow-xl group cursor-pointer border border-white/15 transition-all duration-300 hover:scale-[1.02] hover:-translate-y-1 hover:shadow-2xl active:scale-[0.98]"
                >
                  <!-- Apple Music Logo Badge -->
                  <div class="flex items-center justify-between z-10">
                    <div></div>
                    <span class="text-xs font-bold text-white/95 flex items-center gap-1 drop-shadow-sm">
                      Music
                    </span>
                  </div>

                  <!-- Big Bold Typography Hero Title -->
                  <div class="flex-1 flex flex-col items-center justify-center text-center z-10 px-2">
                    <h3 class="text-4xl sm:text-5xl font-black text-white leading-tight drop-shadow-md group-hover:scale-105 transition-transform duration-300">
                      {pick.title}
                    </h3>
                  </div>

                  <!-- Bottom Title & Desc -->
                  <div class="space-y-1 z-10">
                    <span class="text-[11px] font-semibold tracking-wider text-white/80">
                      {pick.badge}
                    </span>
                    <p class="text-xs text-white/90 line-clamp-2 leading-relaxed drop-shadow-sm font-medium">
                      {pick.desc}
                    </p>
                  </div>
                </div>
              {/each}
            </div>
          </section>

          <!-- Diputar Baru-Baru Ini (Recently Played) -->
          <section class="space-y-3">
            <div class="flex items-center justify-between">
              <h2 class="text-lg font-bold text-neutral-900 dark:text-white tracking-tight flex items-center gap-1.5">
                <span>Diputar Baru-Baru Ini</span>
                <ChevronRight class="w-4 h-4 text-neutral-400 dark:text-white/40" />
              </h2>
            </div>
            <TrackList {tracks} title="" />
          </section>
        </div>

      <!-- =================================================================== -->
      <!-- TAB 2: BARU (Matching Screenshot 4)                                -->
      <!-- =================================================================== -->
      {:else if activeTab === 'new'}
        <div class="space-y-6 anim-tab-view max-w-5xl mx-auto">
          <!-- iOS Large Title Header -->
          <div class="pt-1">
            <h1 class="text-3xl sm:text-4xl font-extrabold text-neutral-900 dark:text-white tracking-tight">Baru</h1>
          </div>

          <!-- Featured Large Banner Cards -->
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div
              role="button"
              tabindex="0"
              on:click={() => selectCategory('Indonesian Music Today')}
              on:keydown={(e) => e.key === 'Enter' && selectCategory('Indonesian Music Today')}
              class="relative h-64 sm:h-72 rounded-3xl overflow-hidden p-6 flex flex-col justify-between bg-gradient-to-t from-black/85 via-black/30 to-transparent border border-white/15 cursor-pointer shadow-xl group"
              style="background-image: url('https://images.unsplash.com/photo-1511671782779-c97d3d27a1d4?w=800&auto=format&fit=crop&q=60'); background-size: cover; background-position: center;"
            >
              <div class="absolute inset-0 bg-gradient-to-t from-black/90 via-black/30 to-transparent"></div>
              <div class="z-10 space-y-0.5">
                <span class="text-[11px] uppercase font-bold text-white/70 tracking-wider">DAFTAR PUTAR DIPERBARUI</span>
                <h3 class="text-2xl sm:text-3xl font-black text-white">Indonesian Music Today</h3>
                <p class="text-xs text-white/60">Apple Music: Indonesia</p>
              </div>

              <div class="z-10 mt-auto">
                <p class="text-xs text-white/80 font-medium">Potret kekayaan warna musik tanah air dalam kurasi karya terkini.</p>
              </div>
            </div>

            <div
              role="button"
              tabindex="0"
              on:click={() => selectCategory('Aruma Dan Ternyata Aku Cukup')}
              on:keydown={(e) => e.key === 'Enter' && selectCategory('Aruma Dan Ternyata Aku Cukup')}
              class="relative h-64 sm:h-72 rounded-3xl overflow-hidden p-6 flex flex-col justify-between bg-gradient-to-t from-black/85 via-black/30 to-transparent border border-white/15 cursor-pointer shadow-xl group"
              style="background-image: url('https://images.unsplash.com/photo-1493225457124-a3eb161ffa5f?w=800&auto=format&fit=crop&q=60'); background-size: cover; background-position: center;"
            >
              <div class="absolute inset-0 bg-gradient-to-t from-black/90 via-black/30 to-transparent"></div>
              <div class="z-10 space-y-0.5">
                <span class="text-[11px] uppercase font-bold text-white/70 tracking-wider">ALBUM MINI</span>
                <h3 class="text-2xl sm:text-3xl font-black text-white">Dan Ternyata Aku Cukup - EP</h3>
                <p class="text-xs text-white/60">Aruma</p>
              </div>

              <div class="z-10 mt-auto">
                <p class="text-xs text-white/80 font-medium">Katarsis personal Aruma yang memikat lewat melodi atmosferik.</p>
              </div>
            </div>
          </div>

          <!-- Favoritkan Hit Viral Ini (Track list) -->
          <section class="space-y-3">
            <h2 class="text-lg font-bold text-neutral-900 dark:text-white tracking-tight flex items-center gap-1.5">
              <span>☆ Favoritkan Hit Viral Ini</span>
              <ChevronRight class="w-4 h-4 text-neutral-400 dark:text-white/40" />
            </h2>
            <TrackList {tracks} title="" />
          </section>
        </div>

      <!-- =================================================================== -->
      <!-- TAB 3: RADIO (Matching Screenshot 2)                               -->
      <!-- =================================================================== -->
      {:else if activeTab === 'radio'}
        <div class="space-y-6 anim-tab-view max-w-5xl mx-auto">
          <!-- iOS Large Title Header -->
          <div class="pt-1">
            <h1 class="text-3xl sm:text-4xl font-extrabold text-neutral-900 dark:text-white tracking-tight">Radio</h1>
          </div>

          <!-- Top 6 Radio Badges (Matching Screenshot 2 exactly) -->
          <div class="grid grid-cols-3 sm:grid-cols-6 gap-3">
            {#each radioStations as station}
              <button
                on:click={() => selectCategory(station.query)}
                class="aspect-square rounded-2xl bg-black/[0.04] dark:bg-white/[0.06] hover:bg-black/[0.08] dark:hover:bg-white/[0.1] active:scale-95 border border-black/10 dark:border-white/10 p-3 flex flex-col items-center justify-between shadow-sm dark:shadow-md transition-all group btn-pressable cursor-pointer"
              >
                <div class="flex-1 flex items-center justify-center">
                  <span class="text-3xl sm:text-4xl font-black {station.color} group-hover:scale-110 transition-transform">
                    {station.num}
                  </span>
                </div>
                <div class="text-center">
                  <span class="text-[10px] font-bold text-neutral-600 dark:text-white/70 block">Music</span>
                  <span class="text-[9px] font-bold text-rose-500 dark:text-rose-400">Radio</span>
                </div>
              </button>
            {/each}
          </div>

          <!-- Stasiun Siaran Langsung -->
          <section class="space-y-3">
            <div>
              <h2 class="text-lg font-bold text-neutral-900 dark:text-white tracking-tight">Stasiun Siaran Langsung</h2>
              <p class="text-xs text-neutral-500 dark:text-white/50">Ketuk stasiun untuk menyimak musik dan banyak lagi — nonstop.</p>
            </div>

            <!-- Big Live Feature Card -->
            <div
              role="button"
              tabindex="0"
              on:click={() => selectCategory('Apple Music 1 Live')}
              on:keydown={(e) => e.key === 'Enter' && selectCategory('Apple Music 1 Live')}
              class="relative rounded-3xl overflow-hidden bg-white/80 dark:bg-[#181a24] border border-black/10 dark:border-white/15 p-6 flex flex-col justify-between min-h-[220px] cursor-pointer hover:border-black/20 dark:hover:border-white/30 transition-all duration-300 hover:scale-[1.01] hover:-translate-y-0.5 shadow-xl backdrop-blur-md"
            >
              <div class="flex flex-col items-center justify-center py-4">
                <span class="text-7xl font-black text-rose-500">1</span>
                <span class="text-sm font-bold text-neutral-900 dark:text-white mt-1">Music <span class="text-rose-500">Radio</span></span>
              </div>

              <div class="pt-3 border-t border-black/10 dark:border-white/10 flex items-center justify-between">
                <div>
                  <span class="text-[11px] font-medium text-neutral-500 dark:text-white/50">Music 1 • 07.00–09.00</span>
                  <h4 class="text-base font-bold text-neutral-900 dark:text-white">Apple Music 1 Live Show</h4>
                  <p class="text-xs text-neutral-600 dark:text-white/70">The world's best new music is on Apple Music 1.</p>
                </div>
                <button class="w-10 h-10 rounded-full bg-neutral-900 dark:bg-white text-white dark:text-slate-950 flex items-center justify-center shadow-lg btn-pressable cursor-pointer">
                  <Play class="w-5 h-5 fill-current ml-0.5" />
                </button>
              </div>
            </div>
          </section>

          <!-- Radio Tracks List -->
          <section class="space-y-3">
            <TrackList {tracks} title="Pilihan Lagu Radio" />
          </section>
        </div>

      <!-- =================================================================== -->
      <!-- TAB 4: PERPUSTAKAAN (Matching Screenshot 1)                         -->
      <!-- =================================================================== -->
      {:else if activeTab === 'library'}
        <div class="space-y-6 anim-tab-view max-w-5xl mx-auto">
          <!-- iOS Large Title Header -->
          <div class="flex items-center justify-between pt-1">
            <h1 class="text-3xl sm:text-4xl font-extrabold text-neutral-900 dark:text-white tracking-tight">Perpustakaan</h1>
            <div class="flex items-center gap-2">
              <button
                on:click={() => (showAccountModal = true)}
                class="w-9 h-9 rounded-full bg-black/5 dark:bg-white/10 hover:bg-black/10 dark:hover:bg-white/20 flex items-center justify-center text-neutral-700 dark:text-white/80 btn-pressable cursor-pointer"
                title="Pilihan"
              >
                <MoreHorizontal class="w-5 h-5" />
              </button>
            </div>
          </div>

          <!-- Selaraskan perpustakaan banner (Matching Screenshot 1) -->
          {#if showSyncBanner}
            <div class="relative rounded-2xl bg-black/[0.04] dark:bg-white/[0.08] backdrop-blur-xl border border-black/10 dark:border-white/10 p-4 flex items-center justify-between gap-3 shadow-md">
              <div class="flex-1 text-center sm:text-left pr-4">
                <p class="text-xs sm:text-sm font-medium text-neutral-800 dark:text-white/90">
                  Selaraskan perpustakaan di semua perangkat Anda.
                </p>
                <button
                  on:click={() => (showAccountModal = true)}
                  class="text-xs font-bold text-[#fa2d48] hover:underline mt-0.5 inline-block"
                >
                  Nyalakan
                </button>
              </div>
              <button
                on:click={() => (showSyncBanner = false)}
                class="w-6 h-6 rounded-full bg-black/5 dark:bg-white/10 hover:bg-black/10 dark:hover:bg-white/20 flex items-center justify-center text-neutral-500 dark:text-white/60 hover:text-black dark:hover:text-white shrink-0 btn-pressable cursor-pointer"
              >
                <X class="w-3.5 h-3.5" />
              </button>
            </div>
          {/if}

          <!-- Library Menu Rows (Daftar Putar, Artis, Album, Lagu) -->
          <div class="rounded-2xl bg-black/[0.03] dark:bg-white/[0.04] border border-black/10 dark:border-white/10 divide-y divide-black/[0.06] dark:divide-white/[0.06] overflow-hidden">
            <button
              on:click={() => (activeTab = 'queue')}
              class="w-full p-4 flex items-center justify-between hover:bg-black/[0.04] dark:hover:bg-white/[0.06] transition-colors group text-left"
            >
              <div class="flex items-center gap-3">
                <ListMusic class="w-5 h-5 text-[#fa2d48]" />
                <span class="text-base font-semibold text-neutral-900 dark:text-white">Daftar Putar</span>
              </div>
              <div class="flex items-center gap-1.5 text-neutral-400 dark:text-white/40">
                {#if $queue.length > 0}
                  <span class="text-xs font-bold px-2 py-0.5 rounded-full bg-[#fa2d48] text-white">
                    {$queue.length}
                  </span>
                {/if}
                <ChevronRight class="w-4 h-4" />
              </div>
            </button>

            <button
              on:click={() => selectCategory('Artis Top Indonesia')}
              class="w-full p-4 flex items-center justify-between hover:bg-black/[0.04] dark:hover:bg-white/[0.06] transition-colors group text-left"
            >
              <div class="flex items-center gap-3">
                <Mic2 class="w-5 h-5 text-[#fa2d48]" />
                <span class="text-base font-semibold text-neutral-900 dark:text-white">Artis</span>
              </div>
              <ChevronRight class="w-4 h-4 text-neutral-400 dark:text-white/40" />
            </button>

            <button
              on:click={() => selectCategory('Album Indonesia')}
              class="w-full p-4 flex items-center justify-between hover:bg-black/[0.04] dark:hover:bg-white/[0.06] transition-colors group text-left"
            >
              <div class="flex items-center gap-3">
                <Disc class="w-5 h-5 text-[#fa2d48]" />
                <span class="text-base font-semibold text-neutral-900 dark:text-white">Album</span>
              </div>
              <ChevronRight class="w-4 h-4 text-neutral-400 dark:text-white/40" />
            </button>

            <button
              on:click={() => selectCategory('Lagu Hits Populer Indonesia')}
              class="w-full p-4 flex items-center justify-between hover:bg-black/[0.04] dark:hover:bg-white/[0.06] transition-colors group text-left"
            >
              <div class="flex items-center gap-3">
                <Music class="w-5 h-5 text-[#fa2d48]" />
                <span class="text-base font-semibold text-neutral-900 dark:text-white">Lagu</span>
              </div>
              <ChevronRight class="w-4 h-4 text-neutral-400 dark:text-white/40" />
            </button>
          </div>

          <!-- Current Queue if Available -->
          {#if $queue.length > 0}
            <section class="space-y-3 pt-2">
              <h2 class="text-lg font-bold text-neutral-900 dark:text-white tracking-tight">Antrean Saat Ini</h2>
              <TrackList tracks={$queue} title="" />
            </section>
          {/if}
        </div>

      <!-- =================================================================== -->
      <!-- TAB 5: PENCARIAN (Search)                                           -->
      <!-- =================================================================== -->
      {:else if activeTab === 'search'}
        <div class="space-y-6 anim-tab-view max-w-5xl mx-auto">
          <div class="pt-1">
            <h1 class="text-3xl sm:text-4xl font-extrabold text-neutral-900 dark:text-white tracking-tight">Pencarian</h1>
          </div>

          <!-- Cupertino Search Bar -->
          <SearchBar bind:query {loading} on:search={handleSearch} on:clear={handleClearSearch} />

          {#if !hasSearched}
            <!-- Recently Searched Row (Horizontal Cards) - Only shown before search -->
            <section class="space-y-3">
              <div class="flex items-center justify-between">
                <h2 class="text-lg font-bold text-neutral-900 dark:text-white tracking-tight">Pencarian Terkini</h2>
                <button on:click={handleClearSearch} class="text-xs font-semibold text-[#fa2d48] hover:underline">Hapus</button>
              </div>

              <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-5 gap-3">
                {#each recentlySearched as item}
                  <button
                    on:click={() => selectCategory(item.query)}
                    class="flex items-center gap-3 p-3 rounded-2xl bg-black/[0.04] dark:bg-white/[0.05] hover:bg-black/[0.08] dark:hover:bg-white/[0.1] border border-black/[0.06] dark:border-white/[0.08] text-left transition-all duration-200 hover:-translate-y-0.5 active:scale-[0.98] group cursor-pointer"
                  >
                    <div class="w-10 h-10 rounded-xl bg-neutral-200 dark:bg-slate-800 flex items-center justify-center shrink-0 border border-black/5 dark:border-white/10 group-hover:scale-105 transition-transform">
                      <Music2 class="w-5 h-5 text-neutral-600 dark:text-white/60 group-hover:text-[#fa2d48] transition-colors" />
                    </div>
                    <div class="min-w-0 flex-1">
                      <h4 class="text-xs font-semibold text-neutral-900 dark:text-white truncate">{item.title}</h4>
                      <p class="text-[11px] text-neutral-500 dark:text-white/40 truncate">{item.subtitle}</p>
                    </div>
                  </button>
                {/each}
              </div>
            </section>

            <!-- Browse Categories Grid - Only shown before search -->
            <section class="space-y-3">
              <h2 class="text-lg font-bold text-neutral-900 dark:text-white tracking-tight">Jelajahi Kategori</h2>
              <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-3.5">
                {#each browseCategories as cat}
                  <button
                    on:click={() => selectCategory(cat.query)}
                    class="h-28 sm:h-32 rounded-2xl p-3.5 flex flex-col justify-between text-left shadow-lg transition-all duration-300 hover:scale-[1.03] hover:-translate-y-1 hover:shadow-2xl active:scale-[0.98] border border-white/10 bg-gradient-to-br {cat.gradient} group relative overflow-hidden select-none cursor-pointer"
                  >
                    <!-- Background Vignette Overlay -->
                    <div class="absolute inset-0 bg-gradient-to-t from-black/70 via-black/15 to-white/10 pointer-events-none"></div>

                    <!-- Large Stylized Watermark Icon in bottom-right -->
                    <div class="absolute -right-2 -bottom-2 w-20 h-20 sm:w-24 sm:h-24 text-white/15 group-hover:text-white/30 group-hover:scale-110 group-hover:-rotate-12 transition-all duration-300 pointer-events-none flex items-center justify-center">
                      <svelte:component this={cat.icon} class="w-full h-full stroke-[1.5]" />
                    </div>

                    <!-- Top Glassy Icon Badge -->
                    <div class="w-8 h-8 rounded-xl bg-black/25 backdrop-blur-md border border-white/20 flex items-center justify-center text-white shadow-sm z-10 group-hover:bg-black/35 group-hover:scale-105 transition-all">
                      <svelte:component this={cat.icon} class="w-4 h-4" />
                    </div>

                    <!-- Bottom Title -->
                    <div class="z-10 mt-auto">
                      <span class="text-sm sm:text-base font-black text-white leading-tight drop-shadow-md group-hover:underline">
                        {cat.title}
                      </span>
                    </div>
                  </button>
                {/each}
              </div>
            </section>
          {:else}
            <!-- Search Results List - Shown immediately without scrolling down -->
            <section class="space-y-3 pt-1 anim-tab-view">
              <div class="flex items-center justify-between">
                <button
                  on:click={handleClearSearch}
                  class="inline-flex items-center gap-1.5 text-xs font-semibold text-neutral-500 hover:text-neutral-900 dark:text-neutral-400 dark:hover:text-white transition-colors cursor-pointer btn-pressable"
                >
                  <span>&larr; Lihat Semua Kategori</span>
                </button>
              </div>
              <TrackList {tracks} title={searchTitle} />
            </section>
          {/if}
        </div>

      <!-- =================================================================== -->
      <!-- TAB 6: ANTREAN KHUSUS                                              -->
      <!-- =================================================================== -->
      {:else if activeTab === 'queue'}
        <div class="space-y-6 anim-tab-view max-w-5xl mx-auto">
          <div class="flex items-center justify-between pt-1">
            <h1 class="text-3xl sm:text-4xl font-extrabold text-neutral-900 dark:text-white tracking-tight">Antrean Putar</h1>
            <button
              on:click={() => (activeTab = 'home')}
              class="text-xs font-semibold text-[#fa2d48] hover:underline btn-pressable"
            >
              Kembali ke Beranda
            </button>
          </div>
          {#if $queue.length === 0}
            <div class="py-24 text-center text-neutral-400 dark:text-white/40">Antrean masih kosong.</div>
          {:else}
            <TrackList tracks={$queue} title="Daftar Antrean Saat Ini" />
          {/if}
        </div>
      {/if}
    </main>

    <!-- Bottom Audio Player Floating Bar (Apple Music Dock) -->
    <PlayerBar />

    <!-- ========================================================================= -->
    <!-- 3. CUPERTINO MOBILE FLOATING CAPSULE TAB BAR (Matching Exact iOS Images)  -->
    <!-- ========================================================================= -->
    <div class="md:hidden fixed bottom-3 left-3 right-3 z-40 flex items-center justify-center pointer-events-none">
      <div class="flex items-center gap-2 pointer-events-auto">
        <!-- Main Navigation Capsule -->
        <nav class="h-14 px-3 rounded-full bg-white/90 dark:bg-[#1c1e28]/95 backdrop-blur-3xl border border-black/10 dark:border-white/15 shadow-[0_16px_36px_rgba(0,0,0,0.15)] dark:shadow-[0_16px_36px_rgba(0,0,0,0.6)] flex items-center gap-1 sm:gap-2 transition-colors">
          <!-- 1. Beranda -->
          <button
            on:click={() => (activeTab = 'home')}
            class="flex flex-col items-center justify-center px-3 py-1 rounded-full transition-all active:scale-95 {activeTab === 'home' ? 'bg-black/5 dark:bg-white/10 text-[#fa2d48]' : 'text-neutral-500 dark:text-white/50 hover:text-neutral-900 dark:hover:text-white'}"
          >
            <Compass class="w-5 h-5 stroke-[2.2]" />
            <span class="text-[9px] font-semibold tracking-tight">Beranda</span>
          </button>

          <!-- 2. Baru -->
          <button
            on:click={() => {
              activeTab = 'new';
              handleSearch({ detail: 'Indonesian Music Today' });
            }}
            class="flex flex-col items-center justify-center px-3 py-1 rounded-full transition-all active:scale-95 {activeTab === 'new' ? 'bg-black/5 dark:bg-white/10 text-[#fa2d48]' : 'text-neutral-500 dark:text-white/50 hover:text-neutral-900 dark:hover:text-white'}"
          >
            <Layers class="w-5 h-5 stroke-[2.2]" />
            <span class="text-[9px] font-semibold tracking-tight">Baru</span>
          </button>

          <!-- 3. Radio -->
          <button
            on:click={() => {
              activeTab = 'radio';
              handleSearch({ detail: 'Apple Music Radio' });
            }}
            class="flex flex-col items-center justify-center px-3 py-1 rounded-full transition-all active:scale-95 {activeTab === 'radio' ? 'bg-black/5 dark:bg-white/10 text-[#fa2d48]' : 'text-neutral-500 dark:text-white/50 hover:text-neutral-900 dark:hover:text-white'}"
          >
            <Radio class="w-5 h-5 stroke-[2.2]" />
            <span class="text-[9px] font-semibold tracking-tight">Radio</span>
          </button>

          <!-- 4. Perpustakaan -->
          <button
            on:click={() => (activeTab = 'library')}
            class="flex flex-col items-center justify-center px-3 py-1 rounded-full transition-all active:scale-95 {activeTab === 'library' ? 'bg-black/5 dark:bg-white/10 text-[#fa2d48]' : 'text-neutral-500 dark:text-white/50 hover:text-neutral-900 dark:hover:text-white'}"
          >
            <Library class="w-5 h-5 stroke-[2.2]" />
            <span class="text-[9px] font-semibold tracking-tight">Perpustakaan</span>
          </button>
        </nav>

        <!-- Separate Circular Search Bubble (Matching Screenshot) -->
        <button
          on:click={() => (activeTab = 'search')}
          class="w-14 h-14 rounded-full bg-white/90 dark:bg-[#1c1e28]/95 backdrop-blur-3xl border border-black/10 dark:border-white/15 shadow-[0_16px_36px_rgba(0,0,0,0.15)] dark:shadow-[0_16px_36px_rgba(0,0,0,0.6)] flex items-center justify-center text-neutral-800 dark:text-white active:scale-90 transition-transform {activeTab === 'search' ? 'text-[#fa2d48] border-[#fa2d48]/40' : 'text-neutral-600 dark:text-white/70 hover:text-neutral-900 dark:hover:text-white'}"
          title="Pencarian"
        >
          <Search class="w-6 h-6 stroke-[2.4]" />
        </button>
      </div>
    </div>

  </div>

  <!-- Settings & Relay Modal -->
  <SettingsModal
    isOpen={showSettingsModal}
    onClose={() => {
      showSettingsModal = false;
      checkSystemAuth();
    }}
    onLogout={handleLogout}
    onOpenCookieModal={() => (showAccountModal = true)}
    currentPlatformName={platformName}
    onPlatformNameChange={(newName) => (platformName = newName)}
    onRelayChange={(relay) => (activeRelay = relay)}
    currentUserName={userDisplayName}
    onUserNameChange={handleUserNameChange}
  />

  <!-- YouTube Cookie Account Modal -->
  <AccountModal 
    isOpen={showAccountModal} 
    onClose={() => { 
      showAccountModal = false; 
      checkAccountStatus(); 
    }} 
  />

  <!-- About Convx Modal -->
  <AboutModal
    isOpen={showAboutModal}
    onClose={() => (showAboutModal = false)}
  />
</div>
{/if}



