<script>
  import { Play, Pause, Plus, Music } from 'lucide-svelte';
  import { currentSong, isPlaying, playSong, addToQueue, togglePlay } from '../stores/player.js';

  export let tracks = [];
  export let title = 'Lagu Pilihan';

  function handlePlay(track) {
    if ($currentSong?.id === track.id) {
      togglePlay();
    } else {
      playSong(track, tracks);
    }
  }

  function formatTime(seconds) {
    if (!seconds) return '--:--';
    const m = Math.floor(seconds / 60);
    const s = Math.floor(seconds % 60);
    return `${m}:${s < 10 ? '0' : ''}${s}`;
  }
</script>

<div class="w-full">
  {#if title}
    <div class="flex items-center justify-between px-1 mb-2.5">
      <h2 class="text-base sm:text-lg font-bold text-neutral-900 dark:text-white tracking-tight flex items-center gap-1.5">
        <span>{title}</span>
      </h2>
      <span class="text-xs text-neutral-500 dark:text-white/40 font-medium">{tracks.length} lagu</span>
    </div>
  {/if}

  {#if tracks.length === 0}
    <div class="py-16 text-center text-neutral-400 dark:text-white/30 font-medium text-sm">
      Tidak ada lagu untuk ditampilkan.
    </div>
  {:else}
    <div class="flex flex-col rounded-2xl bg-white/70 dark:bg-white/[0.03] backdrop-blur-xl border border-black/[0.06] dark:border-white/[0.08] overflow-hidden divide-y divide-black/[0.04] dark:divide-white/[0.05] shadow-lg">
      {#each tracks as track, idx (track.id || idx)}
        {@const isCurrent = $currentSong?.id === track.id}
        <div
          role="button"
          tabindex="0"
          on:click={() => handlePlay(track)}
          on:keydown={(e) => e.key === 'Enter' && handlePlay(track)}
          class="flex items-center justify-between gap-3 px-3 py-2.5 sm:px-4 sm:py-3 hover:bg-black/[0.03] dark:hover:bg-white/[0.07] active:bg-black/[0.06] dark:active:bg-white/[0.1] transition-all cursor-pointer select-none group relative {isCurrent ? 'bg-black/[0.04] dark:bg-white/[0.08]' : ''}"
        >
          <!-- Left: Artwork & Playing Equalizer -->
          <div class="relative w-11 h-11 rounded-lg overflow-hidden flex-shrink-0 bg-[#161822] shadow-sm border border-black/10 dark:border-white/10">
            <img
              src={track.thumbnail}
              alt={track.title}
              class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
              loading="lazy"
            />
            
            <!-- Playing Animated Equalizer Overlay -->
            {#if isCurrent}
              <div class="absolute inset-0 bg-black/65 flex items-center justify-center gap-[2px]">
                {#if $isPlaying}
                  <span class="w-[2.5px] h-3 bg-[#fa2d48] rounded-full animate-[bounce_0.8s_ease-in-out_infinite]"></span>
                  <span class="w-[2.5px] h-4 bg-[#ff6480] rounded-full animate-[bounce_1.1s_ease-in-out_infinite_0.2s]"></span>
                  <span class="w-[2.5px] h-2.5 bg-[#fa2d48] rounded-full animate-[bounce_0.9s_ease-in-out_infinite_0.4s]"></span>
                {:else}
                  <Pause class="w-4 h-4 text-white fill-white" />
                {/if}
              </div>
            {/if}
          </div>

          <!-- Center: Song Info -->
          <div class="flex-1 min-w-0 pr-2">
            <h4 class="text-xs sm:text-sm font-semibold truncate transition-colors {isCurrent ? 'text-[#fa2d48]' : 'text-neutral-900 dark:text-white'}">
              {track.title}
            </h4>
            
            <div class="flex items-center gap-1.5 mt-0.5">
              <p class="text-[11px] sm:text-xs text-neutral-500 dark:text-white/50 truncate font-normal">
                {track.artist}
              </p>
              {#if track.isTopic}
                <span class="text-[9px] font-bold px-1.5 py-0.2 rounded-md bg-[#fa2d48]/20 text-[#ff7588] border border-[#fa2d48]/30 shrink-0">
                  TOPIC
                </span>
              {/if}
              {#if track.durationText || track.duration}
                <span class="text-[11px] text-neutral-400 dark:text-white/30 hidden sm:inline">
                  • {track.durationText || formatTime(track.duration)}
                </span>
              {/if}
            </div>
          </div>

          <!-- Right: Duration & Quick Actions -->
          <div class="flex items-center gap-2 shrink-0">
            {#if track.durationText || track.duration}
              <span class="text-xs text-neutral-500 dark:text-white/40 font-mono sm:hidden">
                {track.durationText || formatTime(track.duration)}
              </span>
              <span class="text-xs text-neutral-400 dark:text-white/35 font-mono hidden sm:inline">
                {track.durationText || formatTime(track.duration)}
              </span>
            {/if}

            <button
              type="button"
              on:click|stopPropagation={() => queue.add(track)}
              class="w-7 h-7 rounded-full flex items-center justify-center text-neutral-400 dark:text-white/40 hover:text-neutral-900 dark:hover:text-white hover:bg-black/5 dark:hover:bg-white/10 transition-colors"
              title="Tambahkan ke Antrean"
            >
              <Plus class="w-4 h-4" />
            </button>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>
