<script>
  import {
    Play,
    Pause,
    SkipBack,
    SkipForward,
    Volume1,
    Volume2,
    VolumeX,
    ListMusic,
    Loader2,
    X,
    Trash2,
    ChevronDown,
    Sparkles,
    Radio,
    MessageSquareQuote,
    AlignLeft
  } from 'lucide-svelte';
  import {
    currentSong,
    isPlaying,
    isLoading,
    currentTime,
    duration,
    volume,
    queue,
    currentIndex,
    togglePlay,
    seek,
    setVolume,
    playNext,
    playPrev,
    playSong
  } from '../stores/player.js';
  import { getApiUrl } from '../api.js';

  let isExpanded = false;
  let isClosingSheet = false;
  let showQueue = false;
  let isClosingQueue = false;
  let showLyrics = true; // Default to true so lyrics are shown immediately on mobile & desktop
  let isMuted = false;
  let prevVolume = 0.85;

  function closeSheet() {
    if (isClosingSheet) return;
    isClosingSheet = true;
    setTimeout(() => {
      isExpanded = false;
      isClosingSheet = false;
    }, 280);
  }

  function closeQueue() {
    if (isClosingQueue) return;
    isClosingQueue = true;
    setTimeout(() => {
      showQueue = false;
      isClosingQueue = false;
    }, 220);
  }

  // Lyrics state
  let lyricsLoading = false;
  let rawLyrics = null;
  let parsedLyrics = []; // [{ time: float, text: string }]
  let activeLyricIndex = -1;
  let lyricsContainer = null;
  let lastFetchedSongId = null;
  let isUserScrolling = false;
  let userScrollTimeout = null;

  function handleUserScroll() {
    isUserScrolling = true;
    if (userScrollTimeout) clearTimeout(userScrollTimeout);
    // Resume auto-scroll after 3 seconds of user inactivity
    userScrollTimeout = setTimeout(() => {
      isUserScrolling = false;
      scrollToActiveLyric(true);
    }, 3000);
  }

  function scrollToActiveLyric(force = false) {
    if (!lyricsContainer || activeLyricIndex < 0) return;
    if (isUserScrolling && !force) return;

    const activeElem = lyricsContainer.querySelector(`[data-lyric-index="${activeLyricIndex}"]`);
    if (activeElem) {
      // Calculate top position within container for exact centering
      const containerHeight = lyricsContainer.clientHeight;
      const elemTop = activeElem.offsetTop;
      const elemHeight = activeElem.clientHeight;
      const targetScroll = elemTop - containerHeight / 2 + elemHeight / 2;

      lyricsContainer.scrollTo({
        top: Math.max(0, targetScroll),
        behavior: 'smooth'
      });
    }
  }

  function parseLRC(lrcText) {
    if (!lrcText) return [];
    const lines = lrcText.split('\n');
    const result = [];
    const timeRegex = /\[(\d{1,2}):(\d{2})(?:[\.:](\d{1,3}))?\]/g;

    for (const line of lines) {
      const match = [...line.matchAll(timeRegex)];
      if (match.length > 0) {
        const text = line.replace(timeRegex, '').trim();
        if (text) {
          for (const m of match) {
            const min = parseInt(m[1], 10);
            const sec = parseInt(m[2], 10);
            const ms = m[3] ? parseFloat('0.' + m[3]) : 0;
            const totalSec = min * 60 + sec + ms;
            result.push({ time: totalSec, text });
          }
        }
      }
    }

    result.sort((a, b) => a.time - b.time);
    return result;
  }

  async function fetchLyrics(song) {
    if (!song || !song.id) return;
    if (lastFetchedSongId === song.id) return;

    lastFetchedSongId = song.id;
    lyricsLoading = true;
    rawLyrics = null;
    parsedLyrics = [];
    activeLyricIndex = -1;

    try {
      const res = await fetch(getApiUrl(`/api/lyrics?title=${encodeURIComponent(song.title)}&artist=${encodeURIComponent(song.artist)}`));
      if (res.ok) {
        const data = await res.json();
        rawLyrics = data;
        if (data.syncedLyrics) {
          parsedLyrics = parseLRC(data.syncedLyrics);
        } else if (data.plainLyrics) {
          // Plain unsynced lyrics
          parsedLyrics = data.plainLyrics.split('\n').filter(l => l.trim()).map(l => ({ time: -1, text: l.trim() }));
        }
      } else {
        parsedLyrics = [];
      }
    } catch (err) {
      console.warn('Lyrics fetch error:', err);
      parsedLyrics = [];
    } finally {
      lyricsLoading = false;
    }
  }

  $: if ($currentSong && isExpanded) {
    fetchLyrics($currentSong);
  }

  // Update active lyrics index as song plays
  $: if (parsedLyrics.length > 0 && parsedLyrics[0].time >= 0) {
    const time = $currentTime;
    let idx = -1;
    for (let i = 0; i < parsedLyrics.length; i++) {
      if (time >= parsedLyrics[i].time - 0.2) {
        idx = i;
      } else {
        break;
      }
    }
    if (idx !== activeLyricIndex) {
      activeLyricIndex = idx;
      // Auto-scroll active lyric smoothly into view if user is not actively scrolling
      if (lyricsContainer && idx >= 0) {
        scrollToActiveLyric(false);
      }
    }
  }

  function formatTime(secs) {
    if (!secs || isNaN(secs) || secs < 0) return '0:00';
    const m = Math.floor(secs / 60);
    const s = Math.floor(secs % 60);
    return `${m}:${s < 10 ? '0' : ''}${s}`;
  }

  function formatRemaining(curr, total) {
    if (!total || isNaN(total) || total <= 0) return '-0:00';
    const remaining = Math.max(0, total - (curr || 0));
    return `-${formatTime(remaining)}`;
  }

  function handleSeek(e) {
    const val = parseFloat(e.target.value);
    seek(val);
  }

  function handleVolume(e) {
    const val = parseFloat(e.target.value);
    isMuted = val === 0;
    setVolume(val);
  }

  function toggleMute() {
    if (isMuted) {
      setVolume(prevVolume || 0.7);
      isMuted = false;
    } else {
      prevVolume = $volume;
      setVolume(0);
      isMuted = true;
    }
  }

  function clearQueue() {
    queue.set($currentSong ? [$currentSong] : []);
    currentIndex.set(0);
  }

  $: progressPercent = $duration > 0 ? Math.min(100, Math.max(0, ($currentTime / $duration) * 100)) : 0;
