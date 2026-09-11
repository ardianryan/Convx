<script>
  import { Search, Loader2, X } from 'lucide-svelte';
  import { createEventDispatcher } from 'svelte';

  export let query = '';
  export let loading = false;

  const dispatch = createEventDispatcher();

  function handleSubmit(e) {
    e.preventDefault();
    if (query.trim()) {
      dispatch('search', query.trim());
    }
  }

  function handleClear() {
    query = '';
    dispatch('clear');
  }
</script>

<form on:submit={handleSubmit} class="w-full relative group">
  <div class="relative flex items-center">
    <!-- Search Icon / Spinner -->
    <div class="absolute left-3.5 text-neutral-400 dark:text-white/40 group-focus-within:text-neutral-900 dark:group-focus-within:text-white transition-colors pointer-events-none flex items-center">
      {#if loading}
        <Loader2 class="w-4 h-4 animate-spin text-[#fa2d48]" />
      {:else}
        <Search class="w-4 h-4" />
      {/if}
    </div>
    
    <!-- Apple Pill Input Field -->
    <input
      type="text"
      bind:value={query}
      placeholder="Apple Music atau Lagu, Artis..."
      class="w-full pl-10 pr-24 py-2.5 rounded-full bg-black/[0.05] dark:bg-white/[0.08] hover:bg-black/[0.08] dark:hover:bg-white/[0.12] focus:bg-white dark:focus:bg-white/[0.14] backdrop-blur-xl border border-black/10 dark:border-white/10 text-neutral-900 dark:text-white placeholder-neutral-400 dark:placeholder-white/35 text-xs sm:text-sm font-medium focus:outline-none focus:ring-2 focus:ring-[#fa2d48]/40 focus:border-[#fa2d48]/60 transition-all shadow-inner"
    />

    <!-- Action Group (Clear & Search) -->
    <div class="absolute right-1.5 flex items-center gap-1.5">
      {#if query}
        <button
          type="button"
          on:click={handleClear}
          class="w-5 h-5 rounded-full bg-black/10 dark:bg-white/20 hover:bg-black/20 dark:hover:bg-white/30 text-neutral-700 dark:text-white flex items-center justify-center transition-all active:scale-90"
          title="Hapus"
        >
          <X class="w-3 h-3" />
        </button>
      {/if}

      <button
        type="submit"
        disabled={loading || !query.trim()}
        class="px-3.5 py-1 rounded-full bg-[#fa2d48] text-white hover:bg-[#ff3b56] font-semibold text-xs transition-all disabled:opacity-40 disabled:cursor-not-allowed shadow-md shadow-[#fa2d48]/25 active:scale-95"
      >
        Cari
      </button>
    </div>
  </div>
</form>
