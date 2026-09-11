<script>
  import { Sun, Moon, SunMoon } from 'lucide-svelte';
  import { themePreference, cycleTheme, isDarkActive } from '../stores/theme.js';

  let isAnimating = false;

  function handleClick() {
    isAnimating = true;
    cycleTheme();
    setTimeout(() => {
      isAnimating = false;
    }, 300);
  }

  $: tooltipText = 
    $themePreference === 'auto'
      ? 'Tema: Otomatis Sistem (Klik untuk Terang)'
      : $themePreference === 'light'
      ? 'Tema: Mode Terang (Klik untuk Gelap)'
      : 'Tema: Mode Gelap (Klik untuk Otomatis Sistem)';
</script>

<button
  type="button"
  on:click={handleClick}
  class="w-8 h-8 rounded-full bg-black/5 dark:bg-white/5 hover:bg-black/10 dark:hover:bg-white/10 text-neutral-700 dark:text-neutral-300 hover:text-neutral-900 dark:hover:text-white flex items-center justify-center transition-all border border-black/10 dark:border-white/10 active:scale-90 cursor-pointer relative group"
  title={tooltipText}
  aria-label={tooltipText}
>
  <div class="transition-transform duration-300 {isAnimating ? 'rotate-90 scale-75' : 'rotate-0 scale-100'}">
    {#if $themePreference === 'auto'}
      <SunMoon class="w-4 h-4 text-amber-500 dark:text-amber-400" />
    {:else if $themePreference === 'light'}
      <Sun class="w-4 h-4 text-amber-500" />
    {:else}
      <Moon class="w-4 h-4 text-sky-400" />
    {/if}
  </div>

  <!-- Subtle indicator dot for auto/system -->
  {#if $themePreference === 'auto'}
    <span class="absolute bottom-0.5 right-0.5 w-1.5 h-1.5 rounded-full bg-emerald-500 border border-white dark:border-black"></span>
  {/if}
</button>
