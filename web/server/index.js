const express = require('express');
const cookieParser = require('cookie-parser');
const cors = require('cors');
const http = require('http');
const path = require('path');
const fs = require('fs');

const { db, schema } = require('./db');
const { eq, gt, and } = require('drizzle-orm');
const { loadEnv, updateEnv } = require('./services/env');
const { hashPassword, verifyPassword, createSession, validateSession, destroySession } = require('./services/auth');
const { getSetting, setSetting, isInitialized, getPlatformName } = require('./services/settings');
const { listRelays, getActiveRelay, addRelay, setActiveRelay, toggleRelay, deleteRelay } = require('./services/relays');
const { deployCloudflareRelay, testRelayHealth } = require('./services/cloudflare');

loadEnv();

const app = express();
const PORT = process.env.PORT || 7554;
const GO_BACKEND_PORT = process.env.GO_BACKEND_PORT || 7555;

app.use(cors({
  origin: true,
  credentials: true,
}));
app.use(express.json());
app.use(cookieParser());

// Helper to notify Go backend of active relay URL
async function syncRelayWithGoBackend(relayUrl) {
  try {
    const payload = JSON.stringify({ url: relayUrl || '' });
    const req = http.request({
      hostname: '127.0.0.1',
      port: GO_BACKEND_PORT,
      path: '/api/internal/set-relay',
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Content-Length': Buffer.byteLength(payload),
      },
    }, (res) => {
      // response received
    });
    req.on('error', (err) => {
      // Go backend may not be up yet, ignore
    });
    req.write(payload);
    req.end();
  } catch (_) {}
}

// Authentication middleware for protected endpoints
async function requireAuth(req, res, next) {
  const token = req.cookies?.convx_session || req.headers['authorization']?.replace(/^Bearer\s+/i, '') || req.query?.token;
  if (!token) {
    return res.status(401).json({ error: 'Unauthorized: Login required' });
  }

  const authData = await validateSession(token);
  if (!authData) {
    res.clearCookie('convx_session', { path: '/' });
    return res.status(401).json({ error: 'Unauthorized: Invalid or expired session' });
  }

  req.user = authData.user;
  req.session = authData.session;
  next();
}

// Optional Auth (attaches req.user if present)
async function optionalAuth(req, res, next) {
  const token = req.cookies?.convx_session || req.headers['authorization']?.replace(/^Bearer\s+/i, '');
  if (token) {
    const authData = await validateSession(token);
    if (authData) {
      req.user = authData.user;
      req.session = authData.session;
    }
  }
  next();
}

// --- SETUP & ONBOARDING ROUTES ---

// GET /api/setup/status
app.get('/api/setup/status', (req, res) => {
  const initialized = isInitialized();
  const platformName = getPlatformName();
  const activeRelay = getActiveRelay();

  res.json({
    isInitialized: initialized,
    platformName,
    hasRelay: !!activeRelay,
  });
});

// POST /api/setup/init
app.post('/api/setup/init', async (req, res) => {
  try {
    if (isInitialized()) {
      return res.status(400).json({ error: 'System is already initialized' });
    }

    const {
      platformName = 'Convx Music',
      username,
      name,
      password,
      cfAccountId,
      cfApiToken,
      cfProjectName = 'convx-relay',
    } = req.body;

    if (!username || !username.trim()) {
      return res.status(400).json({ error: 'Username is required' });
    }
    if (!password || password.length < 4) {
      return res.status(400).json({ error: 'Password must be at least 4 characters' });
    }

    const cleanUsername = username.trim();
    const cleanName = (name && name.trim()) ? name.trim() : cleanUsername;
    const cleanPlatform = platformName.trim() || 'Convx Music';

    // 1. Create Admin User
    const passwordHash = await hashPassword(password);
    const userInsert = db.insert(schema.users).values({
      username: cleanUsername,
      name: cleanName,
      passwordHash,
      createdAt: Date.now(),
    }).run();

    const userId = Number(userInsert.lastInsertRowid);

    // 2. Save Platform Name
    setSetting('platform_name', cleanPlatform);

    // 3. Handle Cloudflare Relay if credentials provided
    let deployedRelayUrl = null;
    let cfDeployError = null;

    if (cfAccountId && cfApiToken) {
      const cleanCfAccount = cfAccountId.trim();
      const cleanCfToken = cfApiToken.trim();

      // Save to .env
      updateEnv({
        CF_ACCOUNT_ID: cleanCfAccount,
        CF_API_TOKEN: cleanCfToken,
        PLATFORM_NAME: cleanPlatform,
      });

      setSetting('cf_account_id', cleanCfAccount);

      try {
        console.log('[Setup] Deploying Cloudflare Worker Relay...');
        const deployResult = await deployCloudflareRelay(cleanCfAccount, cleanCfToken, cfProjectName);
        deployedRelayUrl = deployResult.deployUrl;

        // Save relay in DB
        addRelay(deployResult.projectName, deployedRelayUrl, 'cloudflare');
        updateEnv({ CF_WORKER_URL: deployedRelayUrl });
        setSetting('cf_worker_url', deployedRelayUrl);
        await syncRelayWithGoBackend(deployedRelayUrl);
        console.log('[Setup] Cloudflare Relay deployed successfully:', deployedRelayUrl);
      } catch (deployErr) {
        console.error('[Setup] Cloudflare Relay deployment warning:', deployErr.message);
        cfDeployError = deployErr.message;
      }
    } else {
      updateEnv({ PLATFORM_NAME: cleanPlatform });
    }

    // 4. Mark setup as initialized
    setSetting('is_initialized', 'true');

    // 5. Create session and set cookie
    const { token, expiresAt } = await createSession(userId);
    res.cookie('convx_session', token, {
      httpOnly: true,
      secure: false, // set to true if behind HTTPS proxy
      sameSite: 'lax',
      maxAge: 30 * 24 * 60 * 60 * 1000,
      path: '/',
    });

    res.json({
      status: 'ok',
      message: 'Onboarding completed successfully',
      user: { id: userId, username: cleanUsername, name: cleanName },
      platformName: cleanPlatform,
      relayUrl: deployedRelayUrl,
      relayWarning: cfDeployError,
    });
  } catch (err) {
    console.error('[Setup] Error:', err);
    res.status(500).json({ error: err.message || 'Failed to complete setup' });
  }
});

