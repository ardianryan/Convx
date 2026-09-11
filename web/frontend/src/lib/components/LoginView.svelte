<script>
  import { Sparkles, Lock, User, Eye, EyeOff, Loader2, AlertCircle } from 'lucide-svelte';
  import { getApiUrl } from '../api.js';

  export let platformName = 'Convx Music';
  export let onSuccess = () => {};

  let username = '';
  let password = '';
  let showPassword = false;
  let isLoading = false;
  let errorMessage = '';

  async function handleLogin() {
    errorMessage = '';
    if (!username.trim() || !password) {
      errorMessage = 'Masukkan username dan password';
      return;
    }

    isLoading = true;
    try {
      const res = await fetch(getApiUrl('/api/auth/login'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          username: username.trim(),
          password,
        }),
      });

      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error || 'Login gagal, periksa username atau password');
      }

      onSuccess(data);
    } catch (err) {
      errorMessage = err.message;
    } finally {
      isLoading = false;
    }
  }
</script>

<div class="min-h-screen w-full flex items-center justify-center p-4 bg-gradient-to-b from-neutral-950 via-[#0d0d0e] to-black text-white relative overflow-hidden">
  <!-- Glowing Background Orbs -->
  <div class="absolute -top-32 -left-32 w-80 h-80 bg-red-600/20 rounded-full blur-3xl pointer-events-none"></div>
  <div class="absolute -bottom-32 -right-32 w-80 h-80 bg-pink-600/20 rounded-full blur-3xl pointer-events-none"></div>

  <div class="relative w-full max-w-md bg-[#1c1c1e]/80 border border-white/10 rounded-3xl shadow-2xl backdrop-blur-2xl p-6 sm:p-8">
    <!-- Logo & Title -->
    <div class="text-center mb-8">
      <div class="inline-flex w-14 h-14 rounded-2xl bg-gradient-to-tr from-red-500 to-pink-500 items-center justify-center shadow-lg shadow-red-500/30 mb-3">
        <Sparkles class="w-8 h-8 text-white" />
      </div>
      <h1 class="text-2xl font-bold tracking-tight">{platformName}</h1>
      <p class="text-xs text-neutral-400 mt-1">Masuk untuk mengakses perpustakaan musik Anda</p>
    </div>

    <!-- Error Alert -->
    {#if errorMessage}
      <div class="mb-5 p-3.5 bg-red-500/15 border border-red-500/30 rounded-2xl flex items-center gap-2.5 text-red-200 text-xs sm:text-sm animate-shake">
        <AlertCircle class="w-4 h-4 text-red-400 shrink-0" />
        <span>{errorMessage}</span>
      </div>
    {/if}

    <form on:submit|preventDefault={handleLogin} class="space-y-4">
      <div>
        <label class="block text-xs font-medium text-neutral-300 uppercase tracking-wider mb-2">
          Username
        </label>
        <div class="relative">
          <User class="w-5 h-5 absolute left-3.5 top-1/2 -translate-y-1/2 text-neutral-400" />
          <input
            type="text"
            bind:value={username}
            placeholder="Username admin"
            required
            class="w-full bg-black/40 border border-white/15 focus:border-red-500 rounded-xl pl-11 pr-4 py-3 text-white text-sm focus:outline-none transition-all placeholder:text-neutral-500"
          />
        </div>
      </div>

      <div>
        <label class="block text-xs font-medium text-neutral-300 uppercase tracking-wider mb-2">
          Password
        </label>
        <div class="relative">
          <Lock class="w-5 h-5 absolute left-3.5 top-1/2 -translate-y-1/2 text-neutral-400" />
          <input
            type={showPassword ? 'text' : 'password'}
            bind:value={password}
            placeholder="Password akun"
            required
            class="w-full bg-black/40 border border-white/15 focus:border-red-500 rounded-xl pl-11 pr-11 py-3 text-white text-sm focus:outline-none transition-all placeholder:text-neutral-500"
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

      <button
        type="submit"
        disabled={isLoading}
        class="w-full mt-2 py-3 rounded-xl bg-gradient-to-r from-red-500 to-pink-500 hover:from-red-600 hover:to-pink-600 disabled:opacity-50 text-white text-sm font-semibold flex items-center justify-center gap-2 shadow-lg shadow-red-500/30 transition-all cursor-pointer"
      >
        {#if isLoading}
          <Loader2 class="w-4 h-4 animate-spin" />
          <span>Memverifikasi...</span>
        {:else}
          <span>Masuk</span>
        {/if}
      </button>
    </form>

    <div class="mt-6 text-center text-[11px] text-neutral-500">
      Protected by SQLite & Drizzle ORM • Convx Web
    </div>
  </div>
</div>

<style>
  @keyframes shake {
    0%, 100% { transform: translateX(0); }
    20%, 60% { transform: translateX(-4px); }
    40%, 80% { transform: translateX(4px); }
  }
  .animate-shake {
    animation: shake 0.3s ease-in-out;
  }
</style>
