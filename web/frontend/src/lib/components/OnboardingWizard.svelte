<script>
  import {
    Sparkles,
    Shield,
    Server,
    Globe,
    Check,
    CheckCircle2,
    AlertCircle,
    ArrowRight,
    ArrowLeft,
    Eye,
    EyeOff,
    Loader2,
    Lock,
    User,
    Cloud,
    HelpCircle
  } from 'lucide-svelte';

  export let onComplete = () => {};

  let step = 1;
  const totalSteps = 3;

  // Step 1: Branding
  let platformName = 'Convx Music';

  // Step 2: Admin Account
  let username = 'admin';
  let password = '';
  let confirmPassword = '';
  let showPassword = false;

  // Step 3: Cloudflare Relay
  let enableRelay = true;
  let cfAccountId = '';
  let cfApiToken = '';
  let cfProjectName = 'convx-relay';
  let showCfToken = false;
  let showCfGuide = false;

  // Submission state
  let isSubmitting = false;
  let submitStatusMessage = '';
  let errorMessage = '';

  function nextStep() {
    errorMessage = '';
    if (step === 1) {
      if (!platformName.trim()) {
        errorMessage = 'Nama platform tidak boleh kosong';
        return;
      }
      step = 2;
    } else if (step === 2) {
      if (!username.trim()) {
        errorMessage = 'Username admin tidak boleh kosong';
        return;
      }
      if (!password || password.length < 4) {
        errorMessage = 'Password minimal 4 karakter';
        return;
      }
      if (password !== confirmPassword) {
        errorMessage = 'Konfirmasi password tidak cocok';
        return;
      }
      step = 3;
    }
  }

  function prevStep() {
    errorMessage = '';
    if (step > 1) {
      step -= 1;
    }
  }

  async function handleFinishSetup() {
    errorMessage = '';
    if (enableRelay && (!cfAccountId.trim() || !cfApiToken.trim())) {
      errorMessage = 'Masukkan Cloudflare Account ID & API Token, atau nonaktifkan relay untuk melewatinya.';
      return;
    }

    isSubmitting = true;
    submitStatusMessage = 'Menginisialisasi SQLite Database & Drizzle ORM...';

    try {
      if (enableRelay && cfAccountId.trim() && cfApiToken.trim()) {
        submitStatusMessage = 'Mendeploy Cloudflare Relay Worker (proses upload & subdomain)...';
      }

      const res = await fetch('/api/setup/init', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          platformName: platformName.trim(),
          username: username.trim(),
          password,
          cfAccountId: enableRelay ? cfAccountId.trim() : '',
          cfApiToken: enableRelay ? cfApiToken.trim() : '',
          cfProjectName: cfProjectName.trim() || 'convx-relay',
        }),
      });

      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error || 'Gagal menyelesaikan instalasi');
      }

      submitStatusMessage = 'Instalasi selesai! Menyiapkan antarmuka musik...';
      setTimeout(() => {
        onComplete(data);
      }, 1000);
    } catch (err) {
      errorMessage = err.message || 'Terjadi kesalahan sistem';
      isSubmitting = false;
    }
  }
</script>

