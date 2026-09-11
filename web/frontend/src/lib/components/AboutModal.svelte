<script>
  import {
    X,
    Sparkles,
    ExternalLink,
    Music,
    Code2,
    Heart
  } from 'lucide-svelte';

  export let isOpen = false;
  export let onClose = () => {};

  let isClosing = false;

  function handleClose() {
    if (isClosing) return;
    isClosing = true;
    setTimeout(() => {
      onClose();
      isClosing = false;
    }, 220);
  }
</script>

{#if isOpen}
  <div
    role="presentation"
    on:click|self={handleClose}
    class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-md transition-all duration-300 {isClosing ? 'anim-backdrop-out' : ''}"
  >
    <div
      class="w-full max-w-lg bg-[#181a24]/95 backdrop-blur-2xl border border-white/15 rounded-3xl shadow-2xl overflow-hidden text-white flex flex-col max-h-[90vh] {isClosing ? 'anim-modal-out' : 'anim-modal-in'}"
    >
      <!-- Header -->
      <div class="p-5 border-b border-white/10 flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-2xl bg-gradient-to-tr from-rose-500 to-amber-500 flex items-center justify-center shadow-lg shadow-rose-500/25 text-white">
            <Sparkles class="w-5 h-5" />
          </div>
          <div>
            <h2 class="text-base sm:text-lg font-bold">Tentang Aplikasi Convx Web</h2>
            <p class="text-xs text-neutral-400">Porting Web Modern & Open Source</p>
          </div>
        </div>
        <button
          type="button"
          on:click={handleClose}
          class="w-8 h-8 rounded-full bg-white/5 hover:bg-white/10 flex items-center justify-center text-neutral-400 hover:text-white transition-colors btn-pressable cursor-pointer"
          title="Tutup"
        >
          <X class="w-4 h-4" />
        </button>
      </div>

      <!-- Body -->
      <div class="p-6 overflow-y-auto space-y-5 text-xs sm:text-sm text-neutral-300 leading-relaxed">
        
        <!-- Story / Explanation Card -->
        <div class="p-4 rounded-2xl bg-white/[0.04] border border-white/[0.08] space-y-2.5">
          <div class="flex items-center gap-2 text-[#fa2d48] font-bold text-xs uppercase tracking-wider">
            <Music class="w-4 h-4" />
            <span>Porting Resmi Versi Web</span>
          </div>
          <p class="text-neutral-200">
            Aplikasi web ini merupakan hasil <b>porting Convx</b> (yang awalnya merupakan pemutar musik native Android berbasis <i>Liquid Glass</i> dan <i>Jetpack Compose</i>) ke dalam arsitektur <b>Web Fullstack Modern</b>.
          </p>
          <p class="text-neutral-400 text-xs leading-normal">
            Kini Anda dapat menikmati katalog streaming musik, sinkronisasi lirik berjalan karaoke (LRCLIB), relay audio Cloudflare Workers anti-blokir, serta kontrol multi-perangkat langsung melalui browser komputer maupun ponsel tanpa instalasi aplikasi.
          </p>
        </div>

        <!-- Tech Stack Highlight -->
        <div class="space-y-2">
          <h4 class="text-xs font-semibold text-neutral-400 uppercase tracking-wider">Teknologi yang Digunakan</h4>
          <div class="grid grid-cols-2 gap-2.5">
            <div class="p-3 rounded-xl bg-white/[0.03] border border-white/[0.06] flex items-center gap-2.5">
              <div class="w-7 h-7 rounded-lg bg-orange-500/20 text-orange-400 flex items-center justify-center font-mono font-bold text-xs">S5</div>
              <div>
                <p class="font-semibold text-xs text-white">Svelte 5 & Tailwind</p>
                <p class="text-[11px] text-neutral-400">Apple Glass UI & Motion</p>
              </div>
            </div>

            <div class="p-3 rounded-xl bg-white/[0.03] border border-white/[0.06] flex items-center gap-2.5">
              <div class="w-7 h-7 rounded-lg bg-cyan-500/20 text-cyan-400 flex items-center justify-center font-mono font-bold text-xs">GO</div>
              <div>
                <p class="font-semibold text-xs text-white">Golang Engine</p>
                <p class="text-[11px] text-neutral-400">Innertube & Stream Proxy</p>
              </div>
            </div>

            <div class="p-3 rounded-xl bg-white/[0.03] border border-white/[0.06] flex items-center gap-2.5">
              <div class="w-7 h-7 rounded-lg bg-amber-500/20 text-amber-400 flex items-center justify-center font-mono font-bold text-xs">CF</div>
              <div>
                <p class="font-semibold text-xs text-white">Cloudflare Relay</p>
                <p class="text-[11px] text-neutral-400">Edge Bypass & Global CDN</p>
              </div>
            </div>

            <div class="p-3 rounded-xl bg-white/[0.03] border border-white/[0.06] flex items-center gap-2.5">
              <div class="w-7 h-7 rounded-lg bg-blue-500/20 text-blue-400 flex items-center justify-center font-mono font-bold text-xs">SQL</div>
              <div>
                <p class="font-semibold text-xs text-white">SQLite WAL + Drizzle</p>
                <p class="text-[11px] text-neutral-400">Single File Autonomous DB</p>
              </div>
            </div>
          </div>
        </div>

        <!-- Links to Original Repo -->
        <div class="space-y-2 pt-1">
          <h4 class="text-xs font-semibold text-neutral-400 uppercase tracking-wider">Repository Asli & Sumber Kode</h4>
          <div class="space-y-2">
            <a
              href="https://github.com/cosmictaserdev-creator/Convx"
              target="_blank"
              rel="noopener noreferrer"
              class="flex items-center justify-between p-3.5 rounded-2xl bg-white/[0.05] hover:bg-white/[0.09] border border-white/10 transition-all group btn-pressable"
            >
              <div class="flex items-center gap-3 min-w-0">
                <svg class="w-5 h-5 fill-current text-white shrink-0" viewBox="0 0 24 24">
                  <path fill-rule="evenodd" clip-rule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.53 1.032 1.53 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z" />
                </svg>
                <div class="min-w-0">
                  <p class="font-semibold text-xs text-white group-hover:text-rose-400 transition-colors truncate">
                    cosmictaserdev-creator / Convx
                  </p>
                  <p class="text-[11px] text-neutral-400 truncate">
                    Repositori asli Android (Liquid Glass Music Player)
                  </p>
                </div>
              </div>
              <ExternalLink class="w-4 h-4 text-neutral-400 group-hover:text-white shrink-0 ml-2" />
            </a>

            <a
              href="https://github.com/ardianryan/Convx-Desktop"
              target="_blank"
              rel="noopener noreferrer"
              class="flex items-center justify-between p-3.5 rounded-2xl bg-white/[0.05] hover:bg-white/[0.09] border border-white/10 transition-all group btn-pressable"
            >
              <div class="flex items-center gap-3 min-w-0">
                <Code2 class="w-5 h-5 text-[#fa2d48] shrink-0" />
                <div class="min-w-0">
                  <p class="font-semibold text-xs text-white group-hover:text-rose-400 transition-colors truncate">
                    ardianryan / Convx-Desktop
                  </p>
                  <p class="text-[11px] text-neutral-400 truncate">
                    Repositori porting versi website lengkap + Docker
                  </p>
                </div>
              </div>
              <ExternalLink class="w-4 h-4 text-neutral-400 group-hover:text-white shrink-0 ml-2" />
            </a>
          </div>
        </div>

      </div>

      <!-- Footer -->
      <div class="p-4 border-t border-white/10 bg-white/[0.02] flex items-center justify-between text-xs text-neutral-500">
        <div class="flex items-center gap-1.5">
          <span>Dibuat dengan</span>
          <Heart class="w-3.5 h-3.5 text-rose-500 fill-rose-500" />
          <span>untuk pecinta musik</span>
        </div>
        <button
          type="button"
          on:click={handleClose}
          class="px-4 py-1.5 rounded-xl bg-white/10 hover:bg-white/15 text-white font-semibold transition-colors btn-pressable cursor-pointer"
        >
          Selesai
        </button>
      </div>

    </div>
  </div>
{/if}