</script>

{#if $currentSong}
  <!-- 1. APPLE MUSIC FLOATING GLASS CONTROLLER (Docked above bottom navigation) -->
  <div class="fixed bottom-[74px] md:bottom-6 left-3 right-3 max-w-lg md:max-w-2xl mx-auto z-40 anim-dock-in">
    <div
      role="button"
      tabindex="0"
      on:click={() => {
        isExpanded = true;
        isUserScrolling = false;
        setTimeout(() => scrollToActiveLyric(true), 250);
      }}
      on:keydown={(e) => {
        if (e.key === 'Enter') {
          isExpanded = true;
          isUserScrolling = false;
          setTimeout(() => scrollToActiveLyric(true), 250);
        }
      }}
      class="relative overflow-hidden rounded-2xl bg-white/90 dark:bg-[#1c1e28]/90 hover:bg-neutral-50/95 dark:hover:bg-[#232634]/95 active:scale-[0.985] backdrop-blur-3xl border border-black/10 dark:border-white/15 p-2 sm:px-3 shadow-[0_16px_40px_rgba(0,0,0,0.12)] dark:shadow-[0_16px_40px_rgba(0,0,0,0.7)] flex items-center justify-between gap-3 cursor-pointer select-none transition-all duration-300 hover:-translate-y-0.5"
    >
      <!-- Top 2px Progress Line (Apple Music signature) -->
      <div class="absolute top-0 left-0 right-0 h-[2px] bg-black/5 dark:bg-white/10">
        <div
          class="h-full bg-[#fa2d48] transition-all duration-200"
          style="width: {progressPercent}%"
        ></div>
      </div>

      <!-- Left: Squircle Artwork & Song Details -->
      <div class="flex items-center gap-3 min-w-0 flex-1">
        <div class="relative w-11 h-11 rounded-xl overflow-hidden bg-neutral-200 dark:bg-slate-900 flex-shrink-0 shadow-md border border-black/5 dark:border-white/10 group">
          <img
            src={$currentSong.thumbnail}
            alt={$currentSong.title}
            class="w-full h-full object-cover transition-transform duration-500 hover:scale-110"
          />
          {#if $isLoading}
            <div class="absolute inset-0 bg-black/60 flex items-center justify-center">
              <Loader2 class="w-4 h-4 text-white animate-spin" />
            </div>
          {/if}
        </div>
        
        <div class="min-w-0 flex-1">
          <div class="flex items-center gap-1.5">
            <h4 class="text-xs sm:text-sm font-semibold truncate text-neutral-900 dark:text-white">
              {$currentSong.title}
            </h4>
            {#if $currentSong.isTopic}
              <span class="text-[9px] font-bold px-1.5 py-0.2 rounded-md bg-[#fa2d48]/20 text-[#ff7588] border border-[#fa2d48]/30 shrink-0">
                TOPIC
              </span>
            {/if}
          </div>
          <p class="text-[11px] text-neutral-500 dark:text-white/50 truncate mt-0.5">
            {$currentSong.artist}
          </p>
        </div>
      </div>

      <!-- Right: Touch Controls (Apple native style) -->
      <div class="flex items-center gap-1 shrink-0" on:click|stopPropagation on:keydown|stopPropagation role="toolbar" tabindex="0">
        <button
          type="button"
          on:click={togglePlay}
          class="w-9 h-9 rounded-full bg-neutral-900 dark:bg-white text-white dark:text-neutral-900 flex items-center justify-center shadow-md active:scale-90 transition-all btn-pressable cursor-pointer"
          title={$isPlaying ? 'Jeda' : 'Putar'}
        >
          {#if $isPlaying}
            <Pause class="w-4 h-4 fill-current" />
          {:else}
            <Play class="w-4 h-4 fill-current ml-0.5" />
          {/if}
        </button>

        <button
          type="button"
          on:click={playNext}
          class="w-8 h-8 rounded-full flex items-center justify-center text-neutral-600 dark:text-white/80 hover:text-black dark:hover:text-white hover:bg-black/5 dark:hover:bg-white/10 active:scale-90 transition-all btn-pressable cursor-pointer"
          title="Lagu Berikutnya"
        >
          <SkipForward class="w-4 h-4 fill-current" />
        </button>

        <button
          type="button"
          on:click={() => (showQueue = true)}
          class="w-8 h-8 rounded-full flex items-center justify-center text-neutral-600 dark:text-white/80 hover:text-black dark:hover:text-white hover:bg-black/5 dark:hover:bg-white/10 active:scale-90 transition-all btn-pressable cursor-pointer hidden sm:flex"
          title="Antrean"
        >
          <ListMusic class="w-4 h-4" />
        </button>
      </div>
    </div>
  </div>

  <!-- 2. FULL-SCREEN APPLE MUSIC NOW PLAYING SHEET -->
  {#if isExpanded}
    <div
      role="dialog"
      aria-modal="true"
      class="fixed inset-0 z-50 bg-[#090a10]/95 backdrop-blur-3xl flex flex-col justify-between p-4 sm:p-6 md:p-10 {isClosingSheet ? 'anim-sheet-down' : 'anim-sheet-up'} select-none overflow-hidden"
    >
      <!-- Background Ambient Blur derived from album -->
      <div class="absolute inset-0 pointer-events-none -z-10 overflow-hidden opacity-30">
        <img
          src={$currentSong.thumbnail}
          alt=""
          class="w-full h-full object-cover blur-[140px] scale-150"
        />
      </div>

      <!-- Top Header & Grab Handle -->
      <div class="w-full max-w-7xl mx-auto flex items-center justify-between z-20 shrink-0">
        <!-- Close / Minimize Button -->
        <button
          type="button"
          on:click={closeSheet}
          class="w-10 h-10 rounded-full flex items-center justify-center text-white/70 hover:text-white bg-white/10 hover:bg-white/15 active:scale-90 transition-all btn-pressable cursor-pointer"
          title="Tutup"
        >
          <ChevronDown class="w-6 h-6" />
        </button>

        <!-- Center: Grabber on mobile, or Song status on desktop -->
        <div class="flex flex-col items-center">
          <div class="md:hidden w-10 h-1 rounded-full bg-white/25 mb-1.5"></div>
          <span class="text-[11px] font-bold tracking-widest text-white/40 uppercase">
            Sedang Memutar
          </span>
        </div>

        <!-- Right: Queue and More options -->
        <div class="flex items-center gap-2">
          <button
            type="button"
            on:click={() => (showQueue = true)}
            class="w-10 h-10 rounded-full flex items-center justify-center text-white/70 hover:text-white bg-white/10 hover:bg-white/15 active:scale-90 transition-all relative"
            title="Antrean"
          >
            <ListMusic class="w-5 h-5" />
            {#if $queue.length > 1}
              <span class="absolute -top-1 -right-1 w-4 h-4 rounded-full bg-[#fa2d48] text-[9px] text-white flex items-center justify-center font-bold shadow-md">
                {$queue.length}
              </span>
            {/if}
          </button>
        </div>
      </div>

      <!-- MAIN RESPONSIVE CONTAINER: Split 2-column on Desktop (md:), Stack on Mobile (<md:) -->
      <div class="flex-1 w-full max-w-7xl mx-auto flex flex-col md:flex-row items-center justify-between gap-6 md:gap-12 lg:gap-20 my-auto py-2 md:py-6 overflow-hidden min-h-0">
        
        <!-- LEFT COLUMN (On Desktop: Cover + Info + Controls; On Mobile: Cover or Compact Header) -->
        <div class="w-full md:w-[420px] lg:w-[480px] shrink-0 flex flex-col justify-center items-center md:items-start transition-all">
          
          <!-- MOBILE-ONLY COMPACT HEADER (When lyrics view is active on mobile) -->
          {#if showLyrics}
            <div class="md:hidden w-full flex items-center justify-between gap-3 mb-2 px-1">
              <button
                type="button"
                on:click={() => (showLyrics = false)}
                class="flex items-center gap-3 min-w-0 flex-1 text-left cursor-pointer"
                title="Lihat Sampul"
              >
                <img
                  src={$currentSong.thumbnail}
                  alt={$currentSong.title}
                  class="w-12 h-12 rounded-xl object-cover shadow-lg border border-white/10 shrink-0"
                />
                <div class="min-w-0 flex-1">
                  <h3 class="text-base font-bold text-white truncate leading-tight">
                    {$currentSong.title}
                  </h3>
                  <p class="text-xs text-white/60 truncate mt-0.5">
                    {$currentSong.artist}
                  </p>
                </div>
              </button>
              <button
                type="button"
                on:click={() => (showLyrics = false)}
                class="px-3 py-1.5 rounded-full bg-white/10 hover:bg-white/20 text-white/90 text-xs font-semibold shrink-0 cursor-pointer transition-colors"
              >
                Sampul
              </button>
            </div>
          {/if}

          <!-- COVER ARTWORK (Hidden on mobile when lyrics active; Always visible on desktop) -->
          <div class="{showLyrics ? 'hidden md:flex' : 'flex'} flex-col items-center md:items-start w-full mb-4 md:mb-6">
            <div class="relative w-60 h-60 sm:w-72 sm:h-72 md:w-[380px] md:h-[380px] lg:w-[440px] lg:h-[440px] rounded-2xl md:rounded-3xl overflow-hidden shadow-[0_24px_60px_rgba(0,0,0,0.85)] border border-white/15 group">
              <img
                src={$currentSong.thumbnail}
                alt={$currentSong.title}
                class="w-full h-full object-cover transition-transform duration-700 group-hover:scale-105"
              />
              {#if $isLoading}
                <div class="absolute inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center">
                  <Loader2 class="w-10 h-10 text-white animate-spin" />
                </div>
              {/if}
            </div>

            <!-- Mobile quick button to switch back to Lyrics -->
            <button
              type="button"
              on:click={() => {
                showLyrics = true;
                if ($currentSong) {
                  fetchLyrics($currentSong);
                  isUserScrolling = false;
                  setTimeout(() => scrollToActiveLyric(true), 200);
                }
              }}
              class="md:hidden mt-3 inline-flex items-center gap-1.5 px-4 py-1.5 rounded-full bg-white/15 hover:bg-white/25 text-white text-xs font-semibold backdrop-blur-md cursor-pointer transition-colors"
            >
              <MessageSquareQuote class="w-3.5 h-3.5" />
              <span>Buka Lirik Berjalan</span>
            </button>
          </div>

          <!-- DESKTOP TRACK INFO & CONTROLS (Always on left side on desktop) -->
          <div class="hidden md:flex flex-col w-full gap-4 pr-4">
            <div class="flex items-center justify-between">
              <div class="min-w-0 flex-1">
                <h2 class="text-2xl lg:text-3xl font-extrabold text-white truncate tracking-tight">
                  {$currentSong.title}
                </h2>
                <div class="flex items-center gap-2 mt-1">
                  <p class="text-lg text-white/60 truncate font-medium">
                    {$currentSong.artist}
                  </p>
                  {#if $currentSong.isTopic}
                    <span class="text-[10px] font-bold px-2 py-0.5 rounded-full bg-emerald-500/20 text-emerald-300 border border-emerald-500/30">
                      TOPIC
                    </span>
                  {/if}
                </div>
              </div>
            </div>

            <!-- Scrubber -->
            <div class="flex flex-col gap-1.5 w-full">
              <input
                type="range"
                min="0"
                max={$duration || 100}
                step="0.5"
                value={$currentTime}
                on:input={handleSeek}
                style="background: linear-gradient(to right, #ffffff {progressPercent}%, rgba(255, 255, 255, 0.25) {progressPercent}%);"
                class="w-full h-1.5 rounded-lg cursor-pointer appearance-none accent-white transition-all hover:h-2"
              />
              <div class="flex justify-between text-xs font-semibold text-white/40 font-mono">
                <span>{formatTime($currentTime)}</span>
                <span>{formatRemaining($currentTime, $duration)}</span>
              </div>
            </div>

            <!-- Desktop Playback Controls -->
            <div class="flex items-center justify-center gap-8 py-2">
              <button
                type="button"
                on:click={playPrev}
                class="w-12 h-12 rounded-full flex items-center justify-center text-white/80 hover:text-white hover:scale-110 active:scale-95 transition-all"
                title="Sebelumnya"
              >
                <SkipBack class="w-7 h-7 fill-current" />
              </button>

              <button
                type="button"
                on:click={togglePlay}
                disabled={$isLoading}
                class="w-16 h-16 rounded-full bg-white text-slate-950 flex items-center justify-center shadow-[0_10px_25px_rgba(255,255,255,0.2)] hover:scale-105 active:scale-95 transition-all"
                title={$isPlaying ? 'Jeda' : 'Putar'}
              >
                {#if $isPlaying}
                  <Pause class="w-8 h-8 fill-current" />
                {:else}
                  <Play class="w-8 h-8 fill-current ml-1" />
                {/if}
              </button>

              <button
                type="button"
                on:click={playNext}
                class="w-12 h-12 rounded-full flex items-center justify-center text-white/80 hover:text-white hover:scale-110 active:scale-95 transition-all"
                title="Berikutnya"
              >
                <SkipForward class="w-7 h-7 fill-current" />
              </button>
            </div>

            <!-- Desktop Volume Slider -->
            <div class="flex items-center gap-3 px-1 pt-1">
              <button
                type="button"
                on:click={toggleMute}
                class="text-white/40 hover:text-white transition-colors"
              >
                {#if isMuted || $volume === 0}
                  <VolumeX class="w-4 h-4 text-rose-400" />
                {:else}
                  <Volume1 class="w-4 h-4" />
                {/if}
              </button>

              <input
                type="range"
                min="0"
                max="1"
                step="0.01"
                value={$volume}
                on:input={handleVolume}
                style="background: linear-gradient(to right, #ffffff {$volume * 100}%, rgba(255, 255, 255, 0.25) {$volume * 100}%);"
                class="w-full h-1.5 rounded-lg cursor-pointer appearance-none accent-white"
              />

              <Volume2 class="w-4 h-4 text-white/40" />
            </div>
          </div>
        </div>

        <!-- RIGHT COLUMN ON DESKTOP / CENTER AREA ON MOBILE (Live Lyrics Stream) -->
        <div class="{showLyrics ? 'flex' : 'hidden md:flex'} flex-1 w-full h-full flex-col justify-center overflow-hidden relative md:pl-6">
          <div
            bind:this={lyricsContainer}
            on:scroll={handleUserScroll}
            on:wheel={handleUserScroll}
            on:touchstart={handleUserScroll}
            class="h-[42vh] sm:h-[48vh] md:h-[68vh] overflow-y-auto px-4 md:px-8 py-10 space-y-6 md:space-y-10 scroll-smooth no-scrollbar"
            style="-webkit-mask-image: linear-gradient(to bottom, transparent 0%, black 12%, black 88%, transparent 100%); mask-image: linear-gradient(to bottom, transparent 0%, black 12%, black 88%, transparent 100%);"
          >
            {#if lyricsLoading}
              <div class="h-full flex flex-col items-center justify-center gap-3 text-white/40 py-20">
                <Loader2 class="w-8 h-8 animate-spin text-[#fa2d48]" />
                <span class="text-sm font-medium">Memuat lirik dari LRCLIB...</span>
              </div>
            {:else if parsedLyrics.length === 0}
              <div class="h-full flex flex-col items-center justify-center gap-3 text-white/40 text-center px-4 py-20">
                <MessageSquareQuote class="w-12 h-12 stroke-[1.5] text-white/20 mb-1" />
                <p class="text-lg font-bold text-white/80">Lirik Tidak Tersedia</p>
                <p class="text-sm text-white/40 max-w-sm">Lagu ini belum memiliki lirik di database gratis LRCLIB.</p>
              </div>
            {:else}
              {#each parsedLyrics as line, idx}
                {@const isActive = idx === activeLyricIndex}
                {@const isPast = idx < activeLyricIndex}
                <div
                  data-lyric-index={idx}
                  on:click={() => {
                    if (line.time >= 0) {
                      seek(line.time);
                      activeLyricIndex = idx;
                      isUserScrolling = false;
                      setTimeout(() => scrollToActiveLyric(true), 50);
                    }
                  }}
                  on:keydown={(e) => {
                    if (e.key === 'Enter' && line.time >= 0) {
                      seek(line.time);
                      activeLyricIndex = idx;
                      isUserScrolling = false;
                      setTimeout(() => scrollToActiveLyric(true), 50);
                    }
                  }}
                  tabindex="0"
                  role="button"
                  class="cursor-pointer transition-all duration-300 text-left outline-none select-none {isActive ? 'scale-[1.02] origin-left' : 'hover:text-white/70'}"
                >
                  <p
                    class="font-black tracking-tight leading-snug transition-all duration-300 {isActive
                      ? 'text-2xl sm:text-3xl md:text-4xl lg:text-5xl text-white drop-shadow-[0_4px_24px_rgba(255,255,255,0.4)]'
                      : isPast
                      ? 'text-lg sm:text-xl md:text-2xl lg:text-3xl text-white/25 blur-[0.6px]'
                      : 'text-lg sm:text-xl md:text-2xl lg:text-3xl text-white/35'}"
                  >
                    {line.text}
                  </p>
                </div>
              {/each}
            {/if}
          </div>
        </div>
      </div>

      <!-- MOBILE-ONLY BOTTOM CONTROLS (Shown only on mobile <md) -->
      <div class="md:hidden w-full max-w-md mx-auto flex flex-col gap-3.5 pt-2 z-20 shrink-0">
        <!-- Title & Artist Info on mobile (Only when NOT in lyrics mode, because lyrics mode has mini top header) -->
        {#if !showLyrics}
          <div class="flex items-center justify-between px-1">
            <div class="min-w-0 flex-1 pr-2">
              <h2 class="text-xl font-bold text-white truncate tracking-tight">
                {$currentSong.title}
              </h2>
              <div class="flex items-center gap-2 mt-0.5">
                <p class="text-sm text-white/60 truncate font-medium">
                  {$currentSong.artist}
                </p>
                {#if $currentSong.isTopic}
                  <span class="text-[9px] font-bold px-1.5 py-0.5 rounded-full bg-emerald-500/20 text-emerald-300 border border-emerald-500/30">
                    TOPIC
                  </span>
                {/if}
              </div>
            </div>
          </div>
        {/if}

        <!-- Apple Music Live Scrubber -->
        <div class="flex flex-col gap-1 px-1">
          <input
            type="range"
            min="0"
            max={$duration || 100}
            step="0.5"
            value={$currentTime}
            on:input={handleSeek}
            style="background: linear-gradient(to right, #ffffff {progressPercent}%, rgba(255, 255, 255, 0.25) {progressPercent}%);"
            class="w-full h-1 rounded-lg cursor-pointer appearance-none accent-white transition-all"
          />
          <div class="flex justify-between text-[11px] font-semibold text-white/40 font-mono">
            <span>{formatTime($currentTime)}</span>
            <span>{formatRemaining($currentTime, $duration)}</span>
          </div>
        </div>

        <!-- Large Apple Mobile Playback Controls -->
        <div class="flex items-center justify-center gap-8 py-1">
          <button
            type="button"
            on:click={playPrev}
            class="w-12 h-12 rounded-full flex items-center justify-center text-white/80 active:scale-90 transition-all"
            title="Sebelumnya"
          >
            <SkipBack class="w-7 h-7 fill-current" />
          </button>

          <button
            type="button"
            on:click={togglePlay}
            disabled={$isLoading}
            class="w-16 h-16 rounded-full bg-white text-slate-950 flex items-center justify-center shadow-xl active:scale-90 transition-all"
            title={$isPlaying ? 'Jeda' : 'Putar'}
          >
            {#if $isPlaying}
              <Pause class="w-8 h-8 fill-current" />
            {:else}
              <Play class="w-8 h-8 fill-current ml-1" />
            {/if}
          </button>

          <button
            type="button"
            on:click={playNext}
            class="w-12 h-12 rounded-full flex items-center justify-center text-white/80 active:scale-90 transition-all"
            title="Berikutnya"
          >
            <SkipForward class="w-7 h-7 fill-current" />
          </button>
        </div>

        <!-- Mobile Volume & Utility Pill Bar (Apple Music iOS Signature) -->
        <div class="flex items-center justify-between gap-3 px-2 pt-1">
          <div class="flex items-center gap-2.5 flex-1 max-w-[200px]">
            <button
              type="button"
              on:click={toggleMute}
              class="text-white/40 hover:text-white transition-colors"
            >
              {#if isMuted || $volume === 0}
                <VolumeX class="w-3.5 h-3.5 text-rose-400" />
              {:else}
                <Volume1 class="w-3.5 h-3.5" />
              {/if}
            </button>
            <input
              type="range"
              min="0"
              max="1"
              step="0.01"
              value={$volume}
              on:input={handleVolume}
              style="background: linear-gradient(to right, #ffffff {$volume * 100}%, rgba(255, 255, 255, 0.25) {$volume * 100}%);"
              class="w-full h-1 rounded-lg cursor-pointer appearance-none accent-white"
            />
            <Volume2 class="w-3.5 h-3.5 text-white/40" />
          </div>

          <!-- iOS Bottom Utility Pills: Lyrics & Queue -->
          <div class="flex items-center gap-2">
            <button
              type="button"
              on:click={() => {
                showLyrics = !showLyrics;
                if (showLyrics && $currentSong) {
                  fetchLyrics($currentSong);
                  isUserScrolling = false;
                  setTimeout(() => scrollToActiveLyric(true), 200);
                }
              }}
              class="p-2 rounded-full flex items-center justify-center active:scale-90 transition-all {showLyrics ? 'bg-white text-black shadow-md' : 'text-white/60 bg-white/10'}"
              title={showLyrics ? 'Tutup Lirik' : 'Buka Lirik'}
            >
              <MessageSquareQuote class="w-4 h-4" />
            </button>

            <button
              type="button"
              on:click={() => (showQueue = true)}
              class="p-2 rounded-full flex items-center justify-center text-white/60 bg-white/10 active:scale-90 transition-all"
              title="Daftar Antrean"
            >
              <ListMusic class="w-4 h-4" />
            </button>
          </div>
        </div>
      </div>
    </div>
  {/if}

  <!-- 3. QUEUE DRAWER (Bottom Sheet) -->
  {#if showQueue}
    <div
      role="presentation"
      class="fixed inset-0 bg-black/70 backdrop-blur-md z-[60] flex items-end sm:items-center justify-center p-0 sm:p-4 transition-all duration-300 {isClosingQueue ? 'anim-backdrop-out' : ''}"
      on:click={closeQueue}
      on:keydown={(e) => e.key === 'Escape' && closeQueue()}
    >
      <div
        role="dialog"
        aria-modal="true"
        tabindex="-1"
        class="bg-[#12141e]/95 backdrop-blur-2xl border border-white/15 w-full sm:max-w-lg max-h-[80vh] rounded-t-3xl sm:rounded-3xl p-5 flex flex-col gap-4 shadow-2xl overflow-hidden {isClosingQueue ? 'anim-modal-out' : 'anim-modal-in'}"
        on:click|stopPropagation
        on:keydown|stopPropagation
      >
        <!-- Queue Header -->
        <div class="flex items-center justify-between border-b border-white/10 pb-3">
          <div class="flex items-center gap-2">
            <ListMusic class="w-5 h-5 text-indigo-400" />
            <h3 class="font-bold text-lg text-white">Antrean Pemutaran</h3>
            <span class="text-xs text-white/40">({$queue.length} lagu)</span>
          </div>
          <div class="flex items-center gap-2">
            {#if $queue.length > 1}
              <button
                type="button"
                on:click={clearQueue}
                class="p-2 text-white/40 hover:text-rose-400 active:scale-90 transition-all btn-pressable cursor-pointer"
                title="Bersihkan antrean"
              >
                <Trash2 class="w-4 h-4" />
              </button>
            {/if}
            <button
              type="button"
              on:click={closeQueue}
              class="p-2 text-white/40 hover:text-white active:scale-90 transition-all btn-pressable cursor-pointer"
            >
              <X class="w-5 h-5" />
            </button>
          </div>
        </div>

        <!-- Queue List -->
        <div class="flex-1 overflow-y-auto flex flex-col gap-1 pr-1 divide-y divide-white/[0.04]">
          {#each $queue as qSong, idx (qSong.id + idx)}
            {@const isNow = $currentSong?.id === qSong.id}
            <button
              type="button"
              on:click={() => playSong(qSong)}
              class="w-full text-left p-2.5 rounded-2xl flex items-center gap-3 cursor-pointer transition-all {isNow ? 'bg-white/10 text-indigo-300' : 'hover:bg-white/5 text-white'}"
            >
              <img
                src={qSong.thumbnail}
                alt={qSong.title}
                class="w-11 h-11 rounded-xl object-cover border border-white/10"
              />
              <div class="flex-1 min-w-0">
                <h5 class="text-sm font-semibold truncate {isNow ? 'text-indigo-400' : 'text-white'}">
                  {qSong.title}
                </h5>
                <p class="text-xs text-white/40 truncate mt-0.5">{qSong.artist}</p>
              </div>
              {#if isNow && $isPlaying}
                <div class="flex items-end gap-[2px] h-4">
                  <span class="w-[3px] bg-indigo-400 animate-bounce h-2 rounded-full"></span>
                  <span class="w-[3px] bg-indigo-400 animate-bounce h-4 rounded-full"></span>
                  <span class="w-[3px] bg-indigo-400 animate-bounce h-3 rounded-full"></span>
                </div>
              {/if}
            </button>
          {/each}
        </div>
      </div>
    </div>
  {/if}
{/if}