// --- AUTHENTICATION ROUTES ---

// POST /api/auth/login
app.post('/api/auth/login', async (req, res) => {
  try {
    const { username, password } = req.body;
    if (!username || !password) {
      return res.status(400).json({ error: 'Username and password are required' });
    }

    const cleanUsername = username.trim();
    const userRecords = db
      .select()
      .from(schema.users)
      .where(eq(schema.users.username, cleanUsername))
      .all();

    if (!userRecords || userRecords.length === 0) {
      return res.status(401).json({ error: 'Invalid username or password' });
    }

    const user = userRecords[0];
    const match = await verifyPassword(password, user.passwordHash);
    if (!match) {
      return res.status(401).json({ error: 'Invalid username or password' });
    }

    const { token } = await createSession(user.id);
    res.cookie('convx_session', token, {
      httpOnly: true,
      secure: false,
      sameSite: 'lax',
      maxAge: 30 * 24 * 60 * 60 * 1000,
      path: '/',
    });

    res.json({
      status: 'ok',
      user: { id: user.id, username: user.username, name: user.name || user.username },
      platformName: getPlatformName(),
    });
  } catch (err) {
    console.error('[Auth Login] Error:', err);
    res.status(500).json({ error: 'Internal server error' });
  }
});

// POST /api/auth/logout
app.post('/api/auth/logout', async (req, res) => {
  const token = req.cookies?.convx_session;
  if (token) {
    await destroySession(token);
  }
  res.clearCookie('convx_session', { path: '/' });
  res.json({ status: 'ok', message: 'Logged out successfully' });
});

// GET /api/auth/me
app.get('/api/auth/me', optionalAuth, (req, res) => {
  const initialized = isInitialized();
  const platformName = getPlatformName();
  const activeRelay = getActiveRelay();

  if (!initialized) {
    return res.json({
      isLoggedIn: false,
      isInitialized: false,
      platformName,
      activeRelay: null,
    });
  }

  if (!req.user) {
    return res.json({
      isLoggedIn: false,
      isInitialized: true,
      platformName,
      activeRelay,
    });
  }

  res.json({
    isLoggedIn: true,
    isInitialized: true,
    user: req.user,
    platformName,
    activeRelay,
  });
});

// --- SETTINGS & RELAYS ROUTES ---

// GET /api/settings
app.get('/api/settings', requireAuth, (req, res) => {
  const platformName = getPlatformName();
  const cfAccountId = process.env.CF_ACCOUNT_ID || getSetting('cf_account_id', '');
  const activeRelay = getActiveRelay();
  const allRelays = listRelays();

  res.json({
    platformName,
    name: req.user.name || req.user.username || '',
    username: req.user.username,
    cfAccountId,
    hasApiToken: !!process.env.CF_API_TOKEN,
    activeRelay,
    relays: allRelays,
  });
});