<div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/85 backdrop-blur-2xl overflow-y-auto">
  <!-- Glowing Background Orbs -->
  <div class="fixed -top-40 -left-40 w-96 h-96 bg-red-600/20 rounded-full blur-3xl pointer-events-none"></div>
  <div class="fixed -bottom-40 -right-40 w-96 h-96 bg-pink-600/20 rounded-full blur-3xl pointer-events-none"></div>

  <div class="relative w-full max-w-xl bg-[#1c1c1e]/90 border border-white/10 rounded-3xl shadow-2xl overflow-hidden p-6 sm:p-8 text-white my-auto animate-fadeIn">
    <!-- Header -->
    <div class="flex items-center gap-3 mb-6">
      <div class="w-11 h-11 rounded-2xl bg-gradient-to-tr from-red-500 to-pink-500 flex items-center justify-center shadow-lg shadow-red-500/30">
        <Sparkles class="w-6 h-6 text-white" />
      </div>
      <div>
        <h1 class="text-xl sm:text-2xl font-bold tracking-tight">Selamat Datang di Convx</h1>
        <p class="text-xs sm:text-sm text-neutral-400">Panduan Pengaturan Awal & Keamanan Sistem</p>
      </div>
    </div>

    <!-- Stepper Dots -->
    <div class="flex items-center justify-between mb-8 px-2 relative">
      <div class="absolute left-6 right-6 top-1/2 -translate-y-1/2 h-[2px] bg-white/10 z-0"></div>
      <div 
        class="absolute left-6 top-1/2 -translate-y-1/2 h-[2px] bg-red-500 transition-all duration-300 z-0"
        style="width: {((step - 1) / (totalSteps - 1)) * 88}%;"
      ></div>

      <!-- Step 1 Dot -->
      <div class="relative z-10 flex flex-col items-center">
        <div class="w-8 h-8 rounded-full flex items-center justify-center text-xs font-semibold transition-all duration-200 {step >= 1 ? 'bg-red-500 text-white shadow-lg shadow-red-500/40 ring-4 ring-[#1c1c1e]' : 'bg-neutral-800 text-neutral-400'}">
          {#if step > 1}<Check class="w-4 h-4" />{:else}1{/if}
        </div>
        <span class="text-[11px] font-medium mt-1.5 {step === 1 ? 'text-white' : 'text-neutral-400'}">Platform</span>
      </div>

      <!-- Step 2 Dot -->
      <div class="relative z-10 flex flex-col items-center">
        <div class="w-8 h-8 rounded-full flex items-center justify-center text-xs font-semibold transition-all duration-200 {step >= 2 ? 'bg-red-500 text-white shadow-lg shadow-red-500/40 ring-4 ring-[#1c1c1e]' : 'bg-neutral-800 text-neutral-400'}">
          {#if step > 2}<Check class="w-4 h-4" />{:else}2{/if}
        </div>
        <span class="text-[11px] font-medium mt-1.5 {step === 2 ? 'text-white' : 'text-neutral-400'}">Akun Admin</span>
      </div>

      <!-- Step 3 Dot -->
      <div class="relative z-10 flex flex-col items-center">
        <div class="w-8 h-8 rounded-full flex items-center justify-center text-xs font-semibold transition-all duration-200 {step >= 3 ? 'bg-red-500 text-white shadow-lg shadow-red-500/40 ring-4 ring-[#1c1c1e]' : 'bg-neutral-800 text-neutral-400'}">
          3
        </div>
        <span class="text-[11px] font-medium mt-1.5 {step === 3 ? 'text-white' : 'text-neutral-400'}">Cloudflare Relay</span>
      </div>
    </div>

    <!-- Error Alert -->
    {#if errorMessage}
      <div class="mb-5 p-3.5 bg-red-500/15 border border-red-500/30 rounded-2xl flex items-center gap-3 text-red-200 text-sm animate-shake">
        <AlertCircle class="w-5 h-5 text-red-400 shrink-0" />
        <span>{errorMessage}</span>
      </div>
    {/if}

    <!-- Step Content -->
    {#if step === 1}
      <div class="space-y-4 animate-fadeIn">
        <div class="p-4 bg-white/[0.03] border border-white/[0.06] rounded-2xl">
          <label class="block text-xs font-medium text-neutral-300 uppercase tracking-wider mb-2">
            Nama Platform / Branding Musik
          </label>
          <input
            type="text"
            bind:value={platformName}
            placeholder="Contoh: Convx Music, MelodyCloud"
            class="w-full bg-black/40 border border-white/15 focus:border-red-500 rounded-xl px-4 py-3 text-white text-base focus:outline-none transition-all placeholder:text-neutral-500"
          />
          <p class="text-xs text-neutral-400 mt-2">
            Nama ini akan muncul pada header aplikasi, tab browser, dan halaman login.
          </p>
        </div>

        <div class="p-4 bg-white/[0.02] border border-white/[0.04] rounded-2xl flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-xl bg-white/5 flex items-center justify-center text-red-400">
              <Shield class="w-5 h-5" />
            </div>
            <div>
              <div class="text-sm font-medium">Database SQLite + Drizzle ORM</div>
              <div class="text-xs text-neutral-400">Terproteksi dari SQL Injection & tanpa konfigurasi server DB rumit</div>
            </div>
          </div>
          <CheckCircle2 class="w-5 h-5 text-emerald-400" />
        </div>
      </div>
    {:else if step === 2}
      <div class="space-y-4 animate-fadeIn">
        <div class="p-4 bg-white/[0.03] border border-white/[0.06] rounded-2xl space-y-4">
          <div>
            <label class="block text-xs font-medium text-neutral-300 uppercase tracking-wider mb-2">
              Username Admin
            </label>
            <div class="relative">
              <User class="w-5 h-5 absolute left-3.5 top-1/2 -translate-y-1/2 text-neutral-400" />
              <input
                type="text"
                bind:value={username}
                placeholder="admin"
                class="w-full bg-black/40 border border-white/15 focus:border-red-500 rounded-xl pl-11 pr-4 py-3 text-white text-base focus:outline-none transition-all placeholder:text-neutral-500"
              />
            </div>
          </div>

          <div>
            <label class="block text-xs font-medium text-neutral-300 uppercase tracking-wider mb-2">
              Password Admin
            </label>
            <div class="relative">
              <Lock class="w-5 h-5 absolute left-3.5 top-1/2 -translate-y-1/2 text-neutral-400" />
              <input
                type={showPassword ? 'text' : 'password'}
                bind:value={password}
                placeholder="Minimal 4 karakter"
                class="w-full bg-black/40 border border-white/15 focus:border-red-500 rounded-xl pl-11 pr-11 py-3 text-white text-base focus:outline-none transition-all placeholder:text-neutral-500"
              />
              <button
                type="button"
                on:click={() => (showPassword = !showPassword)}
                class="absolute right-3.5 top-1/2 -translate-y-1/2 text-neutral-400 hover:text-white"
              >
                {#if showPassword}<EyeOff class="w-4 h-4" />{:else}<Eye class="w-4 h-4" />{/if}
              </button>
            </div>
          </div>

          <div>
            <label class="block text-xs font-medium text-neutral-300 uppercase tracking-wider mb-2">
              Konfirmasi Password
            </label>
            <div class="relative">
              <Lock class="w-5 h-5 absolute left-3.5 top-1/2 -translate-y-1/2 text-neutral-400" />
              <input
                type={showPassword ? 'text' : 'password'}
                bind:value={confirmPassword}
                placeholder="Ulangi password"
                class="w-full bg-black/40 border border-white/15 focus:border-red-500 rounded-xl pl-11 pr-4 py-3 text-white text-base focus:outline-none transition-all placeholder:text-neutral-500"
              />
            </div>
          </div>
        </div>

        <p class="text-xs text-neutral-400 px-1">
          Password akan dienkripsi dengan standar industri bcrypt. Sesi login tersimpan dengan aman via HTTP-Only cookie.
        </p>
      </div>
    {:else if step === 3}
      <div class="space-y-4 animate-fadeIn">
        <!-- Toggle Relay -->
        <div class="p-4 bg-white/[0.03] border border-white/[0.06] rounded-2xl flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-xl bg-orange-500/10 flex items-center justify-center text-orange-400">
              <Cloud class="w-5 h-5" />
            </div>
            <div>
              <div class="text-sm font-semibold flex items-center gap-2">
                Cloudflare Worker Relay
                <span class="text-[10px] px-1.5 py-0.5 rounded bg-emerald-500/20 text-emerald-300 font-medium">Direkomendasikan</span>
              </div>
              <div class="text-xs text-neutral-400">Cegah IP VPS/Server diblokir oleh YouTube dengan relay edge Cloudflare</div>
            </div>
          </div>
          <button
            type="button"
            on:click={() => (enableRelay = !enableRelay)}
            class="w-12 h-6 rounded-full transition-colors relative {enableRelay ? 'bg-red-500' : 'bg-neutral-700'}"
          >
            <span
              class="absolute top-1 left-1 bg-white w-4 h-4 rounded-full transition-transform {enableRelay ? 'translate-x-6' : 'translate-x-0'}"
            ></span>
          </button>
        </div>

        {#if enableRelay}
          <div class="p-4 bg-white/[0.03] border border-white/[0.06] rounded-2xl space-y-3.5 animate-fadeIn">
            <div>
              <div class="flex items-center justify-between mb-1.5">
                <label class="block text-xs font-medium text-neutral-300 uppercase tracking-wider">
                  Cloudflare Account ID
                </label>
                <button
                  type="button"
                  on:click={() => (showCfGuide = !showCfGuide)}
                  class="text-[11px] text-red-400 hover:text-red-300 flex items-center gap-1"
                >
                  <HelpCircle class="w-3.5 h-3.5" />
                  Cara Mendapatkan
                </button>
              </div>
              <input
                type="text"
                bind:value={cfAccountId}
                placeholder="Contoh: a1b2c3d4e5f6..."
                class="w-full bg-black/40 border border-white/15 focus:border-red-500 rounded-xl px-4 py-2.5 text-white text-sm focus:outline-none transition-all placeholder:text-neutral-500 font-mono"
              />
            </div>

            <div>
              <label class="block text-xs font-medium text-neutral-300 uppercase tracking-wider mb-1.5">
                Cloudflare API Token
              </label>
              <div class="relative">
                <input
                  type={showCfToken ? 'text' : 'password'}
                  bind:value={cfApiToken}
                  placeholder="Izin: Workers Scripts (Edit)"
                  class="w-full bg-black/40 border border-white/15 focus:border-red-500 rounded-xl pl-4 pr-10 py-2.5 text-white text-sm focus:outline-none transition-all placeholder:text-neutral-500 font-mono"
                />
                <button
                  type="button"
                  on:click={() => (showCfToken = !showCfToken)}
                  class="absolute right-3 top-1/2 -translate-y-1/2 text-neutral-400 hover:text-white"
                >
                  {#if showCfToken}<EyeOff class="w-4 h-4" />{:else}<Eye class="w-4 h-4" />{/if}
                </button>
              </div>
            </div>

            <div>
              <label class="block text-xs font-medium text-neutral-300 uppercase tracking-wider mb-1.5">
                Nama Project Worker
              </label>
              <input
                type="text"
                bind:value={cfProjectName}
                placeholder="convx-relay"
                class="w-full bg-black/40 border border-white/15 focus:border-red-500 rounded-xl px-4 py-2.5 text-white text-sm focus:outline-none transition-all placeholder:text-neutral-500 font-mono"
              />
            </div>

            {#if showCfGuide}
              <div class="p-3 bg-neutral-900/80 border border-white/10 rounded-xl text-xs space-y-1.5 text-neutral-300 animate-fadeIn">
                <div class="font-semibold text-white">Panduan Singkat Cloudflare:</div>
                <p>1. Buka <b>dash.cloudflare.com</b> → Di sidebar kanan bawah terlihat <b>Account ID</b>.</p>
                <p>2. Buka <b>My Profile</b> → <b>API Tokens</b> → <b>Create Token</b>.</p>
                <p>3. Pilih template <b>Edit Cloudflare Workers</b> (atau custom token dengan hak akses <code>Account: Workers Scripts: Edit</code>).</p>
                <p>4. Salin Token ke kolom di atas. Kredensial akan disimpan aman di file <code>.env</code>.</p>
              </div>
            {/if}
          </div>
        {:else}
          <div class="p-4 bg-amber-500/10 border border-amber-500/20 rounded-2xl text-xs text-amber-200">
            Convx akan memanggil YouTube secara langsung tanpa relay. Anda tetap dapat mengaktifkan Cloudflare Relay kapan saja melalui menu Pengaturan.
          </div>
        {/if}
      </div>
    {/if}

    <!-- Action Buttons -->
    <div class="flex items-center justify-between mt-8 pt-4 border-t border-white/10">
      {#if step > 1 && !isSubmitting}
        <button
          type="button"
          on:click={prevStep}
          class="px-4 py-2.5 rounded-xl bg-white/5 hover:bg-white/10 text-neutral-300 hover:text-white text-sm font-medium flex items-center gap-1.5 transition-all"
        >
          <ArrowLeft class="w-4 h-4" />
          Kembali
        </button>
      {:else}
        <div></div>
      {/if}

      {#if step < totalSteps}
        <button
          type="button"
          on:click={nextStep}
          class="px-6 py-2.5 rounded-xl bg-gradient-to-r from-red-500 to-pink-500 hover:from-red-600 hover:to-pink-600 text-white text-sm font-semibold flex items-center gap-2 shadow-lg shadow-red-500/25 transition-all"
        >
          Lanjut
          <ArrowRight class="w-4 h-4" />
        </button>
      {:else}
        <button
          type="button"
          disabled={isSubmitting}
          on:click={handleFinishSetup}
          class="px-6 py-2.5 rounded-xl bg-gradient-to-r from-red-500 to-pink-500 hover:from-red-600 hover:to-pink-600 disabled:opacity-50 text-white text-sm font-semibold flex items-center gap-2 shadow-lg shadow-red-500/30 transition-all"
        >
          {#if isSubmitting}
            <Loader2 class="w-4 h-4 animate-spin" />
            <span>Memproses...</span>
          {:else}
            <span>Selesaikan & Mulai</span>
            <Check class="w-4 h-4" />
          {/if}
        </button>
      {/if}
    </div>

    <!-- Live Status Banner during submit -->
    {#if isSubmitting}
      <div class="mt-4 p-3 bg-red-500/10 border border-red-500/20 rounded-xl flex items-center gap-3 text-xs text-red-200 animate-fadeIn">
        <Loader2 class="w-4 h-4 animate-spin text-red-400 shrink-0" />
        <span>{submitStatusMessage}</span>
      </div>
    {/if}
  </div>
</div>

<style>
  @keyframes fadeIn {
    from { opacity: 0; transform: scale(0.98); }
    to { opacity: 1; transform: scale(1); }
  }
  .animate-fadeIn {
    animation: fadeIn 0.25s ease-out forwards;
  }
  @keyframes shake {
    0%, 100% { transform: translateX(0); }
    20%, 60% { transform: translateX(-4px); }
    40%, 80% { transform: translateX(4px); }
  }
  .animate-shake {
    animation: shake 0.3s ease-in-out;
  }
</style>
