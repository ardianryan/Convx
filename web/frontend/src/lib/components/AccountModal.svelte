<script>
  import { onMount } from 'svelte';
  import {
    X,
    User,
    ShieldCheck,
    Key,
    Check,
    Copy,
    ExternalLink,
    AlertTriangle,
    Trash2,
    LogIn,
    Loader2
  } from 'lucide-svelte';

  export let isOpen = false;
  export let onClose = () => {};

  let isLoggedIn = false;
  let cookieInput = '';
  let isLoading = false;
  let errorMsg = '';
  let successMsg = '';
  let copiedScript = false;

  const quickScript = `copy(document.cookie); alert("Cookie YouTube berhasil disalin ke clipboard!");`;

  onMount(() => {
    checkStatus();
  });

  $: if (isOpen) {
    checkStatus();
    errorMsg = '';
    successMsg = '';
  }

  async function checkStatus() {
    try {
      const res = await fetch('/api/account/status');
      if (res.ok) {
        const data = await res.json();
        isLoggedIn = !!data.isLoggedIn;
      }
    } catch (e) {
      console.warn('Failed to check account status:', e);
    }
  }

  async function saveCookie() {
    const trimmed = cookieInput.trim();
    if (!trimmed) {
      errorMsg = 'Silakan masukkan cookie terlebih dahulu.';
      return;
    }

    isLoading = true;
    errorMsg = '';
    successMsg = '';

    try {
      const res = await fetch('/api/account/cookie', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ cookie: trimmed }),
      });

      if (res.ok) {
        isLoggedIn = true;
        cookieInput = '';
        successMsg = 'Cookie YouTube berhasil disimpan! Mode Audio kini bebas batas 1MB.';
        setTimeout(() => {
          checkStatus();
        }, 500);
      } else {
        const errData = await res.json().catch(() => ({}));
        errorMsg = errData.error || 'Gagal menyimpan cookie.';
      }
    } catch (e) {
      errorMsg = 'Terjadi kesalahan jaringan: ' + e.message;
    } finally {
      isLoading = false;
    }
  }

  async function logout() {
    isLoading = true;
    errorMsg = '';
    successMsg = '';

    try {
      const res = await fetch('/api/account/logout', { method: 'POST' });
      if (res.ok) {
        isLoggedIn = false;
        successMsg = 'Cookie telah dihapus. Kembali ke Mode Tamu.';
      }
    } catch (e) {
      errorMsg = 'Gagal logout: ' + e.message;
    } finally {
      isLoading = false;
    }
  }

  function copyQuickScript() {
    if (navigator.clipboard) {
      navigator.clipboard.writeText(quickScript);
      copiedScript = true;
      setTimeout(() => (copiedScript = false), 2500);
    }
  }
</script>