// POST /api/settings
app.post('/api/settings', requireAuth, async (req, res) => {
  try {
    const { platformName, cfAccountId, cfApiToken, newPassword, name } = req.body;

    if (platformName && platformName.trim()) {
      const clean = platformName.trim();
      setSetting('platform_name', clean);
      updateEnv({ PLATFORM_NAME: clean });
    }

    let updatedName = req.user.name;
    if (name !== undefined) {
      const clean = name.trim();
      if (clean) {
        updatedName = clean;
        db.update(schema.users)
          .set({ name: clean })
          .where(eq(schema.users.id, req.user.id))
          .run();
      }
    }

    const envUpdates = {};
    if (cfAccountId) {
      envUpdates.CF_ACCOUNT_ID = cfAccountId.trim();
      setSetting('cf_account_id', cfAccountId.trim());
    }
    if (cfApiToken) {
      envUpdates.CF_API_TOKEN = cfApiToken.trim();
    }
    if (Object.keys(envUpdates).length > 0) {
      updateEnv(envUpdates);
    }

    if (newPassword && newPassword.length >= 4) {
      const newHash = await hashPassword(newPassword);
      db.update(schema.users)
        .set({ passwordHash: newHash })
        .where(eq(schema.users.id, req.user.id))
        .run();
    }

    res.json({
      status: 'ok',
      message: 'Settings updated successfully',
      platformName: getPlatformName(),
      user: {
        id: req.user.id,
        username: req.user.username,
        name: updatedName || req.user.username,
      },
    });
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

// GET /api/relays
app.get('/api/relays', requireAuth, (req, res) => {
  res.json({ relays: listRelays(), activeRelay: getActiveRelay() });
});

// POST /api/relays/deploy
app.post('/api/relays/deploy', requireAuth, async (req, res) => {
  try {
    const accountId = req.body.cfAccountId || process.env.CF_ACCOUNT_ID;
    const apiToken = req.body.cfApiToken || process.env.CF_API_TOKEN;
    const projectName = req.body.projectName || `convx-relay-${Date.now().toString(36)}`;

    if (!accountId || !apiToken) {
      return res.status(400).json({ error: 'Cloudflare Account ID and API Token are required' });
    }

    updateEnv({ CF_ACCOUNT_ID: accountId.trim(), CF_API_TOKEN: apiToken.trim() });

    console.log('[Relays] Deploying new Cloudflare Relay...');
    const result = await deployCloudflareRelay(accountId, apiToken, projectName);
    const newRelay = addRelay(result.projectName, result.deployUrl, 'cloudflare');

    updateEnv({ CF_WORKER_URL: result.deployUrl });
    setSetting('cf_worker_url', result.deployUrl);
    await syncRelayWithGoBackend(result.deployUrl);

    res.json({
      status: 'ok',
      message: 'Cloudflare Relay deployed successfully',
      relay: newRelay,
      deployUrl: result.deployUrl,
    });
  } catch (err) {
    console.error('[Relays Deploy] Error:', err);
    res.status(500).json({ error: err.message || 'Failed to deploy relay' });
  }
});

// POST /api/relays/toggle
app.post('/api/relays/toggle', requireAuth, async (req, res) => {
  const { id, isActive } = req.body;
  if (!id) return res.status(400).json({ error: 'Missing relay id' });

  toggleRelay(id, !!isActive);
  const active = getActiveRelay();
  await syncRelayWithGoBackend(active ? active.url : '');
  res.json({ status: 'ok', activeRelay: active });
});

// POST /api/relays/test
app.post('/api/relays/test', requireAuth, async (req, res) => {
  const { url } = req.body;
  const targetUrl = url || getActiveRelay()?.url;
  if (!targetUrl) {
    return res.status(400).json({ error: 'No relay URL specified' });
  }

  const result = await testRelayHealth(targetUrl);
  res.json(result);
});

// --- PROXY TO GO BACKEND (AUDIO STREAMING & YOUTUBE API) ---

function proxyToGo(req, res) {
  const options = {
    hostname: '127.0.0.1',
    port: GO_BACKEND_PORT,
    path: req.originalUrl || req.url,
    method: req.method,
    headers: {
      ...req.headers,
      host: `127.0.0.1:${GO_BACKEND_PORT}`,
    },
  };

  const proxyReq = http.request(options, (proxyRes) => {
    res.writeHead(proxyRes.statusCode, proxyRes.headers);
    proxyRes.pipe(res);
  });

  proxyReq.on('error', (err) => {
    console.error('[Gateway Proxy Error]', err.message);
    if (!res.headersSent) {
      res.status(502).json({
        error: 'Backend streaming service unavailable. Ensure Go backend is running.',
        details: err.message,
      });
    }
  });

  req.pipe(proxyReq);
}

// --- DEVICE TRACKING ---

const DEVICE_TIMEOUT_MS = 90_000; // 90 seconds = offline
const sseClients = new Map(); // userId -> Set<res>

function getActiveDevices(userId) {
  const cutoff = Date.now() - DEVICE_TIMEOUT_MS;
  return db.select().from(schema.devices)
    .where(and(eq(schema.devices.userId, userId), gt(schema.devices.lastHeartbeat, cutoff)))
    .all();
}

function broadcastDevices(userId) {
  const devices = getActiveDevices(userId);
  const clients = sseClients.get(userId);
  if (!clients || clients.size === 0) return;
  const data = JSON.stringify(devices);
  for (const client of clients) {
    try { client.write(`data: ${data}\n\n`); } catch (_) {}
  }
}

// POST /api/devices/heartbeat
app.post('/api/devices/heartbeat', requireAuth, (req, res) => {
  const { deviceId, name, deviceType, platform, browser, isPlaying, currentSong } = req.body;
  if (!deviceId || !name || !deviceType || !platform || !browser) {
    return res.status(400).json({ error: 'Missing required device fields' });
  }

  const now = Date.now();
  const userId = req.user.id;

  // Upsert: try update first, insert if not exists
  const existing = db.select().from(schema.devices).where(eq(schema.devices.id, deviceId)).all();
  if (existing.length > 0) {
    db.update(schema.devices)
      .set({
        name, deviceType, platform, browser,
        isPlaying: isPlaying ? 1 : 0,
        currentSong: currentSong ? JSON.stringify(currentSong) : null,
        lastHeartbeat: now,
      })
      .where(eq(schema.devices.id, deviceId))
      .run();
  } else {
    db.insert(schema.devices)
      .values({
        id: deviceId, userId, name, deviceType, platform, browser,
        isPlaying: isPlaying ? 1 : 0,
        currentSong: currentSong ? JSON.stringify(currentSong) : null,
        lastHeartbeat: now,
        createdAt: now,
      })
      .run();
  }

  // Cleanup stale devices
  const cutoff = Date.now() - DEVICE_TIMEOUT_MS * 2;
  db.delete(schema.devices).where(gt(cutoff, schema.devices.lastHeartbeat)).run();

  broadcastDevices(userId);
  res.json({ status: 'ok' });
});

// GET /api/devices/stream — SSE endpoint
app.get('/api/devices/stream', requireAuth, (req, res) => {
  const userId = req.user.id;

  res.writeHead(200, {
    'Content-Type': 'text/event-stream',
    'Cache-Control': 'no-cache',
    'Connection': 'keep-alive',
    'X-Accel-Buffering': 'no',
  });

  // Send initial device list
  const devices = getActiveDevices(userId);
  res.write(`data: ${JSON.stringify(devices)}\n\n`);

  // Register SSE client
  if (!sseClients.has(userId)) sseClients.set(userId, new Set());
  sseClients.get(userId).add(res);

  // Keepalive ping every 30s
  const keepalive = setInterval(() => {
    try { res.write(': keepalive\n\n'); } catch (_) { clearInterval(keepalive); }
  }, 30_000);

  req.on('close', () => {
    clearInterval(keepalive);
    const clients = sseClients.get(userId);
    if (clients) {
      clients.delete(res);
      if (clients.size === 0) sseClients.delete(userId);
    }
  });
});

// GET /api/devices — REST fallback
app.get('/api/devices', requireAuth, (req, res) => {
  const devices = getActiveDevices(req.user.id);
  res.json({ devices });
});

// Protected streaming and YouTube endpoints
app.all('/api/search', requireAuth, proxyToGo);
app.all('/api/lyrics', optionalAuth, proxyToGo);
app.all('/api/account/status', requireAuth, proxyToGo);
app.all('/api/account/cookie', requireAuth, proxyToGo);
app.all('/api/account/logout', requireAuth, proxyToGo);
app.use('/api/stream', optionalAuth, proxyToGo);
app.use('/api/proxy/audio', optionalAuth, proxyToGo);

// --- STATIC ASSETS & SPA ROUTING ---

const distDir = path.resolve(__dirname, '../backend/dist');
if (fs.existsSync(distDir)) {
  app.use(express.static(distDir));
  app.use((req, res, next) => {
    if (req.method === 'GET' && !req.path.startsWith('/api')) {
      return res.sendFile(path.join(distDir, 'index.html'));
    }
    next();
  });
}

// Global error handler
app.use((err, req, res, next) => {
  console.error('[Server Error]', err);
  res.status(500).json({ error: err.message || 'Internal Server Error' });
});

app.listen(PORT, '0.0.0.0', () => {
  console.log(`🚀 Convx Node Gateway + Drizzle ORM listening on http://0.0.0.0:${PORT}`);
  const active = getActiveRelay();
  if (active) {
    console.log(`🌐 Active Cloudflare Relay: ${active.url}`);
    syncRelayWithGoBackend(active.url);
  } else {
    console.log(`ℹ️  No active Cloudflare Relay (direct mode)`);
  }
});
