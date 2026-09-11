// Cloudflare Relay Worker Script
// Forwards YouTube requests with x-relay-target and x-relay-path headers
const RELAY_WORKER_CODE = `
export default {
  async fetch(request, env, ctx) {
    const target = request.headers.get("x-relay-target");
    const relayPath = request.headers.get("x-relay-path") || "/";

    if (!target) {
      return new Response(JSON.stringify({
        status: "ok",
        message: "Convx Cloudflare Relay Active",
        timestamp: Date.now()
      }), {
        status: 200,
        headers: { "content-type": "application/json" }
      });
    }

    const targetUrl = target.replace(/\\/$/, "") + relayPath;
    const newHeaders = new Headers(request.headers);
    newHeaders.delete("x-relay-target");
    newHeaders.delete("x-relay-path");
    newHeaders.delete("host");

    const newRequestInit = {
      method: request.method,
      headers: newHeaders,
    };

    if (request.method !== "GET" && request.method !== "HEAD") {
      newRequestInit.body = request.body;
      newRequestInit.duplex = "half";
    }

    try {
      const response = await fetch(targetUrl, newRequestInit);
      return new Response(response.body, {
        status: response.status,
        headers: response.headers,
      });
    } catch (error) {
      return new Response(JSON.stringify({
        error: error.message,
        relay: "convx-worker"
      }), {
        status: 502,
        headers: { "content-type": "application/json" }
      });
    }
  }
};
`;

/**
 * Deploys the relay worker script to Cloudflare
 * @param {string} accountId Cloudflare Account ID
 * @param {string} apiToken Cloudflare API Token (Workers Scripts:Edit)
 * @param {string} projectName Name of the worker project
 */
async function deployCloudflareRelay(accountId, apiToken, projectName = 'convx-relay') {
  if (!accountId || !apiToken) {
    throw new Error('Cloudflare Account ID and API Token are required');
  }

  const cleanAccountId = accountId.trim();
  const cleanApiToken = apiToken.trim();
  const cleanProjectName = projectName.trim().toLowerCase().replace(/[^a-z0-9_-]/g, '-');

  // 1. Upload Worker Script via Multipart FormData
  const workerScriptUrl = `https://api.cloudflare.com/client/v4/accounts/${cleanAccountId}/workers/scripts/${cleanProjectName}`;
  
  const formData = new FormData();
  formData.append(
    'index.js',
    new Blob([RELAY_WORKER_CODE], { type: 'application/javascript+module' }),
    'index.js'
  );
  formData.append(
    'metadata',
    new Blob(
      [
        JSON.stringify({
          main_module: 'index.js',
          compatibility_date: '2024-03-20',
          observability: { enabled: true }
        })
      ],
      { type: 'application/json' }
    ),
    'metadata.json'
  );

  const uploadRes = await fetch(workerScriptUrl, {
    method: 'PUT',
    headers: {
      Authorization: `Bearer ${cleanApiToken}`,
    },
    body: formData,
  });

  if (!uploadRes.ok) {
    const err = await uploadRes.json().catch(() => ({}));
    const message = err.errors?.[0]?.message || `Upload failed with HTTP ${uploadRes.status}`;
    throw new Error(`Cloudflare upload error: ${message}`);
  }

  // 2. Enable workers.dev subdomain for this script
  const enableSubdomainRes = await fetch(`${workerScriptUrl}/subdomain`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${cleanApiToken}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ enabled: true }),
  });

  if (!enableSubdomainRes.ok) {
    const err = await enableSubdomainRes.json().catch(() => ({}));
    console.warn('[Cloudflare] Subdomain enable warning:', err);
  }

  // 3. Get the workers.dev subdomain for the account to construct final URL
  let deployUrl = '';
  const subdomainRes = await fetch(
    `https://api.cloudflare.com/client/v4/accounts/${cleanAccountId}/workers/subdomain`,
    {
      method: 'GET',
      headers: {
        Authorization: `Bearer ${cleanApiToken}`,
        'Content-Type': 'application/json',
      },
    }
  );

  if (subdomainRes.ok) {
    const subdomainData = await subdomainRes.json();
    if (subdomainData.result && subdomainData.result.subdomain) {
      deployUrl = `https://${cleanProjectName}.${subdomainData.result.subdomain}.workers.dev`;
    }
  }

  if (!deployUrl) {
    throw new Error(
      'Worker deployed but failed to retrieve workers.dev subdomain. Please ensure a workers.dev subdomain is configured in your Cloudflare dashboard.'
    );
  }

  return {
    deployUrl,
    projectName: cleanProjectName,
  };
}

/**
 * Test healthcheck of a deployed relay URL
 */
async function testRelayHealth(relayUrl) {
  try {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), 8000);
    const res = await fetch(relayUrl, { signal: controller.signal });
    clearTimeout(timeoutId);
    if (!res.ok) {
      return { ok: false, status: res.status, error: `HTTP ${res.status}` };
    }
    const data = await res.json().catch(() => ({}));
    return { ok: true, status: res.status, data };
  } catch (err) {
    return { ok: false, error: err.message };
  }
}

module.exports = {
  deployCloudflareRelay,
  testRelayHealth,
  RELAY_WORKER_CODE,
};