{#if isOpen}
  <!-- Backdrop -->
  <div
    class="fixed inset-0 z-50 bg-black/70 backdrop-blur-md flex items-center justify-center p-4 transition-all duration-300"
    on:click|self={onClose}
    on:keydown={(e) => e.key === 'Escape' && onClose()}
    tabindex="-1"
    role="dialog"
  >
    <!-- Modal Card -->
    <div
      class="glass-panel border border-white/15 rounded-3xl w-full max-w-lg p-6 md:p-8 flex flex-col gap-5 shadow-2xl relative anim-modal-in text-slate-200"
    >
      <!-- Close Button -->
      <button
        class="absolute top-5 right-5 p-2 rounded-xl text-slate-400 hover:text-white hover:bg-white/10 transition-all"
        on:click={onClose}
        aria-label="Tutup"
      >
        <X class="w-5 h-5" />
      </button>

      <!-- Header -->
      <div class="flex items-center gap-3">
        <div class="w-12 h-12 rounded-2xl bg-gradient-to-tr from-indigo-500 to-purple-600 flex items-center justify-center shadow-lg shadow-indigo-500/30">
          <Key class="w-6 h-6 text-white" />
        </div>
        <div>
          <h3 class="text-lg font-bold text-white tracking-tight">
            Akun & Cookie YouTube
          </h3>
          <p class="text-xs text-slate-400">
            Lewati pembatasan bot YouTube agar streaming audio lancar tanpa henti
          </p>
        </div>
      </div>

      <!-- Status Badge -->
      <div class="p-3.5 rounded-2xl border {isLoggedIn ? 'bg-emerald-500/10 border-emerald-500/30' : 'bg-amber-500/10 border-amber-500/30'} flex items-start gap-3">
        {#if isLoggedIn}
          <ShieldCheck class="w-5 h-5 text-emerald-400 flex-shrink-0 mt-0.5" />
          <div class="text-xs">
            <div class="font-semibold text-emerald-300 flex items-center gap-1.5">
              <span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
              Terhubung ke YouTube (Login Aktif)
            </div>
            <p class="text-slate-300 mt-1">
              Cookie sesi tersimpan. Server memanggil YouTube sebagai pengguna terautentikasi sehingga bebas batas bot 1MB.
            </p>
          </div>
        {:else}
          <AlertTriangle class="w-5 h-5 text-amber-400 flex-shrink-0 mt-0.5" />
          <div class="text-xs">
            <div class="font-semibold text-amber-300 flex items-center gap-1.5">
              <span class="w-2 h-2 rounded-full bg-amber-400"></span>
              Mode Tamu (Belum Terhubung)
            </div>
            <p class="text-slate-300 mt-1">
              Streaming tanpa login dibatasi oleh Google Video hingga ~1 MB per lagu. Masukkan cookie akun Anda di bawah untuk mengaktifkan pemutaran penuh.
            </p>
          </div>
        {/if}
      </div>

      <!-- Alerts -->
      {#if errorMsg}
        <div class="px-3.5 py-2.5 rounded-xl bg-rose-500/20 border border-rose-500/30 text-rose-300 text-xs">
          {errorMsg}
        </div>
      {/if}

      {#if successMsg}
        <div class="px-3.5 py-2.5 rounded-xl bg-emerald-500/20 border border-emerald-500/30 text-emerald-300 text-xs">
          {successMsg}
        </div>
      {/if}

      <!-- Form Input Cookie -->
      <div class="flex flex-col gap-2">
        <label for="cookie-input" class="text-xs font-semibold text-slate-300 flex items-center justify-between">
          <span>Paste Cookie YouTube:</span>
          <button
            type="button"
            class="text-[11px] text-indigo-400 hover:text-indigo-300 flex items-center gap-1 transition-colors"
            on:click={copyQuickScript}
          >
            {#if copiedScript}
              <Check class="w-3 h-3 text-emerald-400" />
              <span class="text-emerald-300">Script Disalin!</span>
            {:else}
              <Copy class="w-3 h-3" />
              <span>Salin Script 1-Klik</span>
            {/if}
          </button>
        </label>
        <textarea
          id="cookie-input"
          bind:value={cookieInput}
          rows="3"
          placeholder="Paste string cookie di sini (misal: VISITOR_INFO1_LIVE=...; SAPISID=...; __Secure-3PAPISID=...)"
          class="w-full px-3.5 py-2.5 rounded-2xl bg-black/40 border border-white/10 text-white placeholder-slate-500 text-xs font-mono focus:outline-none focus:border-indigo-500/60 focus:ring-1 focus:ring-indigo-500/30 transition-all resize-none"
        ></textarea>
      </div>

      <!-- Panduan Singkat -->
      <div class="rounded-2xl bg-white/[0.03] border border-white/5 p-3 text-[11px] text-slate-400 flex flex-col gap-1.5">
        <div class="font-semibold text-slate-300">Cara Mendapatkan Cookie:</div>
        <ol class="list-decimal list-inside space-y-1 text-slate-400">
          <li>Buka <a href="https://music.youtube.com" target="_blank" rel="noopener noreferrer" class="text-indigo-400 hover:underline inline-flex items-center gap-0.5">music.youtube.com <ExternalLink class="w-2.5 h-2.5" /></a> di tab browser yang sudah login.</li>
          <li>Tekan <kbd class="px-1 py-0.5 rounded bg-white/10 text-white font-mono text-[10px]">F12</kbd> &gt; buka tab <strong>Console</strong>.</li>
          <li>Paste script berikut lalu tekan Enter: <code class="px-1 py-0.5 rounded bg-white/10 text-indigo-300 font-mono text-[10px]">copy(document.cookie)</code>.</li>
          <li>Cookie langsung tersalin ke clipboard, lalu paste ke kotak di atas.</li>
        </ol>
      </div>

      <!-- Action Buttons -->
      <div class="flex items-center justify-between gap-3 pt-2">
        {#if isLoggedIn}
          <button
            type="button"
            class="px-4 py-2.5 rounded-2xl bg-rose-500/20 hover:bg-rose-500/30 text-rose-300 border border-rose-500/30 text-xs font-semibold flex items-center gap-2 transition-all disabled:opacity-50"
            on:click={logout}
            disabled={isLoading}
          >
            <Trash2 class="w-4 h-4" />
            <span>Hapus Cookie</span>
          </button>
        {:else}
          <div></div>
        {/if}

        <div class="flex items-center gap-2">
          <button
            type="button"
            class="px-4 py-2.5 rounded-2xl glass-pill text-slate-400 hover:text-white text-xs font-medium transition-all"
            on:click={onClose}
          >
            Batal
          </button>
          <button
            type="button"
            class="px-5 py-2.5 rounded-2xl bg-gradient-to-r from-indigo-500 to-purple-600 hover:from-indigo-600 hover:to-purple-700 text-white text-xs font-semibold shadow-lg shadow-indigo-500/30 flex items-center gap-2 transition-all disabled:opacity-50"
            on:click={saveCookie}
            disabled={isLoading || !cookieInput.trim()}
          >
            {#if isLoading}
              <Loader2 class="w-4 h-4 animate-spin" />
              <span>Menyimpan...</span>
            {:else}
              <LogIn class="w-4 h-4" />
              <span>Hubungkan Cookie</span>
            {/if}
          </button>
        </div>
      </div>
    </div>
  </div>
{/if}
