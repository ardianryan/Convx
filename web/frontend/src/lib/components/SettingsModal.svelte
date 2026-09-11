<script>
  import {
    X,
    Server,
    Shield,
    Cloud,
    Check,
    Loader2,
    LogOut,
    Eye,
    EyeOff,
    Activity,
    Lock,
    Sparkles,
    CheckCircle2,
    AlertCircle,
    KeyRound,
    User,
    Info,
    ExternalLink,
    Heart,
    Music,
    Code2
  } from 'lucide-svelte';
  import { getApiUrl } from '../api.js';

  export let isOpen = false;
  export let onClose = () => {};
  export let onLogout = () => {};
  export let onOpenCookieModal = () => {};
  export let currentPlatformName = 'Convx Music';
  export let onPlatformNameChange = (name) => {};
  export let onRelayChange = (relay) => {};
  export let currentUserName = 'Ryan Ardian';
  export let onUserNameChange = (name) => {};

  let activeTab = 'relay'; // 'relay' | 'general' | 'security'

  // Settings data
  let platformName = currentPlatformName;
  let userName = currentUserName;
  let cfAccountId = '';
  let cfApiToken = '';
  let showToken = false;
  let activeRelay = null;
  let relaysList = [];

  // Security data
  let newPassword = '';
  let confirmPassword = '';
  let showPassword = false;

  // Status & Feedback
  let isLoading = false;
  let isDeploying = false;
  let isTesting = false;
  let testResult = null;
  let message = '';
  let error = '';

  function getInitials(name) {
    if (!name) return 'U';
    const parts = name.trim().split(/\s+/);
    if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
    return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
  }

  $: userInitials = getInitials(userName);

  $: if (isOpen) {
    loadSettings();
  }

  async function loadSettings() {
    isLoading = true;
    error = '';
    try {
      const res = await fetch(getApiUrl('/api/settings'));
      if (res.ok) {
        const data = await res.json();
        platformName = data.platformName || currentPlatformName;
        if (data.name) {
          userName = data.name;
          onUserNameChange(data.name);
        }
        cfAccountId = data.cfAccountId || '';
        activeRelay = data.activeRelay;
        onRelayChange(activeRelay);
        relaysList = data.relays || [];
      }
    } catch (err) {
      console.error(err);
    } finally {
      isLoading = false;
    }
  }

  async function saveProfileName() {
    error = '';
    message = '';
    if (!userName || !userName.trim()) {
      error = 'Nama profil tidak boleh kosong';
      return;
    }
    try {
      const res = await fetch(getApiUrl('/api/settings'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: userName.trim() }),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error);
      onUserNameChange(userName.trim());
      message = 'Nama profil berhasil diperbarui';
    } catch (err) {
      error = err.message;
    }
  }

  async function saveGeneralSettings() {
    error = '';
    message = '';
    try {
      const res = await fetch(getApiUrl('/api/settings'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          platformName: platformName.trim(),
        }),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error);
      onPlatformNameChange(platformName.trim());
      message = 'Nama platform berhasil diperbarui';
    } catch (err) {
      error = err.message;
    }
  }

  async function savePassword() {
    error = '';
    message = '';
    if (!newPassword || newPassword.length < 4) {
      error = 'Password minimal 4 karakter';
      return;
    }
    if (newPassword !== confirmPassword) {
      error = 'Konfirmasi password tidak cocok';
      return;
    }

    try {
      const res = await fetch(getApiUrl('/api/settings'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ newPassword }),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error);
      message = 'Password berhasil diubah';
      newPassword = '';
      confirmPassword = '';
    } catch (err) {
      error = err.message;
    }
  }

  async function deployRelay() {
    error = '';
    message = '';
    testResult = null;
    if (!cfAccountId.trim() || !cfApiToken.trim()) {
      error = 'Masukkan Account ID dan API Token Cloudflare';
      return;
    }

    isDeploying = true;
    try {
      const res = await fetch(getApiUrl('/api/relays/deploy'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          cfAccountId: cfAccountId.trim(),
          cfApiToken: cfApiToken.trim(),
          projectName: 'convx-relay',
        }),
      });

      const data = await res.json();
      if (!res.ok) throw new Error(data.error || 'Gagal deploy worker');

      activeRelay = data.relay;
      onRelayChange(activeRelay);
      message = 'Worker Cloudflare Relay berhasil di-deploy & diaktifkan!';
      await loadSettings();
    } catch (err) {
      error = err.message;
    } finally {
      isDeploying = false;
    }
  }

  async function testRelay() {
    if (!activeRelay?.url) return;
    isTesting = true;
    testResult = null;
    try {
      const start = Date.now();
      const res = await fetch(getApiUrl('/api/relays/test'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ url: activeRelay.url }),
      });
      const latency = Date.now() - start;
      const data = await res.json();
      if (data.ok) {
        testResult = { success: true, latency };
      } else {
        testResult = { success: false, error: data.error };
      }
    } catch (err) {
      testResult = { success: false, error: err.message };
    } finally {
      isTesting = false;
    }
  }

  async function toggleRelayActive(id, currentState) {
    try {
      const res = await fetch(getApiUrl('/api/relays/toggle'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id, isActive: !currentState }),
      });
      if (res.ok) {
        const toggleData = await res.json();
        activeRelay = toggleData.activeRelay;
        onRelayChange(activeRelay);
        await loadSettings();
      }
    } catch (err) {
      console.error(err);
    }
  }

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
    <div class="w-full max-w-lg bg-[#1c1c1e] border border-white/10 rounded-3xl shadow-2xl overflow-hidden text-white flex flex-col max-h-[90vh] {isClosing ? 'anim-modal-out' : 'anim-modal-in'}">
      <!-- Header -->
      <div class="p-5 border-b border-white/10 flex items-center justify-between">
        <div class="flex items-center gap-2.5">
          <div class="w-9 h-9 rounded-xl bg-white/5 flex items-center justify-center text-red-400">
            <Sparkles class="w-5 h-5" />
          </div>
          <div>
            <h2 class="text-lg font-bold">Pengaturan Sistem</h2>
            <p class="text-xs text-neutral-400">Kelola relay, platform, dan autentikasi</p>
          </div>
        </div>
        <button
          on:click={handleClose}
          class="w-8 h-8 rounded-full bg-white/5 hover:bg-white/10 flex items-center justify-center text-neutral-400 hover:text-white transition-colors btn-pressable cursor-pointer"
        >
          <X class="w-4 h-4" />
        </button>
      </div>

      <!-- Navigation Tabs -->
      <div class="flex border-b border-white/10 px-5 bg-white/[0.02]">
        <button
          on:click={() => (activeTab = 'relay')}
          class="py-3 px-3 text-xs font-semibold border-b-2 flex items-center gap-2 transition-all {activeTab === 'relay' ? 'border-red-500 text-white' : 'border-transparent text-neutral-400 hover:text-neutral-200'}"
        >
          <Cloud class="w-4 h-4" />
          Cloudflare Relay
        </button>
        <button
          on:click={() => (activeTab = 'general')}
          class="py-3 px-3 text-xs font-semibold border-b-2 flex items-center gap-2 transition-all {activeTab === 'general' ? 'border-red-500 text-white' : 'border-transparent text-neutral-400 hover:text-neutral-200'}"
        >
          <Server class="w-4 h-4" />
          Branding Platform
        </button>
        <button
          on:click={() => (activeTab = 'security')}
          class="py-3 px-3 text-xs font-semibold border-b-2 flex items-center gap-2 transition-all {activeTab === 'security' ? 'border-red-500 text-white' : 'border-transparent text-neutral-400 hover:text-neutral-200'}"
        >
          <User class="w-4 h-4" />
          Profil & Akun
        </button>
        <button
          on:click={() => (activeTab = 'about')}
          class="py-3 px-3 text-xs font-semibold border-b-2 flex items-center gap-2 transition-all {activeTab === 'about' ? 'border-red-500 text-white' : 'border-transparent text-neutral-400 hover:text-neutral-200'}"
        >
          <Info class="w-4 h-4" />
          Tentang Aplikasi
        </button>
      </div>

      <!-- Feedback alerts -->
      {#if message}
        <div class="mx-5 mt-4 p-3 bg-emerald-500/15 border border-emerald-500/30 rounded-xl text-emerald-200 text-xs flex items-center gap-2">
          <CheckCircle2 class="w-4 h-4 text-emerald-400 shrink-0" />
          <span>{message}</span>
        </div>
      {/if}
      {#if error}
        <div class="mx-5 mt-4 p-3 bg-red-500/15 border border-red-500/30 rounded-xl text-red-200 text-xs flex items-center gap-2">
          <AlertCircle class="w-4 h-4 text-red-400 shrink-0" />
          <span>{error}</span>
        </div>
      {/if}

      <!-- Body -->
      <div class="p-5 overflow-y-auto space-y-4 flex-1">
        {#if activeTab === 'relay'}
          <!-- Active Relay Status Card -->
          <div class="p-4 bg-white/[0.03] border border-white/[0.06] rounded-2xl">
            <div class="flex items-center justify-between mb-3">
              <div class="text-xs font-medium text-neutral-400 uppercase tracking-wider">Status Relay Saat Ini</div>
              {#if activeRelay && activeRelay.isActive}
                <span class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full bg-emerald-500/20 text-emerald-300 text-[11px] font-medium">
                  <span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
                  Relay Aktif
                </span>
              {:else}
                <span class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full bg-neutral-700 text-neutral-300 text-[11px] font-medium">
                  Direct Mode (Tanpa Relay)
                </span>
              {/if}
            </div>

            {#if activeRelay}
              <div class="font-mono text-xs text-white bg-black/40 p-2.5 rounded-xl border border-white/10 break-all mb-3">
                {activeRelay.url}
              </div>

              <div class="flex items-center gap-2">
                <button
                  type="button"
                  disabled={isTesting}
                  on:click={testRelay}
                  class="px-3 py-1.5 rounded-lg bg-white/10 hover:bg-white/15 text-xs font-medium flex items-center gap-1.5 transition-colors"
                >
                  {#if isTesting}
                    <Loader2 class="w-3.5 h-3.5 animate-spin" />
                    <span>Ping...</span>
                  {:else}
                    <Activity class="w-3.5 h-3.5 text-red-400" />
                    <span>Test Kesehatan</span>
                  {/if}
                </button>

                <button
                  type="button"
                  on:click={() => toggleRelayActive(activeRelay.id, activeRelay.isActive)}
                  class="px-3 py-1.5 rounded-lg bg-white/10 hover:bg-white/15 text-xs font-medium transition-colors"
                >
                  {activeRelay.isActive ? 'Nonaktifkan' : 'Aktifkan'}
                </button>
              </div>

              {#if testResult}
                <div class="mt-3 p-2.5 rounded-xl text-xs {testResult.success ? 'bg-emerald-500/15 text-emerald-300 border border-emerald-500/30' : 'bg-red-500/15 text-red-300 border border-red-500/30'}">
                  {#if testResult.success}
                    🟢 Relay merespons normal! Latensi: {testResult.latency}ms
                  {:else}
                    🔴 Gagal menghubungi relay: {testResult.error}
                  {/if}
                </div>
              {/if}
            {:else}
              <p class="text-xs text-neutral-400">
                Belum ada worker relay aktif. Deploy worker Cloudflare di bawah untuk mengamankan IP server.
              </p>
            {/if}
          </div>

          <!-- Deploy New Worker -->
          <div class="p-4 bg-white/[0.03] border border-white/[0.06] rounded-2xl space-y-3">
            <div class="text-xs font-medium text-neutral-300 uppercase tracking-wider">
              Deploy / Perbarui Worker Relay
            </div>

            <div>
              <label class="block text-[11px] text-neutral-400 mb-1">Cloudflare Account ID</label>
              <input
                type="text"
                bind:value={cfAccountId}
                placeholder="Account ID Cloudflare"
                class="w-full bg-black/40 border border-white/15 focus:border-red-500 rounded-xl px-3 py-2 text-xs font-mono text-white focus:outline-none"
              />
            </div>

            <div>
              <label class="block text-[11px] text-neutral-400 mb-1">Cloudflare API Token (Workers Edit)</label>
              <div class="relative">
                <input
                  type={showToken ? 'text' : 'password'}
                  bind:value={cfApiToken}
                  placeholder="API Token dari Cloudflare"
                  class="w-full bg-black/40 border border-white/15 focus:border-red-500 rounded-xl pl-3 pr-9 py-2 text-xs font-mono text-white focus:outline-none"
                />
                <button
                  type="button"
                  on:click={() => (showToken = !showToken)}
                  class="absolute right-3 top-1/2 -translate-y-1/2 text-neutral-400 hover:text-white"
                >
                  {#if showToken}<EyeOff class="w-3.5 h-3.5" />{:else}<Eye class="w-3.5 h-3.5" />{/if}
                </button>
              </div>
            </div>

            <button
              type="button"
              disabled={isDeploying}
              on:click={deployRelay}
              class="w-full py-2.5 rounded-xl bg-gradient-to-r from-red-500 to-pink-500 hover:from-red-600 hover:to-pink-600 disabled:opacity-50 text-white text-xs font-semibold flex items-center justify-center gap-2 shadow-lg shadow-red-500/25 transition-all"
            >
              {#if isDeploying}
                <Loader2 class="w-3.5 h-3.5 animate-spin" />
                <span>Mendeploy Worker ke Cloudflare...</span>
              {:else}
                <Cloud class="w-3.5 h-3.5" />
                <span>Deploy Worker Sekarang</span>
              {/if}
            </button>
          </div>
        {:else if activeTab === 'general'}
          <!-- General Branding -->
          <div class="p-4 bg-white/[0.03] border border-white/[0.06] rounded-2xl space-y-3">
            <div>
              <label class="block text-xs font-medium text-neutral-300 uppercase tracking-wider mb-2">
                Nama Platform
              </label>
              <input
                type="text"
                bind:value={platformName}
                placeholder="Convx Music"
                class="w-full bg-black/40 border border-white/15 focus:border-red-500 rounded-xl px-3.5 py-2.5 text-sm text-white focus:outline-none"
              />
            </div>

            <button
              type="button"
              on:click={saveGeneralSettings}
              class="px-4 py-2 rounded-xl bg-red-500 hover:bg-red-600 text-white text-xs font-semibold flex items-center gap-1.5 transition-colors"
            >
              <Check class="w-3.5 h-3.5" />
              Simpan Perubahan
            </button>
          </div>

          <!-- YouTube Cookie Integration -->
          <div class="p-4 bg-white/[0.03] border border-white/[0.06] rounded-2xl flex items-center justify-between">
            <div>
              <div class="text-sm font-semibold">YouTube Account & Cookie</div>
              <div class="text-xs text-neutral-400">Sinkronkan playlist pribadi & YouTube Music premium</div>
            </div>
            <button
              type="button"
              on:click={() => {
                onClose();
                onOpenCookieModal();
              }}
              class="px-3 py-1.5 rounded-xl bg-white/10 hover:bg-white/15 text-xs font-medium text-white transition-colors"
            >
              Atur Cookie
            </button>
          </div>
        {:else if activeTab === 'security'}
          <!-- Profile Name Update -->
          <div class="p-4 bg-white/[0.03] border border-white/[0.06] rounded-2xl space-y-3">
            <div class="flex items-center justify-between">
              <div class="text-xs font-medium text-neutral-300 uppercase tracking-wider">
                Profil Pengguna
              </div>
              <div class="w-8 h-8 rounded-full bg-gradient-to-tr from-rose-500 to-amber-500 flex items-center justify-center font-bold text-xs shadow-md text-white">
                {userInitials}
              </div>
            </div>

            <div>
              <label class="block text-[11px] text-neutral-400 mb-1">Nama Anda (Nama Tampilan)</label>
              <div class="relative">
                <input
                  type="text"
                  bind:value={userName}
                  placeholder="Ryan Ardian"
                  class="w-full bg-black/40 border border-white/15 focus:border-red-500 rounded-xl px-3 py-2 text-xs text-white focus:outline-none placeholder:text-neutral-500"
                />
              </div>
              <p class="text-[10px] text-neutral-400 mt-1">Inisial avatar ("{userInitials}") otomatis dibuat dari nama ini untuk navbar & sidebar.</p>
            </div>

            <button
              type="button"
              on:click={saveProfileName}
              class="px-4 py-2 rounded-xl bg-red-500 hover:bg-red-600 text-white text-xs font-semibold flex items-center gap-1.5 transition-colors cursor-pointer"
            >
              <Check class="w-3.5 h-3.5" />
              Simpan Nama Profil
            </button>
          </div>

          <!-- Password update -->
          <div class="p-4 bg-white/[0.03] border border-white/[0.06] rounded-2xl space-y-3">
            <div class="text-xs font-medium text-neutral-300 uppercase tracking-wider">
              Ubah Password Admin
            </div>

            <div>
              <label class="block text-[11px] text-neutral-400 mb-1">Password Baru</label>
              <div class="relative">
                <input
                  type={showPassword ? 'text' : 'password'}
                  bind:value={newPassword}
                  placeholder="Minimal 4 karakter"
                  class="w-full bg-black/40 border border-white/15 focus:border-red-500 rounded-xl pl-3 pr-9 py-2 text-xs text-white focus:outline-none"
                />
                <button
                  type="button"
                  on:click={() => (showPassword = !showPassword)}
                  class="absolute right-3 top-1/2 -translate-y-1/2 text-neutral-400 hover:text-white"
                >
                  {#if showPassword}<EyeOff class="w-3.5 h-3.5" />{:else}<Eye class="w-3.5 h-3.5" />{/if}
                </button>
              </div>
            </div>

            <div>
              <label class="block text-[11px] text-neutral-400 mb-1">Konfirmasi Password Baru</label>
              <input
                type={showPassword ? 'text' : 'password'}
                bind:value={confirmPassword}
                placeholder="Ulangi password"
                class="w-full bg-black/40 border border-white/15 focus:border-red-500 rounded-xl px-3 py-2 text-xs text-white focus:outline-none"
              />
            </div>

            <button
              type="button"
              on:click={savePassword}
              class="px-4 py-2 rounded-xl bg-red-500 hover:bg-red-600 text-white text-xs font-semibold flex items-center gap-1.5 transition-colors"
            >
              <KeyRound class="w-3.5 h-3.5" />
              Ganti Password
            </button>
          </div>
        {:else if activeTab === 'about'}
          <!-- TAB: TENTANG APLIKASI -->
          <div class="space-y-4 animate-in fade-in duration-200">
            <!-- Porting Explanation Card -->
            <div class="p-4 rounded-2xl bg-white/[0.03] border border-white/[0.08] space-y-2.5">
              <div class="flex items-center gap-2 text-[#fa2d48] font-bold text-xs uppercase tracking-wider">
                <Music class="w-4 h-4" />
                <span>Porting Resmi Versi Web</span>
              </div>
              <p class="text-xs text-neutral-200 leading-relaxed">
                Aplikasi ini merupakan hasil <b>porting Convx</b> (pemutar musik open-source Android berbasis <i>Liquid Glass</i> dan <i>Jetpack Compose</i>) ke dalam arsitektur <b>Web Fullstack Modern</b>.
              </p>
              <p class="text-[11px] text-neutral-400 leading-relaxed">
                Menghadirkan pengalaman streaming audio katalog lengkap YouTube Music, sinkronisasi lirik berjalan karaoke (LRCLIB), relay audio Cloudflare Workers anti-403, dan deteksi multi-perangkat langsung lewat browser tanpa instalasi APK.
              </p>
            </div>

            <!-- Repository Links -->
            <div class="space-y-2">
              <h4 class="text-[11px] font-semibold text-neutral-400 uppercase tracking-wider">Tautan Repositori Kode</h4>
              
              <a
                href="https://github.com/cosmictaserdev-creator/Convx"
                target="_blank"
                rel="noopener noreferrer"
                class="flex items-center justify-between p-3.5 rounded-2xl bg-white/[0.04] hover:bg-white/[0.08] border border-white/10 transition-all group btn-pressable"
              >
                <div class="flex items-center gap-3 min-w-0">
                <svg class="w-5 h-5 fill-current text-white shrink-0" viewBox="0 0 24 24">
                  <path fill-rule="evenodd" clip-rule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.53 1.032 1.53 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z" />
                </svg>
                <div class="min-w-0">
                    <p class="font-semibold text-xs text-white group-hover:text-rose-400 transition-colors truncate">
                      cosmictaserdev-creator / Convx
                    </p>
                    <p class="text-[10px] text-neutral-400 truncate">
                      Repositori Asli Android (Liquid Glass Music Player)
                    </p>
                  </div>
                </div>
                <ExternalLink class="w-3.5 h-3.5 text-neutral-400 group-hover:text-white shrink-0 ml-2" />
              </a>

              <a
                href="https://github.com/ardianryan/Convx-Desktop"
                target="_blank"
                rel="noopener noreferrer"
                class="flex items-center justify-between p-3.5 rounded-2xl bg-white/[0.04] hover:bg-white/[0.08] border border-white/10 transition-all group btn-pressable"
              >
                <div class="flex items-center gap-3 min-w-0">
                  <Code2 class="w-5 h-5 text-rose-500 shrink-0" />
                  <div class="min-w-0">
                    <p class="font-semibold text-xs text-white group-hover:text-rose-400 transition-colors truncate">
                      ardianryan / Convx-Desktop
                    </p>
                    <p class="text-[10px] text-neutral-400 truncate">
                      Repositori Porting Web (Svelte 5 + Golang + Cloudflare)
                    </p>
                  </div>
                </div>
                <ExternalLink class="w-3.5 h-3.5 text-neutral-400 group-hover:text-white shrink-0 ml-2" />
              </a>
            </div>

            <!-- Credits & Stack -->
            <div class="p-3 rounded-xl bg-white/[0.02] border border-white/[0.05] flex items-center justify-between text-[11px] text-neutral-400">
              <span>Stack: Svelte 5 • Go • Cloudflare • SQLite</span>
              <div class="flex items-center gap-1 text-rose-400 font-medium">
                <Heart class="w-3 h-3 fill-current" />
                <span>Open Source</span>
              </div>
            </div>
          </div>
        {/if}
      </div>

      <!-- Footer with Logout -->
      <div class="p-4 border-t border-white/10 bg-white/[0.02] flex items-center justify-between">
        <div class="text-[11px] text-neutral-500">
          Database: SQLite WAL • ORM: Drizzle
        </div>
        <button
          type="button"
          on:click={onLogout}
          class="px-3.5 py-1.5 rounded-xl bg-red-500/10 hover:bg-red-500/20 text-red-400 text-xs font-semibold flex items-center gap-1.5 transition-colors"
        >
          <LogOut class="w-3.5 h-3.5" />
          Keluar (Logout)
        </button>
      </div>
    </div>
  </div>
{/if}
